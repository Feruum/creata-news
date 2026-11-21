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

func (s *Service) GetUser(ctx context.Context, req *connect.Request[usersv1.GetUserRequest]) (*connect.Response[usersv1.GetUserResponse], error) {
	slog.Debug("GetUser", "id", req.Msg.Id)

	var user models.User
	err := s.db.NewSelect().Model(&user).Where("id = ?", req.Msg.Id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&usersv1.GetUserResponse{
		User: modelUserToPublicProto(user),
	}), nil
}

func (s *Service) GetUserByEmail(ctx context.Context, req *connect.Request[usersv1.GetUserByEmailRequest]) (*connect.Response[usersv1.GetUserByEmailResponse], error) {
	slog.Debug("GetUserByEmail", "email", req.Msg.Email)

	if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
		return nil, err
	}

	var user models.User
	err := s.db.NewSelect().Model(&user).Where("email = ?", req.Msg.Email).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&usersv1.GetUserByEmailResponse{
		User: s.modelUserToProto(ctx, user),
	}), nil
}

func (s *Service) GetUserByUsername(ctx context.Context, req *connect.Request[usersv1.GetUserByUsernameRequest]) (*connect.Response[usersv1.GetUserByUsernameResponse], error) {
	slog.Debug("GetUserByUsername", "username", req.Msg.Username)

	var user models.User
	err := s.db.NewSelect().Model(&user).Where("username = ?", req.Msg.Username).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&usersv1.GetUserByUsernameResponse{
		User: modelUserToPublicProto(user),
	}), nil
}
