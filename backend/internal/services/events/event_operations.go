package events

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func (s *Service) AddEventOrganizer(ctx context.Context, req *connect.Request[eventsv1.AddEventOrganizerRequest]) (*connect.Response[eventsv1.AddEventOrganizerResponse], error) {
	slog.Debug("AddEventOrganizer", "eventId", req.Msg.EventId, "userId", req.Msg.UserId)

	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}

	event := &models.Event{}
	err = s.db.NewSelect().Model(event).Where("id = ?", req.Msg.EventId).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("event not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get event: %w", err))
	}

	isValidOrg := req.Msg.OrganizationId == event.OrganizationID
	if !isValidOrg {
		var collaboratorOrgIDs []int32
		_ = s.db.NewSelect().Model((*models.EventCollaborator)(nil)).
			Column("organization_id").
			Where("event_id = ?", req.Msg.EventId).
			Scan(ctx, &collaboratorOrgIDs)
		for _, orgID := range collaboratorOrgIDs {
			if orgID == req.Msg.OrganizationId {
				isValidOrg = true
				break
			}
		}
	}
	if !isValidOrg {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization is not the host or a collaborator of this event"))
	}

	if !s.authz.CheckOrgPermission(ctx, localUserID, req.Msg.OrganizationId, "manage_settings") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you don't have permission to manage settings for this organization"))
	}

	organizer := &models.EventOrganizer{
		EventID:        req.Msg.EventId,
		UserID:         req.Msg.UserId,
		OrganizationID: req.Msg.OrganizationId,
		AddedBy:        &localUserID,
	}
	if req.Msg.Role != nil {
		organizer.Role = req.Msg.Role
	}

	_, err = s.db.NewInsert().Model(organizer).Returning("*").Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add organizer: %w", err))
	}

	user := &models.User{}
	_ = s.db.NewSelect().Model(user).Where("id = ?", organizer.UserID).Scan(ctx)
	org := &models.Organization{}
	_ = s.db.NewSelect().Model(org).Where("id = ?", organizer.OrganizationID).Scan(ctx)

	return connect.NewResponse(&eventsv1.AddEventOrganizerResponse{
		Organizer: dbEventOrganizerToProto(*organizer, user, org),
	}), nil
}

func dbEventOrganizerToProto(o models.EventOrganizer, user *models.User, org *models.Organization) *eventsv1.EventOrganizer {
	proto := &eventsv1.EventOrganizer{
		Id:             o.ID,
		EventId:        o.EventID,
		UserId:         o.UserID,
		OrganizationId: o.OrganizationID,
		AddedAt:        o.AddedAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:      o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      o.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if o.Role != nil {
		proto.Role = o.Role
	}
	if o.AddedBy != nil {
		proto.AddedBy = o.AddedBy
	}
	if user != nil {
		proto.Username = &user.Username
		proto.Email = &user.Email
		proto.FirstName = user.FirstName
		proto.LastName = user.LastName
	}
	if org != nil {
		proto.OrganizationTitle = &org.Title
	}
	return proto
}

func (s *Service) ListEventOrganizers(ctx context.Context, req *connect.Request[eventsv1.ListEventOrganizersRequest]) (*connect.Response[eventsv1.ListEventOrganizersResponse], error) {
	slog.Debug("ListEventOrganizers", "eventId", req.Msg.EventId)

	var rows []eventOrganizerRow
	err := s.db.NewSelect().
		TableExpr("event_organizers AS eo").
		ColumnExpr("eo.*").
		ColumnExpr("u.username").
		ColumnExpr("u.email").
		ColumnExpr("u.first_name").
		ColumnExpr("u.last_name").
		ColumnExpr("o.title AS organization_title").
		Join("INNER JOIN users u ON u.id = eo.user_id").
		Join("INNER JOIN organizations o ON o.id = eo.organization_id").
		Where("eo.event_id = ?", req.Msg.EventId).
		OrderExpr("eo.added_at").
		Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list organizers: %w", err))
	}

	organizers := lo.Map(rows, func(row eventOrganizerRow, _ int) *eventsv1.EventOrganizer {
		return dbEventOrganizerRowToProto(row)
	})

	return connect.NewResponse(&eventsv1.ListEventOrganizersResponse{
		Organizers: organizers,
	}), nil
}

func (s *Service) RemoveEventOrganizer(ctx context.Context, req *connect.Request[eventsv1.RemoveEventOrganizerRequest]) (*connect.Response[eventsv1.RemoveEventOrganizerResponse], error) {
	slog.Debug("RemoveEventOrganizer", "eventId", req.Msg.EventId, "userId", req.Msg.UserId)

	_, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "edit")
	if err != nil {
		return nil, err
	}

	_, err = s.db.NewDelete().Model((*models.EventOrganizer)(nil)).
		Where("event_id = ?", req.Msg.EventId).
		Where("user_id = ?", req.Msg.UserId).
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove organizer: %w", err))
	}

	return connect.NewResponse(&eventsv1.RemoveEventOrganizerResponse{Success: true}), nil
}

func (s *Service) UpdateEventOrganizer(ctx context.Context, req *connect.Request[eventsv1.UpdateEventOrganizerRequest]) (*connect.Response[eventsv1.UpdateEventOrganizerResponse], error) {
	slog.Debug("UpdateEventOrganizer", "id", req.Msg.Id)

	// Early auth gate: reject unauthenticated callers before any DB access
	if _, err := s.authz.GetLocalUserID(ctx); err != nil {
		return nil, err
	}

	organizer := &models.EventOrganizer{}
	if err := s.db.NewSelect().Model(organizer).Where("id = ?", req.Msg.Id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load organizer: %w", err))
	}

	if _, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", organizer.EventID), "edit"); err != nil {
		return nil, err
	}

	query := s.db.NewUpdate().Model(organizer).WherePK()
	if req.Msg.Role != nil {
		query = query.Set("role = ?", *req.Msg.Role)
	}
	query = query.Set("updated_at = NOW()")
	if _, err := query.Returning("*").Exec(ctx, organizer); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update organizer: %w", err))
	}

	user := &models.User{}
	_ = s.db.NewSelect().Model(user).Where("id = ?", organizer.UserID).Scan(ctx)
	org := &models.Organization{}
	_ = s.db.NewSelect().Model(org).Where("id = ?", organizer.OrganizationID).Scan(ctx)

	return connect.NewResponse(&eventsv1.UpdateEventOrganizerResponse{
		Organizer: dbEventOrganizerToProto(*organizer, user, org),
	}), nil
}

func (s *Service) AddEventCollaborator(ctx context.Context, req *connect.Request[eventsv1.AddEventCollaboratorRequest]) (*connect.Response[eventsv1.AddEventCollaboratorResponse], error) {
	slog.Debug("AddEventCollaborator", "eventId", req.Msg.EventId, "orgId", req.Msg.OrganizationId)

	localUserID, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "edit")
	if err != nil {
		return nil, err
	}

	collaborator := &models.EventCollaborator{
		EventID:        req.Msg.EventId,
		OrganizationID: req.Msg.OrganizationId,
		AddedBy:        &localUserID,
		AddedAt:        time.Now(),
	}
	_, err = s.db.NewInsert().Model(collaborator).Returning("*").Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add collaborator: %w", err))
	}

	if s.perms != nil {
		if err := s.perms.AddEventCollaborator(ctx, fmt.Sprintf("%d", req.Msg.EventId), fmt.Sprintf("%d", req.Msg.OrganizationId)); err != nil {
			slog.ErrorContext(ctx, "Failed to add SpiceDB collaborator relationship", "error", err)
		}
	}

	org := &models.Organization{}
	_ = s.db.NewSelect().Model(org).Where("id = ?", collaborator.OrganizationID).Scan(ctx)

	var orgSlug string
	slugModel := &models.OrganizationSlug{}
	_ = s.db.NewSelect().Model(slugModel).Where("organization_id = ? AND is_current = ?", collaborator.OrganizationID, true).Scan(ctx)
	orgSlug = slugModel.Slug

	return connect.NewResponse(&eventsv1.AddEventCollaboratorResponse{
		Collaborator: dbEventCollaboratorToProto(*collaborator, *org, orgSlug),
	}), nil
}

func dbEventCollaboratorToProto(c models.EventCollaborator, org models.Organization, orgSlug string) *eventsv1.EventCollaborator {
	proto := &eventsv1.EventCollaborator{
		Id:             c.ID,
		EventId:        c.EventID,
		OrganizationId: c.OrganizationID,
		AddedAt:        c.AddedAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:      c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if c.AddedBy != nil {
		proto.AddedBy = c.AddedBy
	}
	proto.OrganizationTitle = &org.Title
	if org.ImageUrl != nil {
		proto.OrganizationImageUrl = org.ImageUrl
	}
	if orgSlug != "" {
		proto.OrganizationSlug = &orgSlug
	}
	return proto
}

func (s *Service) ListEventCollaborators(ctx context.Context, req *connect.Request[eventsv1.ListEventCollaboratorsRequest]) (*connect.Response[eventsv1.ListEventCollaboratorsResponse], error) {
	slog.Debug("ListEventCollaborators", "eventId", req.Msg.EventId)

	var rows []eventCollaboratorRow
	err := s.db.NewSelect().
		TableExpr("event_collaborators AS ec").
		ColumnExpr("ec.*").
		ColumnExpr("o.title AS organization_title").
		ColumnExpr("o.image_url AS organization_image_url").
		ColumnExpr("os.slug AS organization_slug").
		Join("INNER JOIN organizations o ON o.id = ec.organization_id").
		Join("LEFT JOIN organization_slugs os ON os.organization_id = o.id AND os.is_current = true").
		Where("ec.event_id = ?", req.Msg.EventId).
		OrderExpr("ec.added_at").
		Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list collaborators: %w", err))
	}

	collaborators := lo.Map(rows, func(row eventCollaboratorRow, _ int) *eventsv1.EventCollaborator {
		return dbEventCollaboratorRowToProto(row)
	})

	return connect.NewResponse(&eventsv1.ListEventCollaboratorsResponse{
		Collaborators: collaborators,
	}), nil
}

func (s *Service) RemoveEventCollaborator(ctx context.Context, req *connect.Request[eventsv1.RemoveEventCollaboratorRequest]) (*connect.Response[eventsv1.RemoveEventCollaboratorResponse], error) {
	slog.Debug("RemoveEventCollaborator", "eventId", req.Msg.EventId, "orgId", req.Msg.OrganizationId)

	_, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "edit")
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	event := &models.Event{}
	if err := tx.NewSelect().Model(event).Where("id = ?", req.Msg.EventId).For("UPDATE").Scan(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to lock collaboration event: %w", err))
	}
	_, err = tx.NewDelete().Model((*models.EventCollaborator)(nil)).
		Where("event_id = ?", req.Msg.EventId).
		Where("organization_id = ?", req.Msg.OrganizationId).
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove collaborator: %w", err))
	}
	isSundayOpen, err := organizationIsSundayOpen(ctx, tx, req.Msg.OrganizationId)
	if err != nil {
		return nil, err
	}
	if isSundayOpen {
		// Removing Sunday Open ends its accepted consent. A later direct addition
		// cannot reuse an old invitation to publish on its behalf.
		if _, err := tx.NewUpdate().Model((*models.EventCollaborationInvite)(nil)).
			Set("status = ?", models.CollaborationInviteStatusRevoked).
			Set("updated_at = NOW()").
			Where("event_id = ? AND organization_id = ?", req.Msg.EventId, req.Msg.OrganizationId).
			Where("status = ?", models.CollaborationInviteStatusAccepted).Exec(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke removed collaboration consent: %w", err))
		}
		if err := syncCollaborationPublication(ctx, tx, event, req.Msg.OrganizationId); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if s.perms != nil {
		if err := s.perms.RemoveEventCollaborator(ctx, fmt.Sprintf("%d", req.Msg.EventId), fmt.Sprintf("%d", req.Msg.OrganizationId)); err != nil {
			slog.ErrorContext(ctx, "Failed to remove SpiceDB collaborator relationship", "error", err)
		}
	}

	return connect.NewResponse(&eventsv1.RemoveEventCollaboratorResponse{Success: true}), nil
}

func (s *Service) GetUserSubscribedEvents(ctx context.Context, req *connect.Request[eventsv1.GetUserSubscribedEventsRequest]) (*connect.Response[eventsv1.GetUserSubscribedEventsResponse], error) {
	slog.Debug("GetUserSubscribedEvents", "userId", req.Msg.UserId)
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
	err = s.db.NewSelect().Model(&regs).Where("user_id = ?", req.Msg.UserId).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if len(regs) == 0 {
		return connect.NewResponse(&eventsv1.GetUserSubscribedEventsResponse{
			Events: []*eventsv1.Event{},
		}), nil
	}

	eventIDs := lo.Map(regs, func(reg models.EventRegistration, _ int) int32 {
		return reg.EventID
	})

	var events []models.Event
	err = s.db.NewSelect().Model(&events).Where("id IN (?)", bun.In(eventIDs)).Where("publish_on_map = TRUE").Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	orgIDs := lo.Map(events, func(e models.Event, _ int) int32 {
		return e.OrganizationID
	})
	catIDs := lo.FilterMap(events, func(e models.Event, _ int) (int32, bool) {
		if e.CategoryID == nil {
			return 0, false
		}
		return *e.CategoryID, true
	})

	var orgs []models.Organization
	if len(orgIDs) > 0 {
		_ = s.db.NewSelect().Model(&orgs).Where("id IN (?)", bun.In(orgIDs)).Scan(ctx)
	}
	orgMap := make(map[int32]models.Organization, len(orgs))
	for _, o := range orgs {
		orgMap[o.ID] = o
	}

	var cats []models.Category
	if len(catIDs) > 0 {
		_ = s.db.NewSelect().Model(&cats).Where("id IN (?)", bun.In(catIDs)).Scan(ctx)
	}
	catMap := make(map[int32]models.Category, len(cats))
	for _, c := range cats {
		catMap[c.ID] = c
	}

	protoEvents := make([]*eventsv1.Event, 0, len(events))
	for _, event := range events {
		org := orgMap[event.OrganizationID]
		var category *models.Category
		if event.CategoryID != nil {
			if cat, ok := catMap[*event.CategoryID]; ok {
				category = &cat
			}
		}
		protoEvents = append(protoEvents, dbEventWithRelationsToProto(event, org, category))
	}

	if err := s.populateEventCounts(ctx, protoEvents); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.populateEventParticipants(ctx, protoEvents); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.GetUserSubscribedEventsResponse{
		Events: protoEvents,
	}), nil
}
