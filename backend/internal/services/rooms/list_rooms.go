package rooms

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/samber/lo"

	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) ListRooms(ctx context.Context, req *connect.Request[roomsv1.ListRoomsRequest]) (*connect.Response[roomsv1.ListRoomsResponse], error) {
	if req.Msg.IncludeInactive {
		if _, err := s.authz.RequirePlatformAdmin(ctx); err != nil {
			return nil, err
		}
	}
	limit := int32(20)
	if req.Msg.Limit > 0 {
		limit = req.Msg.Limit
	}
	offset := int32(0)
	if req.Msg.Page > 0 {
		offset = (req.Msg.Page - 1) * limit
	}

	baseQuery := func(q interface {
		Where(string, ...interface{}) interface {
			Where(string, ...interface{}) interface{}
		}
	}) {
	}
	_ = baseQuery

	var rooms []models.Room
	query := s.db.NewSelect().Model(&rooms)
	if !req.Msg.IncludeInactive {
		query = query.Where("is_active = ?", true)
	}

	if req.Msg.Floor != nil {
		query = query.Where("floor = ?", *req.Msg.Floor)
	}
	if req.Msg.Type != nil && *req.Msg.Type != roomsv1.RoomType_ROOM_TYPE_UNSPECIFIED {
		query = query.Where("type = ?", protoToDBRoomType(*req.Msg.Type))
	}
	if req.Msg.Block != nil {
		query = query.Where("block = ?", *req.Msg.Block)
	}
	if req.Msg.BookableOnly != nil && *req.Msg.BookableOnly {
		query = query.Where("is_bookable = ?", true)
	}
	if req.Msg.Search != nil {
		search := "%" + *req.Msg.Search + "%"
		query = query.Where("(name ILIKE ? OR room_number ILIKE ?)", search, search)
	}

	query = query.OrderExpr("block, floor, room_number").Limit(int(limit)).Offset(int(offset))

	err := query.Scan(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "ListRooms query failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list rooms: %w", err))
	}

	countQuery := s.db.NewSelect().Model((*models.Room)(nil))
	if !req.Msg.IncludeInactive {
		countQuery = countQuery.Where("is_active = ?", true)
	}
	if req.Msg.Floor != nil {
		countQuery = countQuery.Where("floor = ?", *req.Msg.Floor)
	}
	if req.Msg.Type != nil && *req.Msg.Type != roomsv1.RoomType_ROOM_TYPE_UNSPECIFIED {
		countQuery = countQuery.Where("type = ?", protoToDBRoomType(*req.Msg.Type))
	}
	if req.Msg.Block != nil {
		countQuery = countQuery.Where("block = ?", *req.Msg.Block)
	}
	if req.Msg.BookableOnly != nil && *req.Msg.BookableOnly {
		countQuery = countQuery.Where("is_bookable = ?", true)
	}
	if req.Msg.Search != nil {
		search := "%" + *req.Msg.Search + "%"
		countQuery = countQuery.Where("(name ILIKE ? OR room_number ILIKE ?)", search, search)
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "CountRooms query failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count rooms: %w", err))
	}

	protoRooms := lo.Map(rooms, func(room models.Room, _ int) *roomsv1.Room {
		return dbRoomToProto(room)
	})

	return connect.NewResponse(&roomsv1.ListRoomsResponse{
		Rooms: protoRooms,
		Total: int32(total),
	}), nil
}
