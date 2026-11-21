package users

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) UpdateUser(ctx context.Context, req *connect.Request[usersv1.UpdateUserRequest]) (*connect.Response[usersv1.UpdateUserResponse], error) {
	slog.Debug("UpdateUser", "id", req.Msg.Id)
	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	if localUserID != req.Msg.Id {
		if _, err := s.authz.RequirePlatformAdmin(ctx); err != nil {
			return nil, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	var user models.User
	query := tx.NewUpdate().
		Model(&user).
		Set("updated_at = NOW()").
		Where("id = ?", req.Msg.Id)
	if req.Msg.Username != nil {
		query = query.Set("username = ?", *req.Msg.Username)
	}
	if req.Msg.Email != nil {
		query = query.Set("email = ?", *req.Msg.Email)
	}
	err = query.Returning("*").Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if publisher != nil {
		if err := publisher.PublishUserUpdated(ctx, events.UserUpdatedPayload{
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

	return connect.NewResponse(&usersv1.UpdateUserResponse{
		User: s.modelUserToProto(ctx, user),
	}), nil
}

func (s *Service) UpdatePassword(ctx context.Context, req *connect.Request[usersv1.UpdatePasswordRequest]) (*connect.Response[usersv1.UpdatePasswordResponse], error) {
	slog.Debug("UpdatePassword", "id", req.Msg.Id)

	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("password updates are handled via authentication provider - use the forgot password flow"))
}
