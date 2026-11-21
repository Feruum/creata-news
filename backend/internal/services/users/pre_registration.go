package users

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) PreRegisterUser(ctx context.Context, req *connect.Request[usersv1.PreRegisterUserRequest]) (*connect.Response[usersv1.PreRegisterUserResponse], error) {
	slog.Debug("PreRegisterUser", "email", req.Msg.Email, "role", req.Msg.PlatformRole)

	email := strings.ToLower(strings.TrimSpace(req.Msg.Email))

	var dbRole models.PlatformRole
	switch req.Msg.PlatformRole {
	case usersv1.PlatformRole_PLATFORM_ROLE_ADMIN:
		dbRole = models.PlatformRoleAdmin
	case usersv1.PlatformRole_PLATFORM_ROLE_STAFF:
		dbRole = models.PlatformRoleStaff
	}

	localUserID, err := s.authz.RequirePlatformStaff(ctx)
	if err != nil {
		return nil, err
	}

	preReg := models.PreRegisteredUser{
		Email:        email,
		PlatformRole: dbRole,
		CreatedBy:    &localUserID,
	}
	err = s.db.NewInsert().Model(&preReg).Returning("*").Scan(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("email is already pre-registered"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.InfoContext(ctx, "Created pre-registration", "email", email, "role", dbRole)

	return connect.NewResponse(&usersv1.PreRegisterUserResponse{
		PreRegisteredUser: modelPreRegToProto(preReg),
	}), nil
}

func (s *Service) ListPreRegisteredUsers(ctx context.Context, req *connect.Request[usersv1.ListPreRegisteredUsersRequest]) (*connect.Response[usersv1.ListPreRegisteredUsersResponse], error) {
	slog.Debug("ListPreRegisteredUsers", "page", req.Msg.Page, "limit", req.Msg.Limit, "includeUsed", req.Msg.IncludeUsed)

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

	var preRegs []models.PreRegisteredUser
	query := s.db.NewSelect().Model(&preRegs).
		Order("created_at DESC").
		Limit(int(limit)).
		Offset(int((page - 1) * limit))
	if !req.Msg.IncludeUsed {
		query = query.Where("used_at IS NULL")
	}
	err := query.Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	countQuery := s.db.NewRaw("SELECT COUNT(*) FROM pre_registered_users")
	if !req.Msg.IncludeUsed {
		countQuery = s.db.NewRaw("SELECT COUNT(*) FROM pre_registered_users WHERE used_at IS NULL")
	}
	var total int32
	err = countQuery.Scan(ctx, &total)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoPreRegs := lo.Map(preRegs, func(preReg models.PreRegisteredUser, _ int) *usersv1.PreRegisteredUser {
		return modelPreRegToProto(preReg)
	})

	return connect.NewResponse(&usersv1.ListPreRegisteredUsersResponse{
		PreRegisteredUsers: protoPreRegs,
		Total:              int32(total),
	}), nil
}

func (s *Service) DeletePreRegisteredUser(ctx context.Context, req *connect.Request[usersv1.DeletePreRegisteredUserRequest]) (*connect.Response[usersv1.DeletePreRegisteredUserResponse], error) {
	slog.Debug("DeletePreRegisteredUser", "id", req.Msg.Id)
	if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
		return nil, err
	}

	_, err := s.db.NewDelete().Model((*models.PreRegisteredUser)(nil)).Where("id = ?", req.Msg.Id).Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&usersv1.DeletePreRegisteredUserResponse{
		Success: true,
	}), nil
}
