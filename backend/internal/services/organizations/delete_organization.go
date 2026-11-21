package organizations

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/services/orgmembers"
)

func (s *Service) DeleteOrganization(ctx context.Context, req *connect.Request[eventsv1.DeleteOrganizationRequest]) (*connect.Response[eventsv1.DeleteOrganizationResponse], error) {
	slog.Debug("DeleteOrganization", "id", req.Msg.Id)

	// Authorization: Check if user can delete organizations (platform admin/staff)
	_, err := s.authz.RequirePlatformStaff(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	// Fetch org before deletion for event payload
	var org models.Organization
	if err := tx.NewSelect().Model(&org).Where("id = ?", req.Msg.Id).Scan(ctx); err != nil {
		slog.Warn("Failed to fetch org before deletion", "error", err)
	}

	if s.orgMembers != nil {
		if err := orgmembers.RecordBulkMembershipHistoryInTransaction(ctx, tx, req.Msg.Id, org.Title, models.MembershipHistoryActionOrgDeleted, nil); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	_, err = tx.NewDelete().Model((*models.Organization)(nil)).Where("id = ?", req.Msg.Id).Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	localUserID, _ := s.authz.GetLocalUserID(ctx)
	if publisher != nil {
		if err := publisher.PublishOrganizationDeleted(ctx, events.OrganizationDeletedPayload{
			OrganizationID:    req.Msg.Id,
			OrganizationTitle: org.Title,
			ActorID:           localUserID,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.DeleteOrganizationResponse{
		Success: true,
	}), nil
}
