package rooms

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"

	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
)

func (s *Service) ListBookings(ctx context.Context, req *connect.Request[roomsv1.ListBookingsRequest]) (*connect.Response[roomsv1.ListBookingsResponse], error) {
	if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
		return nil, err
	}

	limit := int32(20)
	if req.Msg.Limit > 0 {
		limit = req.Msg.Limit
	}
	offset := int32(0)
	if req.Msg.Page > 0 {
		offset = (req.Msg.Page - 1) * limit
	}

	var bookings []models.RoomBooking
	query := s.db.NewSelect().Model(&bookings)

	if req.Msg.RoomId != nil {
		normalizedRoomID := roomid.Normalize(*req.Msg.RoomId)
		if normalizedRoomID == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room_id cannot be empty"))
		}
		query = query.Where("room_id = ?", normalizedRoomID)
	}
	if req.Msg.EventId != nil {
		query = query.Where("event_id = ?", *req.Msg.EventId)
	}
	if req.Msg.FromDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.Msg.FromDate)
		query = query.Where("start_time >= ?", t)
	}
	if req.Msg.ToDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.Msg.ToDate)
		query = query.Where("end_time <= ?", t)
	}

	query = query.OrderExpr("start_time").Limit(int(limit)).Offset(int(offset))

	err := query.Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list bookings: %w", err))
	}

	countQuery := s.db.NewSelect().Model((*models.RoomBooking)(nil))
	if req.Msg.RoomId != nil {
		normalizedRoomID := roomid.Normalize(*req.Msg.RoomId)
		if normalizedRoomID == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room_id cannot be empty"))
		}
		countQuery = countQuery.Where("room_id = ?", normalizedRoomID)
	}
	if req.Msg.EventId != nil {
		countQuery = countQuery.Where("event_id = ?", *req.Msg.EventId)
	}
	if req.Msg.FromDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.Msg.FromDate)
		countQuery = countQuery.Where("start_time >= ?", t)
	}
	if req.Msg.ToDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.Msg.ToDate)
		countQuery = countQuery.Where("end_time <= ?", t)
	}
	total, _ := countQuery.Count(ctx)

	protoBookings := lo.Map(bookings, func(booking models.RoomBooking, _ int) *roomsv1.Booking {
		return dbBookingToProto(booking)
	})

	return connect.NewResponse(&roomsv1.ListBookingsResponse{
		Bookings: protoBookings,
		Total:    int32(total),
	}), nil
}

func (s *Service) CancelBooking(ctx context.Context, req *connect.Request[roomsv1.CancelRoomBookingRequest]) (*connect.Response[roomsv1.CancelRoomBookingResponse], error) {
	_, err := s.authz.RequirePlatformStaff(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("begin booking cancellation: %w", err))
	}
	defer func() { _ = tx.Rollback() }()

	// Lock and cancel the linked request before deleting its reservation, matching
	// event readiness's request-before-booking lock order. Missing/unlinked
	// reservations keep the existing idempotent cancellation behavior.
	_, err = tx.NewUpdate().Model((*models.BookingRequest)(nil)).
		Set("status = ?", models.BookingRequestStatusCancelled).
		Set("updated_at = NOW()").
		Where("id IN (SELECT booking_request_id FROM room_bookings WHERE id = ?)", req.Msg.Id).
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to cancel linked booking request: %w", err))
	}
	_, err = tx.NewDelete().Model((*models.RoomBooking)(nil)).Where("id = ?", req.Msg.Id).Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to cancel booking: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("commit booking cancellation: %w", err))
	}

	return connect.NewResponse(&roomsv1.CancelRoomBookingResponse{
		Success: true,
	}), nil
}
