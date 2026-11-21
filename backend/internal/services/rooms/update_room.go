package rooms

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
)

func (s *Service) UpdateRoom(ctx context.Context, req *connect.Request[roomsv1.UpdateRoomRequest]) (*connect.Response[roomsv1.UpdateRoomResponse], error) {
	_, err := s.authz.RequirePlatformAdmin(ctx)
	if err != nil {
		return nil, err
	}

	normalizedRoomID := roomid.Normalize(req.Msg.Id)
	if normalizedRoomID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room id is required"))
	}

	query := s.db.NewUpdate().Model((*models.Room)(nil)).Where("id = ?", normalizedRoomID)

	if req.Msg.Name != nil {
		query = query.Set("name = ?", *req.Msg.Name)
	}
	if req.Msg.RoomNumber != nil {
		query = query.Set("room_number = ?", *req.Msg.RoomNumber)
	}
	if req.Msg.Block != nil {
		query = query.Set("block = ?", *req.Msg.Block)
	}
	if req.Msg.Floor != nil {
		query = query.Set("floor = ?", *req.Msg.Floor)
	}
	if req.Msg.Type != nil && *req.Msg.Type != roomsv1.RoomType_ROOM_TYPE_UNSPECIFIED {
		query = query.Set("type = ?", protoToDBRoomType(*req.Msg.Type))
	}
	if req.Msg.Capacity != nil {
		query = query.Set("capacity = ?", *req.Msg.Capacity)
	}
	if req.Msg.Description != nil {
		query = query.Set("description = ?", *req.Msg.Description)
	}
	if len(req.Msg.Features) > 0 {
		featuresJSON, _ := json.Marshal(req.Msg.Features)
		query = query.Set("features = ?", json.RawMessage(featuresJSON))
	}
	if req.Msg.PrimaryImageUrl != nil {
		query = query.Set("primary_image_url = ?", *req.Msg.PrimaryImageUrl)
	}
	if req.Msg.IsAccessible != nil {
		query = query.Set("is_accessible = ?", *req.Msg.IsAccessible)
	}
	if req.Msg.IsBookable != nil {
		query = query.Set("is_bookable = ?", *req.Msg.IsBookable)
	}
	if req.Msg.IsActive != nil {
		query = query.Set("is_active = ?", *req.Msg.IsActive)
	}

	query = query.Set("updated_at = NOW()")

	var room models.Room
	err = query.Returning("*").Scan(ctx, &room)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("room not found"))
		}
		slog.ErrorContext(ctx, "UpdateRoom query failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update room: %w", err))
	}

	return connect.NewResponse(&roomsv1.UpdateRoomResponse{
		Room: dbRoomToProto(room),
	}), nil
}
