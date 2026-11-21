package events

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
)

func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Service) InviteOrganizationToCollaborate(ctx context.Context, req *connect.Request[eventsv1.InviteOrganizationToCollaborateRequest]) (*connect.Response[eventsv1.InviteOrganizationToCollaborateResponse], error) {
	slog.Debug("InviteOrganizationToCollaborate", "eventId", req.Msg.EventId, "orgId", req.Msg.OrganizationId)

	localUserID, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "edit")
	if err != nil {
		return nil, err
	}

	var collaboratorOrgIDs []int32
	_ = s.db.NewSelect().Model((*models.EventCollaborator)(nil)).
		Column("organization_id").
		Where("event_id = ?", req.Msg.EventId).
		Scan(ctx, &collaboratorOrgIDs)
	for _, orgID := range collaboratorOrgIDs {
		if orgID == req.Msg.OrganizationId {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("organization is already a collaborator"))
		}
	}

	exists, _ := s.db.NewSelect().Model((*models.EventCollaborationInvite)(nil)).
		Where("event_id = ?", req.Msg.EventId).
		Where("organization_id = ?", req.Msg.OrganizationId).
		Where("status = ?", models.CollaborationInviteStatusPending).
		Exists(ctx)
	if exists {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("pending invite already exists"))
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	token := generateToken()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	invite := &models.EventCollaborationInvite{
		EventID:         req.Msg.EventId,
		OrganizationID:  req.Msg.OrganizationId,
		InvitedByUserID: localUserID,
		Token:           token,
		ExpiresAt:       expiresAt,
		Status:          models.CollaborationInviteStatusPending,
		Message:         req.Msg.Message,
	}

	_, err = tx.NewInsert().Model(invite).Returning("*").Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create invite: %w", err))
	}

	event := &models.Event{}
	_ = tx.NewSelect().Model(event).Where("id = ?", req.Msg.EventId).Scan(ctx)

	if publisher != nil {
		if err := publisher.PublishCollaborationInvited(ctx, domainevents.CollaborationInvitedPayload{
			EventID:            event.ID,
			EventTitle:         event.Title,
			OrganizationID:     req.Msg.OrganizationId,
			HostOrganizationID: event.OrganizationID,
			InviteToken:        token,
			MapAppURL:          s.mapAppURL,
			ActorID:            localUserID,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.InviteOrganizationToCollaborateResponse{
		Invite: dbCollaborationInviteToProto(*invite),
	}), nil
}

func (s *Service) AcceptCollaborationInvite(ctx context.Context, req *connect.Request[eventsv1.AcceptCollaborationInviteRequest]) (*connect.Response[eventsv1.AcceptCollaborationInviteResponse], error) {
	slog.Debug("AcceptCollaborationInvite")

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	// Serialize publication and collaboration changes on the event before locking
	// its invitation, consistently with removal and revocation.
	event := &models.Event{}
	err = tx.NewSelect().Model(event).
		Where("id = (SELECT event_id FROM event_collaboration_invites WHERE token = ?)", req.Msg.Token).
		For("UPDATE").Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to lock collaboration event: %w", err))
	}

	var row inviteByTokenRow
	err = tx.NewSelect().
		TableExpr("event_collaboration_invites AS i").
		ColumnExpr("i.*").
		ColumnExpr("e.title AS event_title").
		ColumnExpr("es.slug AS event_slug").
		ColumnExpr("o.title AS organization_title").
		Join("INNER JOIN events e ON e.id = i.event_id").
		Join("LEFT JOIN event_slugs es ON es.event_id = e.id AND es.is_current = true").
		Join("INNER JOIN organizations o ON o.id = i.organization_id").
		Where("i.token = ?", req.Msg.Token).
		For("UPDATE OF i").
		Scan(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if row.Status != models.CollaborationInviteStatusPending {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("invite is not pending"))
	}
	if row.ExpiresAt.Before(time.Now()) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("invite has expired"))
	}

	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	if !s.authz.CheckOrgPermission(ctx, localUserID, row.OrganizationID, "manage_settings") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you don't have permission to manage settings for this organization"))
	}

	now := time.Now()
	_, err = tx.NewUpdate().Model((*models.EventCollaborationInvite)(nil)).
		Set("status = ?", models.CollaborationInviteStatusAccepted).
		Set("responded_at = ?", now).
		Set("responded_by_user_id = ?", localUserID).
		Set("updated_at = NOW()").
		Where("id = ?", row.ID).
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to accept invite: %w", err))
	}
	row.Status = models.CollaborationInviteStatusAccepted
	row.RespondedAt = &now
	row.RespondedByUserID = &localUserID

	collaborator := &models.EventCollaborator{
		EventID:        row.EventID,
		OrganizationID: row.OrganizationID,
		AddedBy:        &localUserID,
		AddedAt:        time.Now(),
	}
	_, err = tx.NewInsert().Model(collaborator).Returning("*").Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add collaborator: %w", err))
	}

	if err := syncCollaborationPublication(ctx, tx, event, row.OrganizationID); err != nil {
		return nil, err
	}
	if publisher != nil {
		if err := publisher.PublishCollaborationAccepted(ctx, domainevents.CollaborationAcceptedPayload{
			EventID:            row.EventID,
			EventTitle:         event.Title,
			OrganizationID:     row.OrganizationID,
			HostOrganizationID: event.OrganizationID,
			ActorID:            localUserID,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	org := &models.Organization{}
	_ = tx.NewSelect().Model(org).Where("id = ?", collaborator.OrganizationID).Scan(ctx)
	orgSlugModel := &models.OrganizationSlug{}
	_ = tx.NewSelect().Model(orgSlugModel).Where("organization_id = ? AND is_current = ?", collaborator.OrganizationID, true).Scan(ctx)

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if s.perms != nil {
		if err := s.perms.AddEventCollaborator(ctx, fmt.Sprintf("%d", row.EventID), fmt.Sprintf("%d", row.OrganizationID)); err != nil {
			slog.ErrorContext(ctx, "Failed to add SpiceDB collaborator relationship", "error", err)
		}
	}

	return connect.NewResponse(&eventsv1.AcceptCollaborationInviteResponse{
		Invite:       dbCollaborationInviteByTokenRowToProto(row),
		Collaborator: dbEventCollaboratorToProto(*collaborator, *org, orgSlugModel.Slug),
	}), nil
}

func (s *Service) DeclineCollaborationInvite(ctx context.Context, req *connect.Request[eventsv1.DeclineCollaborationInviteRequest]) (*connect.Response[eventsv1.DeclineCollaborationInviteResponse], error) {
	slog.Debug("DeclineCollaborationInvite")

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	var row inviteByTokenRow
	err = tx.NewSelect().
		TableExpr("event_collaboration_invites AS i").
		ColumnExpr("i.*").
		ColumnExpr("e.title AS event_title").
		ColumnExpr("es.slug AS event_slug").
		ColumnExpr("o.title AS organization_title").
		Join("INNER JOIN events e ON e.id = i.event_id").
		Join("LEFT JOIN event_slugs es ON es.event_id = e.id AND es.is_current = true").
		Join("INNER JOIN organizations o ON o.id = i.organization_id").
		Where("i.token = ?", req.Msg.Token).
		For("UPDATE OF i").
		Scan(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if row.Status != models.CollaborationInviteStatusPending {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("invite is not pending"))
	}

	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	if !s.authz.CheckOrgPermission(ctx, localUserID, row.OrganizationID, "manage_settings") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you don't have permission to manage settings for this organization"))
	}

	now := time.Now()
	_, err = tx.NewUpdate().Model((*models.EventCollaborationInvite)(nil)).
		Set("status = ?", models.CollaborationInviteStatusDeclined).
		Set("responded_at = ?", now).
		Set("responded_by_user_id = ?", localUserID).
		Set("updated_at = NOW()").
		Where("id = ?", row.ID).
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decline invite: %w", err))
	}
	row.Status = models.CollaborationInviteStatusDeclined
	row.RespondedAt = &now
	row.RespondedByUserID = &localUserID

	event := &models.Event{}
	_ = tx.NewSelect().Model(event).Where("id = ?", row.EventID).Scan(ctx)
	if publisher != nil {
		if err := publisher.PublishCollaborationDeclined(ctx, domainevents.CollaborationDeclinedPayload{
			EventID:            row.EventID,
			EventTitle:         event.Title,
			OrganizationID:     row.OrganizationID,
			HostOrganizationID: event.OrganizationID,
			ActorID:            localUserID,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.DeclineCollaborationInviteResponse{
		Invite: dbCollaborationInviteByTokenRowToProto(row),
	}), nil
}

func (s *Service) RevokeCollaborationInvite(ctx context.Context, req *connect.Request[eventsv1.RevokeCollaborationInviteRequest]) (*connect.Response[eventsv1.RevokeCollaborationInviteResponse], error) {
	slog.Debug("RevokeCollaborationInvite", "inviteId", req.Msg.InviteId)

	invite := &models.EventCollaborationInvite{}
	err := s.db.NewSelect().Model(invite).Where("id = ?", req.Msg.InviteId).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	_, err = s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", invite.EventID), "edit")
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	event := &models.Event{}
	if err := tx.NewSelect().Model(event).Where("id = ?", invite.EventID).For("UPDATE").Scan(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to lock collaboration event: %w", err))
	}
	if err := tx.NewSelect().Model(invite).WherePK().For("UPDATE").Scan(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to lock collaboration invite: %w", err))
	}
	_, err = tx.NewUpdate().Model((*models.EventCollaborationInvite)(nil)).
		Set("status = ?", models.CollaborationInviteStatusRevoked).
		Set("updated_at = NOW()").
		Where("id = ?", req.Msg.InviteId).
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke invite: %w", err))
	}
	invite.Status = models.CollaborationInviteStatusRevoked
	if err := syncCollaborationPublication(ctx, tx, event, invite.OrganizationID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.RevokeCollaborationInviteResponse{
		Invite: dbCollaborationInviteToProto(*invite),
	}), nil
}

func (s *Service) GetCollaborationInviteByToken(ctx context.Context, req *connect.Request[eventsv1.GetCollaborationInviteByTokenRequest]) (*connect.Response[eventsv1.GetCollaborationInviteByTokenResponse], error) {
	slog.Debug("GetCollaborationInviteByToken")

	var row inviteByTokenRow
	err := s.db.NewSelect().
		TableExpr("event_collaboration_invites AS i").
		ColumnExpr("i.*").
		ColumnExpr("e.title AS event_title").
		ColumnExpr("es.slug AS event_slug").
		ColumnExpr("o.title AS organization_title").
		Join("INNER JOIN events e ON e.id = i.event_id").
		Join("LEFT JOIN event_slugs es ON es.event_id = e.id AND es.is_current = true").
		Join("INNER JOIN organizations o ON o.id = i.organization_id").
		Where("i.token = ?", req.Msg.Token).
		Scan(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.GetCollaborationInviteByTokenResponse{
		Invite: dbCollaborationInviteByTokenRowToProto(row),
	}), nil
}

func (s *Service) ListCollaborationInvitesByEvent(ctx context.Context, req *connect.Request[eventsv1.ListCollaborationInvitesByEventRequest]) (*connect.Response[eventsv1.ListCollaborationInvitesByEventResponse], error) {
	slog.Debug("ListCollaborationInvitesByEvent", "eventId", req.Msg.EventId)

	_, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "edit")
	if err != nil {
		return nil, err
	}

	var rows []collaborationInviteByEventRow
	err = s.db.NewSelect().
		TableExpr("event_collaboration_invites AS i").
		ColumnExpr("i.*").
		ColumnExpr("o.title AS organization_title").
		ColumnExpr("o.image_url AS organization_image_url").
		ColumnExpr("u.username AS invited_by_username").
		ColumnExpr("u.first_name AS invited_by_first_name").
		ColumnExpr("u.last_name AS invited_by_last_name").
		Join("INNER JOIN organizations o ON o.id = i.organization_id").
		Join("INNER JOIN users u ON u.id = i.invited_by_user_id").
		Where("i.event_id = ?", req.Msg.EventId).
		OrderExpr("i.created_at DESC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list invites: %w", err))
	}

	invites := lo.Map(rows, func(row collaborationInviteByEventRow, _ int) *eventsv1.CollaborationInvite {
		return dbCollaborationInviteByEventRowToProto(row)
	})

	return connect.NewResponse(&eventsv1.ListCollaborationInvitesByEventResponse{
		Invites: invites,
	}), nil
}

func (s *Service) ListPendingInvitesForOrganization(ctx context.Context, req *connect.Request[eventsv1.ListPendingInvitesForOrganizationRequest]) (*connect.Response[eventsv1.ListPendingInvitesForOrganizationResponse], error) {
	slog.Debug("ListPendingInvitesForOrganization", "orgId", req.Msg.OrganizationId)

	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	if !s.authz.CheckOrgPermission(ctx, localUserID, req.Msg.OrganizationId, "manage_settings") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you don't have permission to manage settings for this organization"))
	}

	var rows []pendingInviteRow
	err = s.db.NewSelect().
		TableExpr("event_collaboration_invites AS i").
		ColumnExpr("i.*").
		ColumnExpr("e.title AS event_title").
		ColumnExpr("es.slug AS event_slug").
		ColumnExpr("e.start_time").
		ColumnExpr("e.end_time").
		ColumnExpr("ho.title AS host_organization_title").
		ColumnExpr("ho.image_url AS host_organization_image_url").
		ColumnExpr("u.username AS invited_by_username").
		ColumnExpr("u.first_name AS invited_by_first_name").
		ColumnExpr("u.last_name AS invited_by_last_name").
		Join("INNER JOIN events e ON e.id = i.event_id").
		Join("LEFT JOIN event_slugs es ON es.event_id = e.id AND es.is_current = true").
		Join("INNER JOIN organizations ho ON ho.id = e.organization_id").
		Join("INNER JOIN users u ON u.id = i.invited_by_user_id").
		Where("i.organization_id = ?", req.Msg.OrganizationId).
		Where("i.status = ?", models.CollaborationInviteStatusPending).
		Where("i.expires_at > NOW()").
		OrderExpr("i.created_at DESC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list invites: %w", err))
	}

	invites := lo.Map(rows, func(row pendingInviteRow, _ int) *eventsv1.CollaborationInvite {
		return dbCollaborationInvitePendingRowToProto(row)
	})

	return connect.NewResponse(&eventsv1.ListPendingInvitesForOrganizationResponse{
		Invites: invites,
	}), nil
}
