package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	"github.com/uptrace/bun"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/perms"
)

type CategoriesService struct {
	eventsv1connect.UnimplementedCategoriesServiceHandler
	db    *bun.DB
	perms *perms.Client
	authz *authz.Enforcer
}

func NewCategoriesService(db *bun.DB, permsClient *perms.Client, authzEnforcer *authz.Enforcer) *CategoriesService {
	return &CategoriesService{db: db, perms: permsClient, authz: authzEnforcer}
}

func (s *CategoriesService) CreateCategory(ctx context.Context, req *connect.Request[eventsv1.CreateCategoryRequest]) (*connect.Response[eventsv1.CreateCategoryResponse], error) {
	slog.Debug("CreateCategory", "name", req.Msg.Name, "color", req.Msg.Color)

	// Authorization: Check if user can create categories (platform admin/staff)
	_, err := s.authz.RequirePlatformStaff(ctx)
	if err != nil {
		return nil, err
	}

	category := models.Category{
		Name:  req.Msg.Name,
		Color: req.Msg.Color,
	}
	err = s.db.NewInsert().Model(&category).Returning("*").Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.CreateCategoryResponse{
		Category: dbCategoryToProto(&category),
	}), nil
}

func (s *CategoriesService) GetCategory(ctx context.Context, req *connect.Request[eventsv1.GetCategoryRequest]) (*connect.Response[eventsv1.GetCategoryResponse], error) {
	slog.Debug("GetCategory", "id", req.Msg.Id)

	var category models.Category
	err := s.db.NewSelect().Model(&category).Where("id = ?", req.Msg.Id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.GetCategoryResponse{
		Category: dbCategoryToProto(&category),
	}), nil
}

func (s *CategoriesService) ListCategories(ctx context.Context, req *connect.Request[eventsv1.ListCategoriesRequest]) (*connect.Response[eventsv1.ListCategoriesResponse], error) {
	slog.Debug("ListCategories", "page", req.Msg.Page, "limit", req.Msg.Limit)

	page := req.Msg.Page
	if page <= 0 {
		page = 1
	}
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 100 // Default to higher limit for categories since there are few
	}

	var categories []models.Category
	err := s.db.NewSelect().Model(&categories).
		Order("id").
		Limit(int(limit)).
		Offset(int((page - 1) * limit)).
		Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var total int32
	err = s.db.NewRaw("SELECT COUNT(*) FROM categories").Scan(ctx, &total)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoCategories := lo.Map(categories, func(category models.Category, _ int) *eventsv1.Category {
		return dbCategoryToProto(&category)
	})

	return connect.NewResponse(&eventsv1.ListCategoriesResponse{
		Categories: protoCategories,
		Total:      total,
	}), nil
}

func (s *CategoriesService) UpdateCategory(ctx context.Context, req *connect.Request[eventsv1.UpdateCategoryRequest]) (*connect.Response[eventsv1.UpdateCategoryResponse], error) {
	slog.Debug("UpdateCategory", "id", req.Msg.Id)

	// Authorization: Check if user can update categories (platform admin/staff)
	_, err := s.authz.RequirePlatformStaff(ctx)
	if err != nil {
		return nil, err
	}

	var category models.Category
	query := s.db.NewUpdate().
		Model(&category).
		Set("updated_at = NOW()").
		Where("id = ?", req.Msg.Id)
	if req.Msg.Name != nil {
		query = query.Set("name = ?", *req.Msg.Name)
	}
	if req.Msg.Color != nil {
		query = query.Set("color = ?", *req.Msg.Color)
	}
	err = query.Returning("*").Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.UpdateCategoryResponse{
		Category: dbCategoryToProto(&category),
	}), nil
}

func (s *CategoriesService) DeleteCategory(ctx context.Context, req *connect.Request[eventsv1.DeleteCategoryRequest]) (*connect.Response[eventsv1.DeleteCategoryResponse], error) {
	slog.Debug("DeleteCategory", "id", req.Msg.Id)

	// Authorization: Check if user can delete categories (platform admin/staff)
	_, err := s.authz.RequirePlatformStaff(ctx)
	if err != nil {
		return nil, err
	}

	_, err = s.db.NewDelete().Model((*models.Category)(nil)).Where("id = ?", req.Msg.Id).Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.DeleteCategoryResponse{
		Success: true,
	}), nil
}

// Helper function to convert models.Category to proto
func dbCategoryToProto(c *models.Category) *eventsv1.Category {
	return &eventsv1.Category{
		Id:        c.ID,
		Name:      c.Name,
		Color:     c.Color,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
