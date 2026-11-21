package sundayopen

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	formspb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	pbv1 "github.com/studyverse/ems-backend/gen/sundayopen/v1"
	"github.com/studyverse/ems-backend/gen/sundayopen/v1/sundayopenv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/registration"
	"github.com/studyverse/ems-backend/internal/registrationforms"
	"github.com/studyverse/ems-backend/internal/search"
	"github.com/uptrace/bun"
)

var _ sundayopenv1connect.SundayOpenServiceHandler = (*Service)(nil)

type Service struct {
	sundayopenv1connect.UnimplementedSundayOpenServiceHandler
	db                 *bun.DB
	authz              *authz.Enforcer
	cancellationSecret string
	forms              *registrationforms.Service
}

type sundayOpenEventRow struct {
	models.Event
	Slug                 string  `bun:"slug"`
	OrganizationTitle    *string `bun:"organization_title"`
	OrganizationImageUrl *string `bun:"organization_image_url"`
	OrganizationSlug     *string `bun:"organization_slug"`
	CategoryName         *string `bun:"category_name"`
}

func NewService(db *bun.DB, authzEnforcer *authz.Enforcer, cancellationSecret ...string) *Service {
	s := &Service{db: db, authz: authzEnforcer}
	if len(cancellationSecret) > 0 {
		s.cancellationSecret = cancellationSecret[0]
	}
	return s
}

func (s *Service) WithForms(forms *registrationforms.Service) *Service {
	s.forms = forms
	return s
}

func (s *Service) GetUpcomingEvents(ctx context.Context, req *connect.Request[pbv1.GetUpcomingEventsRequest]) (*connect.Response[pbv1.GetUpcomingEventsResponse], error) {
	response, err := s.ListPublicEvents(ctx, connect.NewRequest(&pbv1.ListPublicEventsRequest{Limit: req.Msg.Limit}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pbv1.GetUpcomingEventsResponse{Events: response.Msg.Events}), nil
}

func (s *Service) publicEventsQuery() *bun.SelectQuery {
	return s.db.NewSelect().
		TableExpr("events AS e").
		ColumnExpr("e.*").
		ColumnExpr("es.slug AS slug").
		ColumnExpr("o.title AS organization_title").
		ColumnExpr("o.image_url AS organization_image_url").
		ColumnExpr("os.slug AS organization_slug").
		ColumnExpr("c.name AS category_name").
		Join("INNER JOIN event_slugs AS es ON es.event_id = e.id AND es.is_current = true").
		Join("INNER JOIN organizations AS o ON o.id = e.organization_id").
		Join("LEFT JOIN organization_slugs AS os ON os.organization_id = o.id AND os.is_current = true").
		Join("LEFT JOIN categories AS c ON c.id = e.category_id").
		Where("e.publish_on_sunday_open = true").
		Where("e.status = ?", models.EventStatusPublished)
}

func (s *Service) ListPublicEvents(ctx context.Context, req *connect.Request[pbv1.ListPublicEventsRequest]) (*connect.Response[pbv1.ListPublicEventsResponse], error) {
	msg := req.Msg
	if msg.Period != pbv1.PublicEventPeriod_PUBLIC_EVENT_PERIOD_UNSPECIFIED && msg.Period != pbv1.PublicEventPeriod_PUBLIC_EVENT_PERIOD_UPCOMING && msg.Period != pbv1.PublicEventPeriod_PUBLIC_EVENT_PERIOD_PAST {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid event period"))
	}
	if msg.CategoryId != nil && msg.GetCategoryId() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("category_id must be greater than 0"))
	}
	page := msg.GetPage()
	if page <= 0 {
		page = 1
	}
	limit := msg.GetLimit()
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	query := s.publicEventsQuery()
	if msg.Period == pbv1.PublicEventPeriod_PUBLIC_EVENT_PERIOD_PAST {
		query = query.Where("(e.end_time <= NOW() OR e.completed_at IS NOT NULL)").OrderExpr("e.start_time DESC, e.id DESC")
	} else {
		query = query.Where("e.end_time > NOW()").Where("e.completed_at IS NULL").OrderExpr("e.start_time ASC, e.id ASC")
	}
	if msg.CategoryId != nil {
		query = query.Where("e.category_id = ?", msg.GetCategoryId())
	}
	if term := strings.Join(strings.Fields(msg.Search), " "); term != "" {
		// Escape SQL pattern characters so user search remains a literal substring.
		term = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(term)
		query = query.Where("(e.title ILIKE ? OR regexp_replace("+publicDescriptionSQL+", '[[:space:]]+', ' ', 'g') ILIKE ?)", "%"+term+"%", "%"+term+"%")
	}
	var rows []sundayOpenEventRow
	total, err := query.
		Limit(int(limit)).
		Offset(int((int64(page)-1)*int64(limit))).
		ScanAndCount(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list public sunday-open events: %w", err))
	}

	events := lo.Map(rows, func(row sundayOpenEventRow, _ int) *pbv1.SundayOpenEvent {
		return dbSundayOpenEventToProto(row)
	})
	if err := s.populatePublicHosts(ctx, events); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Category navigation covers the entire published organization catalogue,
	// independent of the current event page, period, search or selected category.
	var categoryRows []models.Category
	err = s.db.NewSelect().TableExpr("categories AS c").
		ColumnExpr("c.id, c.name").
		Join("INNER JOIN events AS e ON e.category_id = c.id").
		Where("e.publish_on_sunday_open = true").
		Where("e.status = ?", models.EventStatusPublished).
		Distinct().OrderExpr("c.name ASC, c.id ASC").Scan(ctx, &categoryRows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list public sunday-open categories: %w", err))
	}
	categories := lo.Map(categoryRows, func(category models.Category, _ int) *pbv1.PublicEventCategory {
		return &pbv1.PublicEventCategory{Id: category.ID, Name: category.Name}
	})
	return connect.NewResponse(&pbv1.ListPublicEventsResponse{Events: events, Total: int32(total), Categories: categories}), nil
}

func (s *Service) GetEventBySlug(ctx context.Context, req *connect.Request[pbv1.GetEventBySlugRequest]) (*connect.Response[pbv1.GetEventBySlugResponse], error) {
	slog.Debug("SundayOpen.GetEventBySlug", "slug", req.Msg.Slug)

	slug := strings.TrimSpace(req.Msg.Slug)
	if slug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("slug is required"))
	}

	var row sundayOpenEventRow
	err := s.publicEventsQuery().
		Where("es.slug = ?", slug).
		Limit(1).
		Scan(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("event not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get sunday-open event by slug: %w", err))
	}

	event := dbSundayOpenEventToProto(row)
	if err := s.populatePublicHosts(ctx, []*pbv1.SundayOpenEvent{event}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pbv1.GetEventBySlugResponse{Event: event}), nil
}

func (s *Service) RegisterExternally(ctx context.Context, req *connect.Request[pbv1.RegisterExternallyRequest]) (*connect.Response[pbv1.RegisterExternallyResponse], error) {
	msg := req.Msg
	slog.Debug("SundayOpen.RegisterExternally", "eventId", msg.EventId)

	fullName := strings.TrimSpace(msg.FullName)
	phone := strings.TrimSpace(msg.Phone)
	email := strings.TrimSpace(msg.Email)

	if msg.EventId <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event_id must be greater than 0"))
	}
	if fullName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("full_name is required"))
	}
	if phone == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("phone is required"))
	}
	if email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email is required"))
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	// Both account and legacy guests serialize on this same event-row lock.
	event, err := registration.LockOpenEvent(ctx, tx, msg.EventId, time.Now)
	if err != nil {
		return nil, err
	}
	if !event.PublishOnSundayOpen {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("event not found"))
	}
	if s.forms != nil {
		configured, err := s.forms.HasPublishedForm(ctx, tx, event.ID)
		if err != nil {
			return nil, err
		}
		if configured {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("sign in to your Sunday Open account and complete the event registration form"))
		}
	}
	var reg models.ExternalRegistration
	err = tx.NewSelect().Model(&reg).Where("event_id = ?", event.ID).Where("phone = ?", phone).Scan(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	exists := err == nil
	// A phone number is contact information, not an account capability. The
	// legacy flow must never restore an account-owned booking or mint its token.
	if exists && reg.SundayOpenAccountID != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("registration already exists"))
	}
	if exists && reg.Status != models.ExternalRegistrationStatusCancelled {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("registration already exists"))
	}
	if err := registration.EnsureCapacity(ctx, tx, event); err != nil {
		return nil, err
	}
	if exists {
		err = tx.NewUpdate().Model(&reg).Set("full_name = ?", fullName).Set("email = ?", email).
			Set("status = ?", models.ExternalRegistrationStatusRegistered).Set("cancelled_at = NULL").
			Set("registered_at = NOW()").Set("updated_at = NOW()").WherePK().Returning("*").Scan(ctx)
	} else {
		reg = models.ExternalRegistration{EventID: event.ID, FullName: fullName, Phone: &phone, Email: email, Status: models.ExternalRegistrationStatusRegistered, RegisteredAt: time.Now()}
		_, err = tx.NewInsert().Model(&reg).Returning("*").Exec(ctx)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save external registration: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	token := ""
	if cancellationConfigured(s.cancellationSecret) {
		token, err = signCancellationToken(s.cancellationSecret, reg.ID, reg.EventID, phone, reg.RegisteredAt)
		if err != nil {
			return nil, err
		}
	}
	return connect.NewResponse(&pbv1.RegisterExternallyResponse{
		Registration:      dbExternalRegistrationToProto(reg),
		CancellationToken: token,
	}), nil
}

func (s *Service) CancelExternalRegistration(ctx context.Context, req *connect.Request[pbv1.CancelExternalRegistrationRequest]) (*connect.Response[pbv1.CancelExternalRegistrationResponse], error) {
	msg := req.Msg
	var claim *cancellationClaim
	if msg.CancellationToken == "" {
		if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
			return nil, err
		}
	} else {
		verified, err := verifyCancellationToken(s.cancellationSecret, msg.CancellationToken, msg.EventId, strings.TrimSpace(msg.Phone))
		if err != nil {
			return nil, err
		}
		claim = &verified
	}
	slog.Debug("SundayOpen.CancelExternalRegistration", "eventId", msg.EventId)

	phone := strings.TrimSpace(msg.Phone)

	if msg.EventId <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event_id must be greater than 0"))
	}
	if phone == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("phone is required"))
	}

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Match account cancellation and review lock order: event, then registration.
		var eventID int32
		if err := tx.NewRaw(`SELECT id FROM events WHERE id=? FOR UPDATE`, msg.EventId).Scan(ctx, &eventID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return connect.NewError(connect.CodeNotFound, errors.New("event not found"))
			}
			return err
		}
		query := tx.NewUpdate().
			Model((*models.ExternalRegistration)(nil)).
			Set("status = ?", models.ExternalRegistrationStatusCancelled).
			Set("cancelled_at = NOW()").
			Set("updated_at = NOW()").
			Where("event_id = ?", msg.EventId).
			Where("phone = ?", phone).
			Where("status IN ('registered','pending_approval')")
		if claim != nil {
			query = query.Where("id = ?", claim.RegistrationID).
				Where("sunday_open_account_id IS NULL").
				Where("registered_at = ?", time.UnixMicro(claim.RegisteredAt))
		}
		var rows []models.ExternalRegistration
		err := query.Returning("*").Scan(ctx, &rows)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("cancel external registration: %w", err))
		}
		if len(rows) == 0 {
			return connect.NewError(connect.CodeNotFound, errors.New("registration not found or already cancelled"))
		}
		if s.forms != nil {
			for _, row := range rows {
				if err := s.forms.Cancel(ctx, tx, formspb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN, row.ID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pbv1.CancelExternalRegistrationResponse{Success: true}), nil
}

func (s *Service) GetExternalRegistrations(ctx context.Context, req *connect.Request[pbv1.GetExternalRegistrationsRequest]) (*connect.Response[pbv1.GetExternalRegistrationsResponse], error) {
	slog.Debug("SundayOpen.GetExternalRegistrations", "eventId", req.Msg.EventId)

	if req.Msg.EventId <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event_id must be greater than 0"))
	}
	if _, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprint(req.Msg.EventId), "manage_registrations"); err != nil {
		return nil, err
	}

	page := req.Msg.GetPage()
	if page <= 0 {
		page = 1
	}

	limit := req.Msg.GetLimit()
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var regs []models.ExternalRegistration
	err := s.db.NewSelect().
		Model(&regs).
		Where("event_id = ?", req.Msg.EventId).
		OrderExpr("registered_at DESC, id DESC").
		Limit(int(limit)).
		Offset(int((int64(page) - 1) * int64(limit))).
		Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list external registrations: %w", err))
	}

	total, err := s.db.NewSelect().
		Model((*models.ExternalRegistration)(nil)).
		Where("event_id = ?", req.Msg.EventId).
		Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count external registrations: %w", err))
	}

	protoRegs := lo.Map(regs, func(r models.ExternalRegistration, _ int) *pbv1.ExternalRegistration {
		return dbExternalRegistrationToProto(r)
	})

	return connect.NewResponse(&pbv1.GetExternalRegistrationsResponse{
		Registrations: protoRegs,
		Total:         int32(total),
	}), nil
}

func dbSundayOpenEventToProto(e sundayOpenEventRow) *pbv1.SundayOpenEvent {
	return &pbv1.SundayOpenEvent{
		Id:                   e.ID,
		Title:                e.Title,
		Description:          publicDescription(e.Description),
		ImageUrl:             e.ImageUrl,
		StartTime:            e.StartTime.Format(time.RFC3339),
		EndTime:              e.EndTime.Format(time.RFC3339),
		Slug:                 e.Slug,
		OrganizationTitle:    e.OrganizationTitle,
		OrganizationImageUrl: e.OrganizationImageUrl,
		OrganizationSlug:     e.OrganizationSlug,
		OrganizationId:       &e.OrganizationID,
		LocationText:         extractLocationText(e.StructuredLocation),
		CategoryId:           e.CategoryID,
		CategoryName:         e.CategoryName,
		RegistrationOpen:     e.Status == models.EventStatusPublished && e.CompletedAt == nil && e.EndTime.After(time.Now()),
	}
}

func extractLocationText(raw json.RawMessage) *string {
	if location, _ := search.ExtractLocationFromStructured(raw); location != "" {
		return &location
	}
	empty := ""
	if len(raw) == 0 {
		return &empty
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		slog.Warn("failed to parse sunday-open structured_location", "error", err)
		return &empty
	}

	value, ok := payload["text"]
	if !ok || value == nil {
		return &empty
	}

	if text, ok := value.(string); ok {
		result := text
		return &result
	}

	result := fmt.Sprint(value)
	return &result
}

func dbExternalRegistrationToProto(r models.ExternalRegistration) *pbv1.ExternalRegistration {
	reg := &pbv1.ExternalRegistration{
		Id:           r.ID,
		EventId:      r.EventID,
		FullName:     r.FullName,
		Phone:        lo.FromPtr(r.Phone),
		Email:        r.Email,
		Status:       dbExternalStatusToProto(r.Status),
		RegisteredAt: r.RegisteredAt.Format(time.RFC3339),
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
	}
	if r.CancelledAt != nil {
		cancelledAt := r.CancelledAt.Format(time.RFC3339)
		reg.CancelledAt = &cancelledAt
	}
	if r.CheckedInAt != nil {
		checkedInAt := r.CheckedInAt.Format(time.RFC3339)
		reg.CheckedInAt = &checkedInAt
	}
	return reg
}

func dbExternalStatusToProto(status models.ExternalRegistrationStatus) pbv1.ExternalRegistrationStatus {
	switch status {
	case models.ExternalRegistrationStatusRegistered:
		return pbv1.ExternalRegistrationStatus_EXTERNAL_REGISTRATION_STATUS_REGISTERED
	case models.ExternalRegistrationStatusCancelled:
		return pbv1.ExternalRegistrationStatus_EXTERNAL_REGISTRATION_STATUS_CANCELLED
	case models.ExternalRegistrationStatusPendingApproval:
		return pbv1.ExternalRegistrationStatus_EXTERNAL_REGISTRATION_STATUS_PENDING_APPROVAL
	case models.ExternalRegistrationStatusRejected:
		return pbv1.ExternalRegistrationStatus_EXTERNAL_REGISTRATION_STATUS_REJECTED
	default:
		return pbv1.ExternalRegistrationStatus_EXTERNAL_REGISTRATION_STATUS_UNSPECIFIED
	}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}
