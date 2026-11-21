package users

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) DeleteUser(ctx context.Context, req *connect.Request[usersv1.DeleteUserRequest]) (*connect.Response[usersv1.DeleteUserResponse], error) {
	slog.Debug("DeleteUser", "id", req.Msg.Id)
	if _, err := s.authz.RequirePlatformAdmin(ctx); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	_, err = tx.NewDelete().Model((*models.User)(nil)).Where("id = ?", req.Msg.Id).Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if publisher != nil {
		if err := publisher.PublishUserDeleted(ctx, events.UserDeletedPayload{
			UserID: req.Msg.Id,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&usersv1.DeleteUserResponse{
		Success: true,
	}), nil
}
