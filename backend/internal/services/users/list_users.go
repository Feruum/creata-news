package users

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) ListUsers(ctx context.Context, req *connect.Request[usersv1.ListUsersRequest]) (*connect.Response[usersv1.ListUsersResponse], error) {
	slog.Debug("ListUsers", "page", req.Msg.Page, "limit", req.Msg.Limit)
	if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
		return nil, err
	}

	page := req.Msg.Page
	if page <= 0 {
		page = 1
	}
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 50
	}

	var users []models.User
	err := s.db.NewSelect().Model(&users).
		Order("id").
		Limit(int(limit)).
		Offset(int((page - 1) * limit)).
		Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var total int32
	err = s.db.NewRaw("SELECT COUNT(*) FROM users").Scan(ctx, &total)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoUsers := lo.Map(users, func(user models.User, _ int) *usersv1.User {
		return s.modelUserToProto(ctx, user)
	})

	return connect.NewResponse(&usersv1.ListUsersResponse{
		Users: protoUsers,
		Total: int32(total),
	}), nil
}
