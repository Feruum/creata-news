package rooms

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	"github.com/uptrace/bun"

	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
)

func (s *Service) GetRoomAvailability(ctx context.Context, req *connect.Request[roomsv1.GetRoomAvailabilityRequest]) (*connect.Response[roomsv1.GetRoomAvailabilityResponse], error) {
	normalizedRoomID := roomid.Normalize(req.Msg.RoomId)
	if normalizedRoomID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room_id is required"))
	}

	date, err := campusAvailabilityDay(req.Msg.Date)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid date format, expected YYYY-MM-DD"))
	}

	dayStart := date
	dayEnd := dayStart.AddDate(0, 0, 1)

	var bookings []models.RoomBooking
	err = s.db.NewSelect().Model(&bookings).
		Where("room_id = ?", normalizedRoomID).
		Where("end_time > ?", dayStart).
		Where("start_time < ?", dayEnd).
		OrderExpr("start_time").
		Scan(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "GetRoomBookingsForDate failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get bookings: %w", err))
	}

	var slots []*roomsv1.TimeSlot
	hourSlots := []struct{ start, end int }{{8, 9}, {9, 10}, {10, 11}, {11, 12}, {12, 13}, {13, 14}, {14, 15}, {15, 16}, {16, 17}, {17, 18}, {18, 19}, {19, 20}, {20, 21}}

	for _, hs := range hourSlots {
		slotStart := time.Date(date.Year(), date.Month(), date.Day(), hs.start, 0, 0, 0, date.Location())
		slotEnd := time.Date(date.Year(), date.Month(), date.Day(), hs.end, 0, 0, 0, date.Location())

		slot := &roomsv1.TimeSlot{
			StartTime:   slotStart.Format(time.RFC3339),
			EndTime:     slotEnd.Format(time.RFC3339),
			IsAvailable: true,
		}

		for _, b := range bookings {
			if overlaps(slotStart, slotEnd, b.StartTime, b.EndTime) {
				slot.IsAvailable = false
				slot.Booking = dbBookingToProto(b)
				break
			}
		}

		slots = append(slots, slot)
	}

	return connect.NewResponse(&roomsv1.GetRoomAvailabilityResponse{
		RoomId: normalizedRoomID,
		Date:   req.Msg.Date,
		Slots:  slots,
	}), nil
}

func (s *Service) CheckTimeSlot(ctx context.Context, req *connect.Request[roomsv1.CheckTimeSlotRequest]) (*connect.Response[roomsv1.CheckTimeSlotResponse], error) {
	normalizedRoomID := roomid.Normalize(req.Msg.RoomId)
	if normalizedRoomID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room_id is required"))
	}

	startTime, err := time.Parse(time.RFC3339, req.Msg.StartTime)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid start_time format"))
	}
	endTime, err := time.Parse(time.RFC3339, req.Msg.EndTime)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid end_time format"))
	}

	var bookings []models.RoomBooking
	err = s.db.NewSelect().Model(&bookings).
		Where("room_id = ?", normalizedRoomID).
		Where("(start_time, end_time) OVERLAPS (?, ?)", startTime, endTime).
		Scan(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "GetOverlappingBookings failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check bookings: %w", err))
	}

	var pendingRequests []models.BookingRequest
	err = s.db.NewSelect().Model(&pendingRequests).
		Where("room_id = ?", normalizedRoomID).
		Where("status = ?", models.BookingRequestStatusPending).
		Where("(start_time, end_time) OVERLAPS (?, ?)", startTime, endTime).
		Scan(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "GetOverlappingBookingRequests failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check pending requests: %w", err))
	}

	protoBookings := lo.Map(bookings, func(booking models.RoomBooking, _ int) *roomsv1.Booking {
		return dbBookingToProto(booking)
	})

	protoRequests := lo.Map(pendingRequests, func(request models.BookingRequest, _ int) *roomsv1.BookingRequest {
		return dbBookingRequestToProto(request)
	})

	return connect.NewResponse(&roomsv1.CheckTimeSlotResponse{
		IsAvailable:         len(bookings) == 0,
		ConflictingBookings: protoBookings,
		PendingRequests:     protoRequests,
	}), nil
}

func (s *Service) CheckRoomsTimeSlot(ctx context.Context, req *connect.Request[roomsv1.CheckRoomsTimeSlotRequest]) (*connect.Response[roomsv1.CheckRoomsTimeSlotResponse], error) {
	normalizedRoomIDs := roomid.NormalizeList(req.Msg.RoomIds)
	if len(normalizedRoomIDs) == 0 {
		return connect.NewResponse(&roomsv1.CheckRoomsTimeSlotResponse{
			Rooms: make(map[string]*roomsv1.RoomTimeSlotStatus),
		}), nil
	}

	startTime, err := time.Parse(time.RFC3339, req.Msg.StartTime)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid start_time format"))
	}
	endTime, err := time.Parse(time.RFC3339, req.Msg.EndTime)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid end_time format"))
	}

	var bookings []models.RoomBooking
	err = s.db.NewSelect().Model(&bookings).
		Where("room_id IN (?)", bun.In(normalizedRoomIDs)).
		Where("(start_time, end_time) OVERLAPS (?, ?)", startTime, endTime).
		Scan(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "GetOverlappingBookingsForRooms failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check bookings: %w", err))
	}

	bookingsByRoom := make(map[string]models.RoomBooking)
	for _, b := range bookings {
		if _, exists := bookingsByRoom[b.RoomID]; !exists {
			bookingsByRoom[b.RoomID] = b
		}
	}

	result := make(map[string]*roomsv1.RoomTimeSlotStatus, len(normalizedRoomIDs))
	for _, roomID := range normalizedRoomIDs {
		status := &roomsv1.RoomTimeSlotStatus{IsAvailable: true}
		if booking, hasConflict := bookingsByRoom[roomID]; hasConflict {
			status.IsAvailable = false
			status.ConflictingBooking = dbBookingToProto(booking)
		}
		result[roomID] = status
	}

	return connect.NewResponse(&roomsv1.CheckRoomsTimeSlotResponse{
		Rooms: result,
	}), nil
}

func (s *Service) GetAlternativeRooms(ctx context.Context, req *connect.Request[roomsv1.GetAlternativeRoomsRequest]) (*connect.Response[roomsv1.GetAlternativeRoomsResponse], error) {
	startTime, err := time.Parse(time.RFC3339, req.Msg.StartTime)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid start_time format"))
	}
	endTime, err := time.Parse(time.RFC3339, req.Msg.EndTime)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid end_time format"))
	}

	var rooms []models.Room
	query := s.db.NewSelect().Model(&rooms).
		Where("is_bookable = ?", true).
		Where("is_active = ?", true).
		Where("NOT EXISTS (SELECT 1 FROM room_bookings rb WHERE rb.room_id = r.id AND (rb.start_time, rb.end_time) OVERLAPS (?, ?))", startTime, endTime)

	if req.Msg.MinCapacity != nil {
		query = query.Where("capacity >= ?", *req.Msg.MinCapacity)
	}
	if req.Msg.PreferredType != nil && *req.Msg.PreferredType != roomsv1.RoomType_ROOM_TYPE_UNSPECIFIED {
		query = query.Where("type = ?", protoToDBRoomType(*req.Msg.PreferredType))
	}
	if req.Msg.PreferredFloor != nil {
		query = query.Where("floor = ?", *req.Msg.PreferredFloor)
	}

	query = query.OrderExpr("block, floor, room_number")

	err = query.Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get available rooms: %w", err))
	}

	protoRooms := lo.Map(rooms, func(room models.Room, _ int) *roomsv1.Room {
		return dbRoomToProto(room)
	})

	return connect.NewResponse(&roomsv1.GetAlternativeRoomsResponse{
		AvailableRooms: protoRooms,
	}), nil
}
