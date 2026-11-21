package rooms

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
)

func (s *Service) GetRoom(ctx context.Context, req *connect.Request[roomsv1.GetRoomRequest]) (*connect.Response[roomsv1.GetRoomResponse], error) {
	normalizedRoomID := roomid.Normalize(req.Msg.Id)
	if normalizedRoomID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room id is required"))
	}

	var room models.Room
	err := s.db.NewSelect().Model(&room).Where("id = ?", normalizedRoomID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("room not found"))
		}
		slog.ErrorContext(ctx, "GetRoom query failed", "error", err, "id", normalizedRoomID)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("room not found: %w", err))
	}

	return connect.NewResponse(&roomsv1.GetRoomResponse{
		Room: dbRoomToProto(room),
	}), nil
}
