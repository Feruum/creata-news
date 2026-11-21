package users

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) CreateUser(ctx context.Context, req *connect.Request[usersv1.CreateUserRequest]) (*connect.Response[usersv1.CreateUserResponse], error) {
	slog.Debug("CreateUser", "username", req.Msg.Username, "email", req.Msg.Email)
	if _, err := s.authz.RequirePlatformAdmin(ctx); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	user := models.User{
		Username: req.Msg.Username,
		Email:    req.Msg.Email,
		Password: req.Msg.Password,
	}
	err = tx.NewInsert().Model(&user).Returning("*").Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if publisher != nil {
		if err := publisher.PublishUserCreated(ctx, events.UserCreatedPayload{
			UserID:   user.ID,
			Username: user.Username,
			Email:    user.Email,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&usersv1.CreateUserResponse{
		User: s.modelUserToProto(ctx, user),
	}), nil
}
