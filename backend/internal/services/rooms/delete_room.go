package rooms

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
)

func (s *Service) DeleteRoom(ctx context.Context, req *connect.Request[roomsv1.DeleteRoomRequest]) (*connect.Response[roomsv1.DeleteRoomResponse], error) {
	_, err := s.authz.RequirePlatformAdmin(ctx)
	if err != nil {
		return nil, err
	}

	normalizedRoomID := roomid.Normalize(req.Msg.Id)
	if normalizedRoomID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room id is required"))
	}

	_, err = s.db.NewDelete().Model((*models.Room)(nil)).Where("id = ?", normalizedRoomID).Exec(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "DeleteRoom query failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete room: %w", err))
	}

	return connect.NewResponse(&roomsv1.DeleteRoomResponse{
		Success: true,
	}), nil
}
