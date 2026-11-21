package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	formsv1 "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/registration"
	"github.com/studyverse/ems-backend/internal/registrationfiles"
	"github.com/studyverse/ems-backend/internal/registrationforms"
	"github.com/uptrace/bun"
)

type EventRegistrationsService struct {
	eventsv1connect.UnimplementedEventRegistrationsServiceHandler
	db                     *bun.DB
	publisher              *events.Publisher
	authz                  *authz.Enforcer
	internalReminderSecret string
	forms                  *registrationforms.Service
}

func (s *EventRegistrationsService) WithForms(forms *registrationforms.Service) *EventRegistrationsService {
	s.forms = forms
	return s
}

func NewEventRegistrationsService(db *bun.DB, publisher *events.Publisher, authzEnforcer *authz.Enforcer, internalSecrets ...string) *EventRegistrationsService {
	secret := ""
	if len(internalSecrets) > 0 {
		secret = internalSecrets[0]
	}
	return &EventRegistrationsService{db: db, publisher: publisher, authz: authzEnforcer, internalReminderSecret: secret}
}

func (s *EventRegistrationsService) RegisterForEvent(ctx context.Context, req *connect.Request[eventsv1.RegisterForEventRequest]) (*connect.Response[eventsv1.RegisterForEventResponse], error) {
	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	if localUserID != req.Msg.UserId {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("can only register yourself"))
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	event, err := registration.LockOpenEvent(ctx, tx, req.Msg.EventId, time.Now)
	if err != nil {
		return nil, err
	}
	if !event.PublishOnMap {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("event not found"))
	}
	var reg models.EventRegistration
	err = tx.NewSelect().Model(&reg).Where("event_id = ?", event.ID).Where("user_id = ?", localUserID).Scan(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	exists := err == nil
	if exists && reg.Status != models.RegistrationStatusCancelled {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("registration already exists"))
	}
	status := models.RegistrationStatusRegistered
	var prepared *registrationforms.Prepared
	if s.forms != nil {
		prepared, err = s.forms.Prepare(ctx, tx, event.ID, req.Msg.FormSubmission)
		if err != nil {
			return nil, err
		}
		status = models.RegistrationStatus(prepared.Status)
	}
	if status == models.RegistrationStatusRegistered {
		if err := registration.EnsureCapacity(ctx, tx, event); err != nil {
			return nil, err
		}
	}
	if exists {
		err = tx.NewUpdate().Model(&reg).Set("status = ?", status).
			Set("cancelled_at = NULL").Set("registered_at = NOW()").Set("updated_at = NOW()").WherePK().Returning("*").Scan(ctx)
	} else {
		reg = models.EventRegistration{EventID: event.ID, UserID: localUserID, Status: status, RegisteredAt: time.Now()}
		_, err = tx.NewInsert().Model(&reg).Returning("*").Exec(ctx)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save registration: %w", err))
	}
	if s.forms != nil {
		var contact struct{ FullName, Email string }
		if err := tx.NewRaw(`SELECT COALESCE(NULLIF(trim(concat_ws(' ',first_name,last_name)),''),username) AS full_name,email FROM users WHERE id=?`, localUserID).Scan(ctx, &contact); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if _, err := s.forms.Record(ctx, tx, prepared, registrationfiles.Owner{UserID: localUserID}, formsv1.RegistrationSource_REGISTRATION_SOURCE_MAP, reg.ID, &formsv1.ResponseContact{FullName: contact.FullName, Email: contact.Email}); err != nil {
			var connectErr *connect.Error
			if errors.As(err, &connectErr) {
				return nil, err
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if publisher := s.publisher.InTransaction(tx); publisher != nil && status == models.RegistrationStatusRegistered {
		if err := publisher.PublishRegistrationCreated(ctx, events.RegistrationCreatedPayload{
			RegistrationID: reg.ID, EventID: event.ID, EventTitle: event.Title, OrganizationID: event.OrganizationID,
			ParticipantUserID: localUserID, ActorID: localUserID,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&eventsv1.RegisterForEventResponse{Registration: dbEventRegistrationToProto(reg)}), nil
}

func (s *EventRegistrationsService) CancelRegistration(ctx context.Context, req *connect.Request[eventsv1.CancelRegistrationRequest]) (*connect.Response[eventsv1.CancelRegistrationResponse], error) {
	localUserID, trustedTelegramSender, err := s.cancellationCaller(ctx, req)
	if err != nil {
		return nil, err
	}
	if trustedTelegramSender && req.Msg.UserId != nil && req.Msg.GetUserId() != localUserID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("can only cancel the linked sender's registration"))
	}
	var reg models.EventRegistration
	query := s.db.NewSelect().Model(&reg)
	switch {
	case req.Msg.RegistrationId != nil:
		query = query.Where("id = ?", req.Msg.GetRegistrationId())
	case trustedTelegramSender && req.Msg.EventId != nil:
		query = query.Where("event_id = ?", req.Msg.GetEventId()).Where("user_id = ?", localUserID)
	case req.Msg.EventId != nil && req.Msg.UserId != nil:
		query = query.Where("event_id = ?", req.Msg.GetEventId()).Where("user_id = ?", req.Msg.GetUserId())
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provide registration_id or event_id and user_id"))
	}
	if err := query.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if localUserID != reg.UserID {
		if trustedTelegramSender {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("can only cancel the linked sender's registration"))
		}
		if _, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprint(reg.EventID), "manage_registrations"); err != nil {
			return nil, err
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	var event models.Event
	if err := tx.NewSelect().Model(&event).Where("id = ?", reg.EventID).For("UPDATE").Scan(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Re-read after the event lock: an organizer may have approved this pending
	// attempt while cancellation waited, changing which confirmed events apply.
	if err := tx.NewSelect().Model(&reg).WherePK().Scan(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result, err := tx.NewUpdate().Model((*models.EventRegistration)(nil)).Set("status = ?", models.RegistrationStatusCancelled).
		Set("cancelled_at = NOW()").Set("updated_at = NOW()").Where("id = ?", reg.ID).Where("status IN (?)", bun.In([]models.RegistrationStatus{models.RegistrationStatusRegistered, models.RegistrationStatusPendingApproval})).Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if affected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("active registration not found"))
	}
	if s.forms != nil {
		if err := s.forms.Cancel(ctx, tx, formsv1.RegistrationSource_REGISTRATION_SOURCE_MAP, reg.ID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if publisher := s.publisher.InTransaction(tx); publisher != nil && reg.Status == models.RegistrationStatusRegistered {
		if err := publisher.PublishRegistrationCancelled(ctx, events.RegistrationCancelledPayload{
			RegistrationID: reg.ID, EventID: event.ID, EventTitle: event.Title, OrganizationID: event.OrganizationID,
			ParticipantUserID: reg.UserID, ActorID: localUserID,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&eventsv1.CancelRegistrationResponse{Success: true}), nil
}

// The bot can cancel only for the Telegram sender verified by its webhook.
// This credential does not grant access to any other registration RPC.
func (s *EventRegistrationsService) cancellationCaller(ctx context.Context, req *connect.Request[eventsv1.CancelRegistrationRequest]) (int32, bool, error) {
	sender := req.Header().Get("X-Telegram-User-Id")
	if sender == "" && req.Header().Get(auth.InternalSecretHeader) == "" {
		id, err := s.authz.GetLocalUserID(ctx)
		return id, false, err
	}
	if s.internalReminderSecret == "" {
		return 0, false, connect.NewError(connect.CodeUnauthenticated, errors.New("internal cancellation authentication is unavailable"))
	}
	if err := auth.RequireInternalSecret(req, s.internalReminderSecret); err != nil {
		return 0, false, err
	}
	telegramID, err := strconv.ParseInt(sender, 10, 64)
	if err != nil || telegramID <= 0 {
		return 0, false, connect.NewError(connect.CodeInvalidArgument, errors.New("positive Telegram sender ID is required"))
	}
	var user models.User
	if err := s.db.NewSelect().Model(&user).Where("telegram_id = ?", telegramID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, connect.NewError(connect.CodeUnauthenticated, errors.New("Telegram account is not linked"))
		}
		return 0, false, connect.NewError(connect.CodeInternal, err)
	}
	return user.ID, true, nil
}

func (s *EventRegistrationsService) GetEventRegistrations(ctx context.Context, req *connect.Request[eventsv1.GetEventRegistrationsRequest]) (*connect.Response[eventsv1.GetEventRegistrationsResponse], error) {
	slog.Debug("GetEventRegistrations", "eventId", req.Msg.EventId)

	if _, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "manage_registrations"); err != nil {
		return nil, err
	}

	page := int32(1)
	if req.Msg.Page != nil {
		page = *req.Msg.Page
	}
	limit := int32(50)
	if req.Msg.Limit != nil {
		limit = *req.Msg.Limit
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var regs []models.EventRegistration
	err := s.db.NewSelect().Model(&regs).
		Where("event_id = ?", req.Msg.EventId).
		Order("registered_at").
		Limit(int(limit)).
		Offset(int((int64(page) - 1) * int64(limit))).
		Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	total, err := s.db.NewSelect().Model((*models.EventRegistration)(nil)).
		Where("event_id = ?", req.Msg.EventId).
		Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRegs := lo.Map(regs, func(reg models.EventRegistration, _ int) *eventsv1.EventRegistration {
		return dbEventRegistrationToProto(reg)
	})

	return connect.NewResponse(&eventsv1.GetEventRegistrationsResponse{
		Registrations: protoRegs,
		Total:         int32(total),
	}), nil
}

func (s *EventRegistrationsService) GetUserRegistrations(ctx context.Context, req *connect.Request[eventsv1.GetUserRegistrationsRequest]) (*connect.Response[eventsv1.GetUserRegistrationsResponse], error) {
	slog.Debug("GetUserRegistrations", "userId", req.Msg.UserId)
	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	if localUserID != req.Msg.UserId {
		if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
			return nil, err
		}
	}

	var regs []models.EventRegistration
	err = s.db.NewSelect().Model(&regs).
		Where("user_id = ?", req.Msg.UserId).
		Order("registered_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRegs := lo.Map(regs, func(reg models.EventRegistration, _ int) *eventsv1.EventRegistration {
		return dbEventRegistrationToProto(reg)
	})

	return connect.NewResponse(&eventsv1.GetUserRegistrationsResponse{
		Registrations: protoRegs,
	}), nil
}

func dbEventRegistrationToProto(r models.EventRegistration) *eventsv1.EventRegistration {
	reg := &eventsv1.EventRegistration{
		Id:           r.ID,
		EventId:      r.EventID,
		UserId:       r.UserID,
		Status:       dbRegistrationStatusToProto(r.Status),
		RegisteredAt: r.RegisteredAt.Format(time.RFC3339),
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
	}
	if r.CancelledAt != nil {
		cancelledAt := r.CancelledAt.Format(time.RFC3339)
		reg.CancelledAt = &cancelledAt
	}
	return reg
}

func dbRegistrationStatusToProto(status models.RegistrationStatus) eventsv1.RegistrationStatus {
	switch status {
	case models.RegistrationStatusRegistered:
		return eventsv1.RegistrationStatus_REGISTRATION_STATUS_REGISTERED
	case models.RegistrationStatusCancelled:
		return eventsv1.RegistrationStatus_REGISTRATION_STATUS_CANCELLED
	case models.RegistrationStatusWaitlist:
		return eventsv1.RegistrationStatus_REGISTRATION_STATUS_WAITLIST
	case models.RegistrationStatusPendingApproval:
		return eventsv1.RegistrationStatus_REGISTRATION_STATUS_PENDING_APPROVAL
	case models.RegistrationStatusRejected:
		return eventsv1.RegistrationStatus_REGISTRATION_STATUS_REJECTED
	default:
		return eventsv1.RegistrationStatus_REGISTRATION_STATUS_UNSPECIFIED
	}
}
