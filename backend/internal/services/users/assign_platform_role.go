package users

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) AssignPlatformRole(ctx context.Context, req *connect.Request[usersv1.AssignPlatformRoleRequest]) (*connect.Response[usersv1.AssignPlatformRoleResponse], error) {
	slog.Debug("AssignPlatformRole", "userId", req.Msg.UserId, "role", req.Msg.Role)
	if _, err := s.authz.RequirePlatformAdmin(ctx); err != nil {
		return nil, err
	}

	relation, err := platformRoleRelation(req.Msg.Role)
	if err != nil {
		return nil, err
	}
	var user models.User
	err = s.db.NewSelect().Model(&user).Where("id = ?", req.Msg.UserId).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.authz.AssignPlatformRole(ctx, user.ID, relation); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "Assigned platform role", "userId", user.ID, "role", req.Msg.Role)

	return connect.NewResponse(&usersv1.AssignPlatformRoleResponse{
		User: s.modelUserToProto(ctx, user),
	}), nil
}
func platformRoleRelation(role usersv1.PlatformRole) (string, error) {
	switch role {
	case usersv1.PlatformRole_PLATFORM_ROLE_ADMIN:
		return "administrator", nil
	case usersv1.PlatformRole_PLATFORM_ROLE_STAFF:
		return "staff", nil
	case usersv1.PlatformRole_PLATFORM_ROLE_USER:
		return "", nil
	default:
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported platform role"))
	}
}
