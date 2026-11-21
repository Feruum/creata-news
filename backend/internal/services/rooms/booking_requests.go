package rooms

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	"github.com/uptrace/bun"

	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
)

// "BOOK" reserves a single-bigint advisory namespace separate from search and
// the two-int organization member locks. Only linked booking creation uses it.
const bookingRequestLockNamespace int64 = 0x424F4F4B << 32

func (s *Service) CreateBookingRequest(ctx context.Context, req *connect.Request[roomsv1.CreateBookingRequestRequest]) (*connect.Response[roomsv1.CreateBookingRequestResponse], error) {
	return s.createBookingRequest(ctx, req, true)
}

// RequestReviewedBooking uses the normal requester permissions and forces human review,
// even when the actor is staff. The caller commits the draft location and request together.
func (s *Service) RequestReviewedBooking(ctx context.Context, tx bun.Tx, req *connect.Request[roomsv1.CreateBookingRequestRequest]) (*connect.Response[roomsv1.CreateBookingRequestResponse], error) {
	scoped := *s
	scoped.db = tx
	scoped.authz = s.authz.InTransaction(tx)
	return scoped.createBookingRequest(ctx, req, false)
}

// LockEventBooking must precede event and room row locks in workflows that create linked requests.
func LockEventBooking(ctx context.Context, tx bun.Tx, eventID int32) error {
	key := bookingRequestLockNamespace | int64(uint32(eventID))
	_, err := tx.NewRaw("SELECT pg_advisory_xact_lock(?::bigint)", key).Exec(ctx)
	return err
}

func (s *Service) createBookingRequest(ctx context.Context, req *connect.Request[roomsv1.CreateBookingRequestRequest], allowAutoApproval bool) (*connect.Response[roomsv1.CreateBookingRequestResponse], error) {
	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}

	normalizedRoomID := roomid.Normalize(req.Msg.RoomId)
	if normalizedRoomID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room_id is required"))
	}

	var room models.Room
	if err := s.db.NewSelect().Model(&room).Column("id", "is_active", "is_bookable").Where("id = ?", normalizedRoomID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("room not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load room booking rules: %w", err))
	}
	if !room.IsActive || !room.IsBookable {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("room is not available for booking"))
	}

	priority := models.BookingPriorityIndividual
	var orgID *int32

	if req.Msg.OrganizationId != nil {
		orgID = req.Msg.OrganizationId
		if !s.authz.CheckOrgPermission(ctx, localUserID, *req.Msg.OrganizationId, "book_rooms") {
			if req.Msg.EventId == nil {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("no permission to book rooms for this organization"))
			}
			if err := s.requireEventLinkedBookingPermission(ctx, *req.Msg.OrganizationId, *req.Msg.EventId); err != nil {
				return nil, err
			}
		}
		priority = s.determineOrgPriority(ctx, *req.Msg.OrganizationId)
	}

	startTime, err := time.Parse(time.RFC3339, req.Msg.StartTime)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid start_time format"))
	}
	endTime, err := time.Parse(time.RFC3339, req.Msg.EndTime)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid end_time format"))
	}
	if !endTime.After(startTime) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("end_time must be after start_time"))
	}

	var conflicts []models.BookingRequest
	if err := s.db.NewSelect().Model(&conflicts).
		Where("room_id = ?", normalizedRoomID).
		Where("status = ?", models.BookingRequestStatusPending).
		Where("(start_time, end_time) OVERLAPS (?, ?)", startTime, endTime).
		Scan(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check pending booking requests: %w", err))
	}

	bookingReq := models.BookingRequest{
		RoomID:         normalizedRoomID,
		RequesterID:    localUserID,
		OrganizationID: orgID,
		Priority:       priority,
		Status:         models.BookingRequestStatusPending,
		Title:          req.Msg.Title,
		StartTime:      startTime,
		EndTime:        endTime,
	}

	if req.Msg.Description != nil {
		bookingReq.Description = req.Msg.Description
	}
	if req.Msg.EventId != nil {
		bookingReq.EventID = req.Msg.EventId
	}

	// Complete network authorization before transactional booking/FK locks.
	isStaff := allowAutoApproval && s.authz.CheckPlatformStaff(ctx, localUserID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// A nested Bun transaction is a savepoint. Rolling it back after a successful
	// RELEASE would poison the caller's still-open PostgreSQL transaction.
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if req.Msg.EventId != nil {
		if err := LockEventBooking(ctx, tx, *req.Msg.EventId); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("lock linked booking request: %w", err))
		}
		exists, err := tx.NewSelect().Model((*models.BookingRequest)(nil)).
			Where("event_id = ?", *req.Msg.EventId).
			Where("room_id = ?", normalizedRoomID).
			Where("start_time = ? AND end_time = ?", startTime, endTime).
			Where("status IN (?, ?)", models.BookingRequestStatusPending, models.BookingRequestStatusApproved).
			Exists(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check linked booking request: %w", err))
		}
		if exists {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("an active booking request already exists for this event, room, and time range"))
		}
	}

	if isStaff {
		// Take the event lock before the room lock, and acquire the room's final
		// lock mode before request insertion acquires a room FK KEY SHARE lock.
		if bookingReq.EventID != nil {
			var event models.Event
			if err := tx.NewSelect().Model(&event).Column("id").Where("id = ?", *bookingReq.EventID).
				For("KEY SHARE").Scan(ctx); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("linked event not found"))
				}
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("lock linked event: %w", err))
			}
		}
		if err := tx.NewSelect().Model(&room).Column("id", "is_active", "is_bookable").
			Where("id = ?", normalizedRoomID).For("UPDATE").Scan(ctx); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("room not found"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("lock room booking rules: %w", err))
		}
		if !room.IsActive || !room.IsBookable {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("room is not available for booking"))
		}
	}

	_, err = tx.NewInsert().Model(&bookingReq).Returning("*").Exec(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "CreateBookingRequest query failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create booking request: %w", err))
	}

	protoConflicts := lo.Map(conflicts, func(conflict models.BookingRequest, _ int) *roomsv1.BookingRequest {
		return dbBookingRequestToProto(conflict)
	})

	response := &roomsv1.CreateBookingRequestResponse{
		BookingRequest:    dbBookingRequestToProto(bookingReq),
		ExistingConflicts: protoConflicts,
		AutoApproved:      false,
	}

	if isStaff {
		var existingBookings []models.RoomBooking
		bookingQuery := tx.NewSelect().Model(&existingBookings).
			Where("room_id = ?", normalizedRoomID).
			Where("(start_time, end_time) OVERLAPS (?, ?)", startTime, endTime)
		if bookingReq.EventID != nil {
			bookingQuery = bookingQuery.Where("event_id IS DISTINCT FROM ?", *bookingReq.EventID)
		}
		if err := bookingQuery.Scan(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing bookings: %w", err))
		}

		if len(existingBookings) == 0 {
			now := time.Now()
			_, updateErr := tx.NewUpdate().Model((*models.BookingRequest)(nil)).
				Set("status = ?", models.BookingRequestStatusApproved).
				Set("reviewed_by = ?", localUserID).
				Set("reviewed_at = ?", now).
				Set("updated_at = NOW()").
				Where("id = ?", bookingReq.ID).
				Returning("*").
				Exec(ctx, &bookingReq)
			if updateErr != nil {
				slog.WarnContext(ctx, "Auto-approve: failed to update request status", "error", updateErr, "requestId", bookingReq.ID)
				return nil, connect.NewError(connect.CodeInternal, updateErr)
			}

			booking := models.RoomBooking{
				RoomID:           normalizedRoomID,
				BookingRequestID: &bookingReq.ID,
				CreatedBy:        localUserID,
				Title:            req.Msg.Title,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			if req.Msg.EventId != nil {
				booking.EventID = req.Msg.EventId
			}

			_, bookingErr := tx.NewInsert().Model(&booking).Returning("*").Exec(ctx)
			if bookingErr != nil {
				slog.ErrorContext(ctx, "Auto-approve: failed to create booking", "error", bookingErr, "requestId", bookingReq.ID)
				return nil, roomBookingInsertError(bookingErr)
			}

			response.BookingRequest = dbBookingRequestToProto(bookingReq)
			response.Booking = dbBookingToProto(booking)
			response.AutoApproved = true
			slog.InfoContext(ctx, "Auto-approved booking request", "requestId", bookingReq.ID, "userId", localUserID)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	committed = true

	return connect.NewResponse(response), nil
}

func (s *Service) requireEventLinkedBookingPermission(ctx context.Context, organizationID, eventID int32) error {
	var event models.Event
	err := s.db.NewSelect().
		Model(&event).
		Column("id", "organization_id").
		Where("id = ?", eventID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return connect.NewError(connect.CodeNotFound, fmt.Errorf("linked event not found"))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load linked event: %w", err))
	}

	if event.OrganizationID != organizationID {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("linked event does not belong to the selected organization"))
	}

	if _, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", eventID), "edit"); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetBookingRequest(ctx context.Context, req *connect.Request[roomsv1.GetBookingRequestRequest]) (*connect.Response[roomsv1.GetBookingRequestResponse], error) {
	if _, err := s.authz.RequirePlatformStaff(ctx); err != nil {
		return nil, err
	}

	var row bookingRequestRow
	err := s.db.NewSelect().
		TableExpr("booking_requests AS br").
		ColumnExpr("br.*").
		ColumnExpr("r.name AS room_name").
		ColumnExpr("r.block AS room_block").
		ColumnExpr("r.room_number AS room_room_number").
		ColumnExpr("r.floor AS room_floor").
		ColumnExpr("COALESCE(u.first_name || ' ' || u.last_name, u.username) AS requester_name").
		ColumnExpr("o.title AS organization_name").
		Join("LEFT JOIN rooms r ON r.id = br.room_id").
		Join("LEFT JOIN users u ON u.id = br.requester_id").
		Join("LEFT JOIN organizations o ON o.id = br.organization_id").
		Where("br.id = ?", req.Msg.Id).
		Scan(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("booking request not found"))
		}
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("booking request not found"))
	}

	return connect.NewResponse(&roomsv1.GetBookingRequestResponse{
		BookingRequest: bookingRequestRowToProto(row),
	}), nil
}

func (s *Service) ListBookingRequests(ctx context.Context, req *connect.Request[roomsv1.ListBookingRequestsRequest]) (*connect.Response[roomsv1.ListBookingRequestsResponse], error) {
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

	var rows []bookingRequestRow
	query := s.db.NewSelect().
		TableExpr("booking_requests AS br").
		ColumnExpr("br.*").
		ColumnExpr("r.name AS room_name").
		ColumnExpr("r.block AS room_block").
		ColumnExpr("r.room_number AS room_room_number").
		ColumnExpr("r.floor AS room_floor").
		ColumnExpr("COALESCE(u.first_name || ' ' || u.last_name, u.username) AS requester_name").
		ColumnExpr("o.title AS organization_name").
		Join("LEFT JOIN rooms r ON r.id = br.room_id").
		Join("LEFT JOIN users u ON u.id = br.requester_id").
		Join("LEFT JOIN organizations o ON o.id = br.organization_id")

	if req.Msg.Status != nil && *req.Msg.Status != roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_UNSPECIFIED {
		query = query.Where("br.status = ?", protoToDBBookingStatus(*req.Msg.Status))
	}
	if req.Msg.RoomId != nil {
		normalizedRoomID := roomid.Normalize(*req.Msg.RoomId)
		if normalizedRoomID == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room_id cannot be empty"))
		}
		query = query.Where("br.room_id = ?", normalizedRoomID)
	}
	if req.Msg.OrganizationId != nil {
		query = query.Where("br.organization_id = ?", *req.Msg.OrganizationId)
	}
	if req.Msg.RequesterId != nil {
		query = query.Where("br.requester_id = ?", *req.Msg.RequesterId)
	}
	if req.Msg.FromDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.Msg.FromDate)
		query = query.Where("br.start_time >= ?", t)
	}
	if req.Msg.ToDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.Msg.ToDate)
		query = query.Where("br.end_time <= ?", t)
	}

	query = query.OrderExpr("br.created_at DESC").Limit(int(limit)).Offset(int(offset))

	err := query.Scan(ctx, &rows)
	if err != nil {
		slog.ErrorContext(ctx, "ListBookingRequests failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list booking requests: %w", err))
	}

	countQuery := s.db.NewSelect().TableExpr("booking_requests AS br")
	if req.Msg.Status != nil && *req.Msg.Status != roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_UNSPECIFIED {
		countQuery = countQuery.Where("br.status = ?", protoToDBBookingStatus(*req.Msg.Status))
	}
	if req.Msg.RoomId != nil {
		normalizedRoomID := roomid.Normalize(*req.Msg.RoomId)
		if normalizedRoomID == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("room_id cannot be empty"))
		}
		countQuery = countQuery.Where("br.room_id = ?", normalizedRoomID)
	}
	if req.Msg.OrganizationId != nil {
		countQuery = countQuery.Where("br.organization_id = ?", *req.Msg.OrganizationId)
	}
	if req.Msg.RequesterId != nil {
		countQuery = countQuery.Where("br.requester_id = ?", *req.Msg.RequesterId)
	}
	if req.Msg.FromDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.Msg.FromDate)
		countQuery = countQuery.Where("br.start_time >= ?", t)
	}
	if req.Msg.ToDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.Msg.ToDate)
		countQuery = countQuery.Where("br.end_time <= ?", t)
	}
	total, _ := countQuery.Count(ctx)

	protoRequests := lo.Map(rows, func(row bookingRequestRow, _ int) *roomsv1.BookingRequest {
		return bookingRequestRowToProto(row)
	})

	return connect.NewResponse(&roomsv1.ListBookingRequestsResponse{
		BookingRequests: protoRequests,
		Total:           int32(total),
	}), nil
}

func (s *Service) ApproveBookingRequest(ctx context.Context, req *connect.Request[roomsv1.ApproveBookingRequestRequest]) (*connect.Response[roomsv1.ApproveBookingRequestResponse], error) {
	localUserID, err := s.authz.RequirePlatformStaff(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	var bookingReq models.BookingRequest
	err = tx.NewSelect().Model(&bookingReq).Where("id = ?", req.Msg.Id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("booking request not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load booking request: %w", err))
	}

	linkedEventID := bookingReq.EventID
	if req.Msg.EventId != nil && linkedEventID != nil && *req.Msg.EventId != *linkedEventID {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("booking request is already linked to a different event"))
	}
	eventID := linkedEventID
	if req.Msg.EventId != nil {
		eventID = req.Msg.EventId
	}
	// Event saves lock the event before its booking requests. Follow that order
	// before changing approval state or acquiring the reservation's event FK lock.
	if eventID != nil {
		var event models.Event
		if err := tx.NewSelect().Model(&event).Column("id").Where("id = ?", *eventID).
			For("KEY SHARE").Scan(ctx); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("linked event not found"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("lock linked event: %w", err))
		}
	}
	err = tx.NewSelect().Model(&bookingReq).Where("id = ?", req.Msg.Id).
		Where("event_id IS NOT DISTINCT FROM ?", linkedEventID).For("UPDATE").Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("booking request changed; retry approval"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("lock booking request: %w", err))
	}

	if bookingReq.Status != models.BookingRequestStatusPending {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("booking request is not pending"))
	}

	// Keep the current room rules stable and match the occupancy trigger's lock
	// mode, so concurrent approvals cannot deadlock while upgrading a SHARE lock.
	var room models.Room
	if err := tx.NewSelect().Model(&room).Column("id", "is_active", "is_bookable").
		Where("id = ?", bookingReq.RoomID).For("UPDATE").Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("room not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load room booking rules: %w", err))
	}
	if !room.IsActive || !room.IsBookable {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("room is not available for booking"))
	}

	var existingBookings []models.RoomBooking
	bookingQuery := tx.NewSelect().Model(&existingBookings).
		Where("room_id = ?", bookingReq.RoomID).
		Where("(start_time, end_time) OVERLAPS (?, ?)", bookingReq.StartTime, bookingReq.EndTime)
	if eventID != nil {
		bookingQuery = bookingQuery.Where("event_id IS DISTINCT FROM ?", *eventID)
	}
	if err := bookingQuery.Scan(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing bookings: %w", err))
	}
	if len(existingBookings) > 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("time slot is no longer available"))
	}

	now := time.Now()
	_, err = tx.NewUpdate().Model((*models.BookingRequest)(nil)).
		Set("status = ?", models.BookingRequestStatusApproved).
		Set("event_id = ?", eventID).
		Set("reviewed_by = ?", localUserID).
		Set("reviewed_at = ?", now).
		Set("updated_at = NOW()").
		Where("id = ?", req.Msg.Id).
		Returning("*").
		Exec(ctx, &bookingReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update request status: %w", err))
	}

	booking := models.RoomBooking{
		RoomID:           bookingReq.RoomID,
		BookingRequestID: &req.Msg.Id,
		EventID:          eventID,
		CreatedBy:        localUserID,
		Title:            bookingReq.Title,
		StartTime:        bookingReq.StartTime,
		EndTime:          bookingReq.EndTime,
	}
	_, err = tx.NewInsert().Model(&booking).Returning("*").Exec(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "CreateRoomBooking failed", "error", err)
		return nil, roomBookingInsertError(err)
	}

	if publisher != nil {
		var eventID int32
		if booking.EventID != nil {
			eventID = *booking.EventID
		}
		eventTitle := bookingReq.Title
		if booking.EventID != nil {
			event := &models.Event{}
			if err := tx.NewSelect().Model(event).Where("id = ?", *booking.EventID).Scan(ctx); err == nil {
				eventTitle = event.Title
			}
		}

		var organizationID int32
		if bookingReq.OrganizationID != nil {
			organizationID = *bookingReq.OrganizationID
		}

		room := &models.Room{}
		roomName := ""
		if err := tx.NewSelect().Model(room).Where("id = ?", bookingReq.RoomID).Scan(ctx); err == nil {
			roomName = room.Name
		}

		if err := publisher.PublishBookingApproved(ctx, domainevents.BookingApprovedPayload{
			BookingRequestID: bookingReq.ID,
			EventID:          eventID,
			EventTitle:       eventTitle,
			OrganizationID:   organizationID,
			RoomID:           bookingReq.RoomID,
			RoomName:         roomName,
			ActorID:          localUserID,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&roomsv1.ApproveBookingRequestResponse{
		BookingRequest: dbBookingRequestToProto(bookingReq),
		Booking:        dbBookingToProto(booking),
	}), nil
}

func roomBookingInsertError(err error) error {
	// pgx exposes SQLState; Bun's pgdriver exposes PostgreSQL error fields.
	var sqlState interface{ SQLState() string }
	var fields interface{ Field(byte) string }
	if (errors.As(err, &sqlState) && sqlState.SQLState() == "23P01") ||
		(errors.As(err, &fields) && fields.Field('C') == "23P01") {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("time slot is no longer available"))
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create booking: %w", err))
}

func (s *Service) RejectBookingRequest(ctx context.Context, req *connect.Request[roomsv1.RejectBookingRequestRequest]) (*connect.Response[roomsv1.RejectBookingRequestResponse], error) {
	localUserID, err := s.authz.RequirePlatformStaff(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	var bookingReq models.BookingRequest
	err = tx.NewSelect().Model(&bookingReq).Where("id = ?", req.Msg.Id).For("UPDATE").Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("booking request not found"))
	}

	if bookingReq.Status != models.BookingRequestStatusPending {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("booking request is not pending"))
	}

	now := time.Now()
	_, err = tx.NewUpdate().Model((*models.BookingRequest)(nil)).
		Set("status = ?", models.BookingRequestStatusRejected).
		Set("reviewed_by = ?", localUserID).
		Set("reviewed_at = ?", now).
		Set("rejection_reason = ?", req.Msg.Reason).
		Set("updated_at = NOW()").
		Where("id = ?", req.Msg.Id).
		Returning("*").
		Exec(ctx, &bookingReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update request status: %w", err))
	}

	if publisher != nil {
		var eventID int32
		if bookingReq.EventID != nil {
			eventID = *bookingReq.EventID
		}
		eventTitle := bookingReq.Title
		if bookingReq.EventID != nil {
			event := &models.Event{}
			if err := tx.NewSelect().Model(event).Where("id = ?", *bookingReq.EventID).Scan(ctx); err == nil {
				eventTitle = event.Title
			}
		}

		var organizationID int32
		if bookingReq.OrganizationID != nil {
			organizationID = *bookingReq.OrganizationID
		}

		room := &models.Room{}
		roomName := ""
		if err := tx.NewSelect().Model(room).Where("id = ?", bookingReq.RoomID).Scan(ctx); err == nil {
			roomName = room.Name
		}

		if err := publisher.PublishBookingRejected(ctx, domainevents.BookingRejectedPayload{
			BookingRequestID: bookingReq.ID,
			EventID:          eventID,
			EventTitle:       eventTitle,
			OrganizationID:   organizationID,
			RoomID:           bookingReq.RoomID,
			RoomName:         roomName,
			ActorID:          localUserID,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&roomsv1.RejectBookingRequestResponse{
		BookingRequest: dbBookingRequestToProto(bookingReq),
	}), nil
}

func (s *Service) CancelBookingRequest(ctx context.Context, req *connect.Request[roomsv1.CancelBookingRequestRequest]) (*connect.Response[roomsv1.CancelBookingRequestResponse], error) {
	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("begin booking request cancellation: %w", err))
	}
	defer func() { _ = tx.Rollback() }()

	var bookingReq models.BookingRequest
	err = tx.NewSelect().Model(&bookingReq).Column("id").Where("id = ?", req.Msg.Id).
		Where("requester_id = ?", localUserID).For("UPDATE").Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return connect.NewResponse(&roomsv1.CancelBookingRequestResponse{Success: true}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("lock owned booking request: %w", err))
	}

	if _, err := tx.NewDelete().Model((*models.RoomBooking)(nil)).
		Where("booking_request_id = ?", bookingReq.ID).Exec(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to release request reservations: %w", err))
	}
	_, err = tx.NewUpdate().Model((*models.BookingRequest)(nil)).
		Set("status = ?", models.BookingRequestStatusCancelled).
		Set("updated_at = NOW()").
		Where("id = ?", bookingReq.ID).
		Where("requester_id = ?", localUserID).
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to cancel booking request: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("commit booking request cancellation: %w", err))
	}

	return connect.NewResponse(&roomsv1.CancelBookingRequestResponse{
		Success: true,
	}), nil
}

func (s *Service) GetBookingConflicts(ctx context.Context, req *connect.Request[roomsv1.GetBookingConflictsRequest]) (*connect.Response[roomsv1.GetBookingConflictsResponse], error) {
	_, err := s.authz.RequirePlatformStaff(ctx)
	if err != nil {
		return nil, err
	}

	var rows []conflictingRequestRow
	query := s.db.NewSelect().
		TableExpr("booking_requests AS br").
		ColumnExpr("br.*").
		ColumnExpr("r.name AS room_name").
		Join("JOIN rooms r ON r.id = br.room_id").
		Where("br.status = ?", models.BookingRequestStatusPending).
		Where("EXISTS (SELECT 1 FROM booking_requests br2 WHERE br2.room_id = br.room_id AND br2.id != br.id AND br2.status = ? AND (br.start_time, br.end_time) OVERLAPS (br2.start_time, br2.end_time))", models.BookingRequestStatusPending)

	if req.Msg.FromDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.Msg.FromDate)
		query = query.Where("br.start_time >= ?", t)
	}
	if req.Msg.ToDate != nil {
		t, _ := time.Parse(time.RFC3339, *req.Msg.ToDate)
		query = query.Where("br.end_time <= ?", t)
	}

	query = query.OrderExpr("br.room_id, br.start_time")

	err = query.Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get conflicts: %w", err))
	}

	conflictMap := make(map[string]*roomsv1.BookingConflict)
	for _, r := range rows {
		key := fmt.Sprintf("%s-%s-%s", r.RoomID, r.StartTime.Format(time.RFC3339), r.EndTime.Format(time.RFC3339))

		if _, exists := conflictMap[key]; !exists {
			conflictMap[key] = &roomsv1.BookingConflict{
				RoomId:              r.RoomID,
				StartTime:           r.StartTime.Format(time.RFC3339),
				EndTime:             r.EndTime.Format(time.RFC3339),
				ConflictingRequests: []*roomsv1.BookingRequest{},
			}
		}

		protoReq := dbBookingRequestToProto(r.BookingRequest)
		conflictMap[key].ConflictingRequests = append(conflictMap[key].ConflictingRequests, protoReq)
	}

	for _, conflict := range conflictMap {
		if len(conflict.ConflictingRequests) > 0 {
			var highestPriority *roomsv1.BookingRequest
			for _, r := range conflict.ConflictingRequests {
				if highestPriority == nil || r.Priority < highestPriority.Priority {
					highestPriority = r
				}
			}
			conflict.SuggestedWinner = highestPriority
		}
	}

	conflicts := make([]*roomsv1.BookingConflict, 0, len(conflictMap))
	for _, c := range conflictMap {
		conflicts = append(conflicts, c)
	}

	return connect.NewResponse(&roomsv1.GetBookingConflictsResponse{
		Conflicts: conflicts,
	}), nil
}
