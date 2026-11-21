package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

type OrganizationTypesService struct {
	eventsv1connect.UnimplementedOrganizationTypesServiceHandler
	db    *bun.DB
	authz *authz.Enforcer
}

func NewOrganizationTypesService(db *bun.DB, authzEnforcer *authz.Enforcer) *OrganizationTypesService {
	return &OrganizationTypesService{db: db, authz: authzEnforcer}
}

func (s *OrganizationTypesService) CreateOrganizationType(ctx context.Context, req *connect.Request[eventsv1.CreateOrganizationTypeRequest]) (*connect.Response[eventsv1.CreateOrganizationTypeResponse], error) {
	slog.Debug("CreateOrganizationType", "title", req.Msg.Title)

	if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
		return nil, err
	}

	ot := models.OrganizationType{Title: req.Msg.Title}
	err := s.db.NewInsert().Model(&ot).Returning("*").Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.CreateOrganizationTypeResponse{
		OrganizationType: dbOrganizationTypeToProto(ot),
	}), nil
}

func (s *OrganizationTypesService) GetOrganizationType(ctx context.Context, req *connect.Request[eventsv1.GetOrganizationTypeRequest]) (*connect.Response[eventsv1.GetOrganizationTypeResponse], error) {
	slog.Debug("GetOrganizationType", "id", req.Msg.Id)

	var ot models.OrganizationType
	err := s.db.NewSelect().Model(&ot).Where("id = ?", req.Msg.Id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.GetOrganizationTypeResponse{
		OrganizationType: dbOrganizationTypeToProto(ot),
	}), nil
}

func (s *OrganizationTypesService) ListOrganizationTypes(ctx context.Context, req *connect.Request[eventsv1.ListOrganizationTypesRequest]) (*connect.Response[eventsv1.ListOrganizationTypesResponse], error) {
	slog.Debug("ListOrganizationTypes", "page", req.Msg.Page, "limit", req.Msg.Limit)

	page := req.Msg.Page
	if page <= 0 {
		page = 1
	}
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 10
	}

	var types []models.OrganizationType
	err := s.db.NewSelect().Model(&types).Order("id").Limit(int(limit)).Offset(int((page - 1) * limit)).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	total, err := s.db.NewSelect().Model((*models.OrganizationType)(nil)).Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTypes := lo.Map(types, func(orgType models.OrganizationType, _ int) *eventsv1.OrganizationType {
		return dbOrganizationTypeToProto(orgType)
	})

	return connect.NewResponse(&eventsv1.ListOrganizationTypesResponse{
		OrganizationTypes: protoTypes,
		Total:             int32(total),
	}), nil
}

func (s *OrganizationTypesService) UpdateOrganizationType(ctx context.Context, req *connect.Request[eventsv1.UpdateOrganizationTypeRequest]) (*connect.Response[eventsv1.UpdateOrganizationTypeResponse], error) {
	slog.Debug("UpdateOrganizationType", "id", req.Msg.Id)

	if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
		return nil, err
	}

	query := s.db.NewUpdate().Model((*models.OrganizationType)(nil)).Where("id = ?", req.Msg.Id)

	if req.Msg.Title != nil {
		query = query.Set("title = ?", *req.Msg.Title)
	}

	query = query.Set("updated_at = NOW()")

	var ot models.OrganizationType
	err := query.Returning("*").Scan(ctx, &ot)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.UpdateOrganizationTypeResponse{
		OrganizationType: dbOrganizationTypeToProto(ot),
	}), nil
}

func (s *OrganizationTypesService) DeleteOrganizationType(ctx context.Context, req *connect.Request[eventsv1.DeleteOrganizationTypeRequest]) (*connect.Response[eventsv1.DeleteOrganizationTypeResponse], error) {
	slog.Debug("DeleteOrganizationType", "id", req.Msg.Id)

	if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
		return nil, err
	}

	_, err := s.db.NewDelete().Model((*models.OrganizationType)(nil)).Where("id = ?", req.Msg.Id).Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.DeleteOrganizationTypeResponse{
		Success: true,
	}), nil
}

func dbOrganizationTypeToProto(ot models.OrganizationType) *eventsv1.OrganizationType {
	return &eventsv1.OrganizationType{
		Id:        ot.ID,
		Title:     ot.Title,
		CreatedAt: ot.CreatedAt.Format(time.RFC3339),
		UpdatedAt: ot.UpdatedAt.Format(time.RFC3339),
	}
}
