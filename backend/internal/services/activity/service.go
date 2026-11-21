package activity

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	activityv1 "github.com/studyverse/ems-backend/gen/activity/v1"
	"github.com/studyverse/ems-backend/gen/activity/v1/activityv1connect"
	activitylog "github.com/studyverse/ems-backend/internal/activity"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

var _ activityv1connect.ActivityServiceHandler = (*Service)(nil)

type Service struct {
	activityv1connect.UnimplementedActivityServiceHandler
	db    *bun.DB
	authz *authz.Enforcer
}

func NewService(db *bun.DB, authzEnforcer *authz.Enforcer) *Service {
	return &Service{db: db, authz: authzEnforcer}
}

func (s *Service) ListActivityLog(ctx context.Context, req *connect.Request[activityv1.ListActivityLogRequest]) (*connect.Response[activityv1.ListActivityLogResponse], error) {
	slog.Debug("ListActivityLog",
		"org_ids", req.Msg.OrganizationIds,
		"actions", req.Msg.ActionFilters,
		"page", req.Msg.Page,
		"limit", req.Msg.Limit,
	)

	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}

	isAdmin := s.authz.CheckPlatformAdmin(ctx, localUserID)
	isStaff := s.authz.CheckPlatformStaff(ctx, localUserID)

	if !isAdmin && !isStaff {
		if len(req.Msg.OrganizationIds) == 0 {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("managers must specify organization_ids"))
		}
		for _, orgID := range req.Msg.OrganizationIds {
			if !s.authz.CheckOrgPermission(ctx, localUserID, orgID, "manage") {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("no permission for organization %d", orgID))
			}
		}
	}

	q := s.db.NewSelect().Model((*models.ActivityLog)(nil))

	if len(req.Msg.OrganizationIds) > 0 {
		q = q.Where("organization_id IN (?)", bun.In(req.Msg.OrganizationIds))
	}
	if len(req.Msg.ActionFilters) > 0 {
		q = q.Where("action IN (?)", bun.In(req.Msg.ActionFilters))
	}
	if req.Msg.After != nil {
		q = q.Where("created_at >= ?", *req.Msg.After)
	}
	if req.Msg.Before != nil {
		q = q.Where("created_at <= ?", *req.Msg.Before)
	}
	if req.Msg.ActorId != nil {
		q = q.Where("actor_id = ?", *req.Msg.ActorId)
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 20
	}
	page := int(req.Msg.Page)
	if page <= 0 {
		page = 1
	}

	var entries []models.ActivityLog
	err = q.OrderExpr("created_at DESC").
		Limit(limit).
		Offset((page-1)*limit).
		Scan(ctx, &entries)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	actorEntries := make([]*models.ActivityLog, len(entries))
	for i := range entries {
		actorEntries[i] = &entries[i]
	}
	if err := activitylog.FillActorSnapshots(ctx, s.db, actorEntries); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&activityv1.ListActivityLogResponse{
		Entries: toProtoEntries(entries),
		Total:   int32(total),
		Page:    int32(page),
		Limit:   int32(limit),
	}), nil
}

func (s *Service) ListEntityActivityLog(ctx context.Context, req *connect.Request[activityv1.ListEntityActivityLogRequest]) (*connect.Response[activityv1.ListActivityLogResponse], error) {
	slog.Debug("ListEntityActivityLog",
		"entity_type", req.Msg.EntityType,
		"entity_id", req.Msg.EntityId,
	)

	q := s.db.NewSelect().Model((*models.ActivityLog)(nil)).
		Where("entity_type = ?", req.Msg.EntityType).
		Where("entity_id = ?", req.Msg.EntityId)

	total, err := q.Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 20
	}
	page := int(req.Msg.Page)
	if page <= 0 {
		page = 1
	}

	var entries []models.ActivityLog
	err = q.OrderExpr("created_at DESC").
		Limit(limit).
		Offset((page-1)*limit).
		Scan(ctx, &entries)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	actorEntries := make([]*models.ActivityLog, len(entries))
	for i := range entries {
		actorEntries[i] = &entries[i]
	}
	if err := activitylog.FillActorSnapshots(ctx, s.db, actorEntries); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&activityv1.ListActivityLogResponse{
		Entries: toProtoEntries(entries),
		Total:   int32(total),
		Page:    int32(page),
		Limit:   int32(limit),
	}), nil
}
