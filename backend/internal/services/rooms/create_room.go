package rooms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
)

func (s *Service) CreateRoom(ctx context.Context, req *connect.Request[roomsv1.CreateRoomRequest]) (*connect.Response[roomsv1.CreateRoomResponse], error) {
	_, err := s.authz.RequirePlatformAdmin(ctx)
	if err != nil {
		return nil, err
	}

	normalizedRoomID := roomid.Normalize(req.Msg.Id)
	if normalizedRoomID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room id is required"))
	}

	room := models.Room{
		ID:           normalizedRoomID,
		Name:         req.Msg.Name,
		RoomNumber:   req.Msg.RoomNumber,
		Block:        req.Msg.Block,
		Floor:        req.Msg.Floor,
		Type:         protoToDBRoomType(req.Msg.Type),
		IsAccessible: req.Msg.IsAccessible,
		IsBookable:   req.Msg.IsBookable,
		IsActive:     true,
	}

	if req.Msg.Capacity != nil {
		room.Capacity = req.Msg.Capacity
	}
	if req.Msg.Description != nil {
		room.Description = req.Msg.Description
	}
	if len(req.Msg.Features) > 0 {
		featuresJSON, _ := json.Marshal(req.Msg.Features)
		raw := json.RawMessage(featuresJSON)
		room.Features = &raw
	}
	if req.Msg.PrimaryImageUrl != nil {
		room.PrimaryImageUrl = req.Msg.PrimaryImageUrl
	}

	_, err = s.db.NewInsert().Model(&room).Returning("*").Exec(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "CreateRoom query failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create room: %w", err))
	}

	return connect.NewResponse(&roomsv1.CreateRoomResponse{
		Room: dbRoomToProto(room),
	}), nil
}
