package events

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
	"github.com/uptrace/bun"
)

func replacedPrimaryBooking(original, candidate models.Event, booking models.RoomBooking) bool {
	previous, next := parseStructuredLocation(original.StructuredLocation), parseStructuredLocation(candidate.StructuredLocation)
	if previous.GetType() != eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM && previous.GetType() != eventsv1.LocationType_LOCATION_TYPE_HYBRID {
		return false
	}
	if booking.EventID == nil || *booking.EventID != original.ID || roomid.Normalize(booking.RoomID) != roomid.Normalize(previous.GetRoomId()) || booking.StartTime.After(original.StartTime) || booking.EndTime.Before(original.EndTime) {
		return false
	}
	needsRoom := next.GetType() == eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM || next.GetType() == eventsv1.LocationType_LOCATION_TYPE_HYBRID
	return !needsRoom || roomid.Normalize(booking.RoomID) != roomid.Normalize(next.GetRoomId()) || booking.StartTime.After(candidate.StartTime) || booking.EndTime.Before(candidate.EndTime)
}

// Called inside the event mutation transaction, after the new booking was checked.
// Only replace the reservation for the previous primary location; unrelated rooms
// linked to this event and reservations still covering the new schedule are kept.
func (s *Service) releaseReplacedPrimaryBooking(ctx context.Context, original, candidate models.Event) error {
	var bookings []models.RoomBooking
	if err := s.db.NewSelect().Model(&bookings).Where("event_id = ?", original.ID).Scan(ctx); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("load previous event reservations: %w", err))
	}
	var bookingIDs, requestIDs []int32
	for _, booking := range bookings {
		if !replacedPrimaryBooking(original, candidate, booking) {
			continue
		}
		bookingIDs = append(bookingIDs, booking.ID)
		if booking.BookingRequestID != nil {
			requestIDs = append(requestIDs, *booking.BookingRequestID)
		}
	}
	if len(bookingIDs) == 0 {
		return nil
	}
	// Match cancellation's request-before-reservation lock order.
	if len(requestIDs) > 0 {
		var requests []models.BookingRequest
		if err := s.db.NewSelect().Model(&requests).Where("id IN (?)", bun.In(requestIDs)).OrderExpr("id ASC").For("UPDATE").Scan(ctx); err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("lock replaced booking requests: %w", err))
		}
	}
	if _, err := s.db.NewDelete().Model((*models.RoomBooking)(nil)).Where("id IN (?)", bun.In(bookingIDs)).Where("event_id = ?", original.ID).Exec(ctx); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("release previous event reservations: %w", err))
	}
	return s.cancelUnreservedBookingRequests(ctx, requestIDs)
}

func (s *Service) cancelUnreservedBookingRequests(ctx context.Context, requestIDs []int32) error {
	if len(requestIDs) == 0 {
		return nil
	}
	if _, err := s.db.NewUpdate().Model((*models.BookingRequest)(nil)).
		Set("status = ?", models.BookingRequestStatusCancelled).Set("updated_at = NOW()").
		Where("id IN (?)", bun.In(requestIDs)).
		Where("status IN (?, ?)", models.BookingRequestStatusPending, models.BookingRequestStatusApproved).
		Where("NOT EXISTS (SELECT 1 FROM room_bookings AS remaining WHERE remaining.booking_request_id = br.id)").Exec(ctx); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("cancel released booking requests: %w", err))
	}
	return nil
}

func (s *Service) releaseDeletedEventBookings(ctx context.Context, eventID int32) error {
	var requests []models.BookingRequest
	if err := s.db.NewSelect().Model(&requests).
		Where("event_id = ? OR id IN (SELECT booking_request_id FROM room_bookings WHERE event_id = ?)", eventID, eventID).
		OrderExpr("id ASC").For("UPDATE").Scan(ctx); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("lock deleted event booking requests: %w", err))
	}
	if _, err := s.db.NewDelete().Model((*models.RoomBooking)(nil)).Where("event_id = ?", eventID).Exec(ctx); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("release deleted event reservations: %w", err))
	}
	ids := make([]int32, 0, len(requests))
	for _, request := range requests {
		ids = append(ids, request.ID)
	}
	return s.cancelUnreservedBookingRequests(ctx, ids)
}
