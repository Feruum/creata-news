//go:build integration

package rooms

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	eventservice "github.com/studyverse/ems-backend/internal/services/events"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

type bookingReplacementFixture struct {
	db         *bun.DB
	svc        *Service
	staffCtx   context.Context
	creatorCtx context.Context
	event      models.Event
	room       models.Room
	original   *roomsv1.CreateBookingRequestResponse
}

func newBookingReplacementFixture(t *testing.T) bookingReplacementFixture {
	t.Helper()
	db := testsupport.ClonedDB(t)
	creator := seedAuthUser(t, db, "booking-replacement-creator")
	seedAuthUser(t, db, "booking-replacement-staff")
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id, role) VALUES (?, 'none')", creator); err != nil {
		t.Fatal(err)
	}
	var typeID, orgID int32
	if err := db.NewRaw("INSERT INTO organization_types(title) VALUES ('Booking replacement type') RETURNING id").Scan(ctx, &typeID); err != nil {
		t.Fatal(err)
	}
	if err := db.NewRaw("INSERT INTO organizations(title, organization_type_id) VALUES ('Booking replacement organization', ?) RETURNING id", typeID).Scan(ctx, &orgID); err != nil {
		t.Fatal(err)
	}
	room := seedRoom(t, db, "BOOKING-EDIT-101", "Booking replacement room")
	event := seedEvent(t, db, creator, orgID, "Published booking replacement event")
	event.StructuredLocation = []byte(fmt.Sprintf(`{"type":"campus_room","room_id":%q}`, room.ID))
	if _, err := db.NewUpdate().Model(&event).Column("structured_location").WherePK().Returning("*").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
	f := bookingReplacementFixture{
		db: db, svc: svc, staffCtx: authedContext("booking-replacement-staff"),
		creatorCtx: authedContext("booking-replacement-creator"), event: event, room: room,
	}
	original, err := svc.CreateBookingRequest(f.staffCtx, f.request(event.EndTime))
	if err != nil {
		t.Fatalf("create original approved reservation: %v", err)
	}
	if !original.Msg.AutoApproved || original.Msg.Booking == nil {
		t.Fatalf("staff original reservation must be approved: %+v", original.Msg)
	}
	f.original = original.Msg
	return f
}

func (f bookingReplacementFixture) request(end time.Time) *connect.Request[roomsv1.CreateBookingRequestRequest] {
	return connect.NewRequest(&roomsv1.CreateBookingRequestRequest{
		RoomId: f.room.ID, EventId: &f.event.ID, OrganizationId: &f.event.OrganizationID,
		Title: f.event.Title, StartTime: f.event.StartTime.Format(time.RFC3339), EndTime: end.Format(time.RFC3339),
	})
}

func (f bookingReplacementFixture) assertOriginalPreserved(t *testing.T) {
	t.Helper()
	var original models.RoomBooking
	if err := f.db.NewSelect().Model(&original).Where("id = ?", f.original.Booking.Id).Scan(f.staffCtx); err != nil {
		t.Fatalf("original reservation must remain until event save: %v", err)
	}
	if !original.StartTime.Equal(f.event.StartTime) || !original.EndTime.Equal(f.event.EndTime) {
		t.Fatalf("original reservation changed before event save: %+v", original)
	}
	var request models.BookingRequest
	if err := f.db.NewSelect().Model(&request).Where("id = ?", f.original.BookingRequest.Id).Scan(f.staffCtx); err != nil {
		t.Fatal(err)
	}
	if request.Status != models.BookingRequestStatusApproved {
		t.Fatalf("original reservation request status = %s, want approved", request.Status)
	}
	var event models.Event
	if err := f.db.NewSelect().Model(&event).Where("id = ?", f.event.ID).Scan(f.staffCtx); err != nil {
		t.Fatal(err)
	}
	if event.Status != models.EventStatusPublished || !event.StartTime.Equal(f.event.StartTime) ||
		!event.EndTime.Equal(f.event.EndTime) || string(event.StructuredLocation) != string(f.event.StructuredLocation) {
		t.Fatalf("published event changed during booking approval: %+v", event)
	}
}

func TestLinkedBookingReplacementPreservesOriginalUntilEventSave(t *testing.T) {
	for _, staff := range []bool{false, true} {
		t.Run(fmt.Sprintf("staff_%t", staff), func(t *testing.T) {
			f := newBookingReplacementFixture(t)
			ctx := f.creatorCtx
			if staff {
				ctx = f.staffCtx
			}
			end := f.event.EndTime.Add(30 * time.Minute)
			endText := end.Format(time.RFC3339)
			events := eventservice.NewService(f.db, nil, f.svc.authz, nil, nil, "", "")
			update := &eventsv1.UpdateEventRequest{Id: f.event.ID, EndTime: &endText}
			created, err := f.svc.CreateBookingRequest(ctx, f.request(end))
			if err != nil {
				t.Fatalf("request same-room extension: %v", err)
			}
			if created.Msg.AutoApproved != staff {
				t.Fatalf("auto approval = %t, want %t for existing role policy", created.Msg.AutoApproved, staff)
			}
			f.assertOriginalPreserved(t)
			booking := created.Msg.Booking
			if !staff {
				if created.Msg.BookingRequest.Status != roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_PENDING || booking != nil {
					t.Fatalf("nonstaff extension must wait for review: %+v", created.Msg)
				}
				if _, err := events.UpdateEvent(f.creatorCtx, connect.NewRequest(update)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
					t.Fatalf("pending replacement must block published event save: %v", err)
				}
				f.assertOriginalPreserved(t)
				approved, err := f.svc.ApproveBookingRequest(f.staffCtx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: created.Msg.BookingRequest.Id}))
				if err != nil {
					t.Fatalf("approve same-room extension with original reservation present: %v", err)
				}
				booking = approved.Msg.Booking
			}
			if booking == nil || booking.EventId == nil || *booking.EventId != f.event.ID || booking.EndTime != end.Format(time.RFC3339) {
				t.Fatalf("replacement reservation was not created for proposed event schedule: %+v", booking)
			}
			f.assertOriginalPreserved(t)
			count, err := f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("event_id = ?", f.event.ID).Count(ctx)
			if err != nil || count != 2 {
				t.Fatalf("want original and proposed reservations until save: count=%d, err=%v", count, err)
			}
			saved, err := events.UpdateEvent(f.creatorCtx, connect.NewRequest(update))
			if err != nil {
				t.Fatalf("save approved same-room extension: %v", err)
			}
			if saved.Msg.Event.EndTime != endText || saved.Msg.Event.Status != eventsv1.EventStatus_EVENT_STATUS_PUBLISHED {
				t.Fatalf("approved schedule was not saved as published: %+v", saved.Msg.Event)
			}
			exists, err := f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("id = ?", f.original.Booking.Id).Exists(ctx)
			if err != nil || exists {
				t.Fatalf("successful event save must release superseded reservation: exists=%t, err=%v", exists, err)
			}
			var originalRequest models.BookingRequest
			if err := f.db.NewSelect().Model(&originalRequest).Where("id = ?", f.original.BookingRequest.Id).Scan(ctx); err != nil {
				t.Fatal(err)
			}
			if originalRequest.Status != models.BookingRequestStatusCancelled {
				t.Fatalf("superseded request status = %s, want cancelled after event save", originalRequest.Status)
			}
			exists, err = f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("id = ?", booking.Id).Exists(ctx)
			if err != nil || !exists {
				t.Fatalf("successful event save must preserve approved replacement: exists=%t, err=%v", exists, err)
			}
		})
	}
}

func TestLinkedBookingReplacementKeepsOtherReservationsConflicting(t *testing.T) {
	for _, linked := range []bool{false, true} {
		t.Run(fmt.Sprintf("other_reservation_linked_%t", linked), func(t *testing.T) {
			f := newBookingReplacementFixture(t)
			end := f.event.EndTime.Add(30 * time.Minute)
			other := models.RoomBooking{
				RoomID: f.room.ID, CreatedBy: f.event.UserID, Title: "Other occupancy",
				StartTime: f.event.EndTime, EndTime: end,
			}
			if linked {
				event := seedEvent(t, f.db, f.event.UserID, f.event.OrganizationID, "Other event")
				other.EventID = &event.ID
			}
			if _, err := f.db.NewInsert().Model(&other).Returning("*").Exec(f.staffCtx); err != nil {
				t.Fatal(err)
			}
			created, err := f.svc.CreateBookingRequest(f.staffCtx, f.request(end))
			if err != nil {
				t.Fatalf("staff may submit a conflicting request for review: %v", err)
			}
			if created.Msg.AutoApproved || created.Msg.Booking != nil || created.Msg.BookingRequest.Status != roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_PENDING {
				t.Fatalf("other occupancy must prevent staff auto approval: %+v", created.Msg)
			}
			_, err = f.svc.ApproveBookingRequest(f.staffCtx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: created.Msg.BookingRequest.Id}))
			if connect.CodeOf(err) != connect.CodeFailedPrecondition {
				t.Fatalf("other occupancy must block approval: %v", err)
			}
			f.assertOriginalPreserved(t)
			var request models.BookingRequest
			if err := f.db.NewSelect().Model(&request).Where("id = ?", created.Msg.BookingRequest.Id).Scan(f.staffCtx); err != nil {
				t.Fatal(err)
			}
			if request.Status != models.BookingRequestStatusPending {
				t.Fatalf("blocked request status = %s, want pending", request.Status)
			}
			count, err := f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("room_id = ?", f.room.ID).Count(f.staffCtx)
			if err != nil || count != 2 {
				t.Fatalf("blocked approval changed reservations: count=%d, err=%v", count, err)
			}
		})
	}
}

func TestDeleteEventReleasesOriginalAndApprovedReplacement(t *testing.T) {
	f := newBookingReplacementFixture(t)
	created, err := f.svc.CreateBookingRequest(f.staffCtx, f.request(f.event.EndTime.Add(30*time.Minute)))
	if err != nil || !created.Msg.AutoApproved || created.Msg.Booking == nil {
		t.Fatalf("approve overlapping replacement: response=%v, err=%v", created, err)
	}
	f.assertOriginalPreserved(t)
	events := eventservice.NewService(f.db, nil, f.svc.authz, nil, nil, "", "")
	if _, err := events.DeleteEvent(f.creatorCtx, connect.NewRequest(&eventsv1.DeleteEventRequest{Id: f.event.ID})); err != nil {
		t.Fatalf("delete event with overlapping approved reservations under normal FK/trigger checks: %v", err)
	}
	exists, err := f.db.NewSelect().Model((*models.Event)(nil)).Where("id = ?", f.event.ID).Exists(f.creatorCtx)
	if err != nil || exists {
		t.Fatalf("deleted event still exists: exists=%t, err=%v", exists, err)
	}
	bookingIDs := []int32{f.original.Booking.Id, created.Msg.Booking.Id}
	count, err := f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("id IN (?)", bun.In(bookingIDs)).Count(f.creatorCtx)
	if err != nil || count != 0 {
		t.Fatalf("event deletion retained original or proposed occupancy: count=%d, err=%v", count, err)
	}
	requestIDs := []int32{f.original.BookingRequest.Id, created.Msg.BookingRequest.Id}
	var requests []models.BookingRequest
	if err := f.db.NewSelect().Model(&requests).Where("id IN (?)", bun.In(requestIDs)).Scan(f.creatorCtx); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 {
		t.Fatalf("event deletion lost booking request history: %+v", requests)
	}
	for _, request := range requests {
		if request.Status != models.BookingRequestStatusCancelled {
			t.Fatalf("event deletion retained approved request %d: %s", request.ID, request.Status)
		}
	}
}

// A cancelled query context simulates a failed occupancy read without aborting
// the transaction itself, so ignoring the error would incorrectly approve it.
type cancelBookingConflictRead struct {
	table  string
	failed bool
}

func (h *cancelBookingConflictRead) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	if !h.failed && strings.HasPrefix(event.Query, "SELECT ") && strings.Contains(event.Query, `FROM "`+h.table+`"`) && strings.Contains(event.Query, "OVERLAPS") {
		h.failed = true
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		return cancelled
	}
	return ctx
}

func (*cancelBookingConflictRead) AfterQuery(context.Context, *bun.QueryEvent) {}

func TestBookingConflictReadFailureDoesNotApproveOrPersistNewRequest(t *testing.T) {
	for _, action := range []string{"create_pending_conflicts", "create_booking_conflicts", "approve_booking_conflicts"} {
		t.Run(action, func(t *testing.T) {
			f := newBookingReplacementFixture(t)
			proposal := f.request(f.event.EndTime.Add(30 * time.Minute))
			var pendingID int32
			if action == "approve_booking_conflicts" {
				created, err := f.svc.CreateBookingRequest(f.creatorCtx, proposal)
				if err != nil {
					t.Fatal(err)
				}
				pendingID = created.Msg.BookingRequest.Id
			}
			hook := &cancelBookingConflictRead{table: "room_bookings"}
			if action == "create_pending_conflicts" {
				hook.table = "booking_requests"
			}
			f.db.AddQueryHook(hook)
			var err error
			if pendingID != 0 {
				_, err = f.svc.ApproveBookingRequest(f.staffCtx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: pendingID}))
			} else {
				_, err = f.svc.CreateBookingRequest(f.staffCtx, proposal)
			}
			if !hook.failed {
				t.Fatal("conflict read failure was not injected")
			}
			if connect.CodeOf(err) != connect.CodeInternal {
				t.Fatalf("failed conflict read must be internal error: %v", err)
			}
			f.assertOriginalPreserved(t)
			count, err := f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("event_id = ?", f.event.ID).Count(f.staffCtx)
			if err != nil || count != 1 {
				t.Fatalf("failed conflict read created a booking: count=%d, err=%v", count, err)
			}
			wantRequests := 1
			if pendingID != 0 {
				wantRequests++
				var request models.BookingRequest
				if err := f.db.NewSelect().Model(&request).Where("id = ?", pendingID).Scan(f.staffCtx); err != nil {
					t.Fatal(err)
				}
				if request.Status != models.BookingRequestStatusPending {
					t.Fatalf("failed conflict read approved request: %s", request.Status)
				}
			}
			count, err = f.db.NewSelect().Model((*models.BookingRequest)(nil)).Where("event_id = ?", f.event.ID).Count(f.staffCtx)
			if err != nil || count != wantRequests {
				t.Fatalf("failed conflict read persisted a new request: count=%d, want=%d, err=%v", count, wantRequests, err)
			}
		})
	}
}

func TestBookingInsertConflictReturnsFailedPrecondition(t *testing.T) {
	for _, staff := range []bool{false, true} {
		t.Run(fmt.Sprintf("staff_%t", staff), func(t *testing.T) {
			f := newBookingReplacementFixture(t)
			room := seedRoom(t, f.db, "BOOKING-EDIT-102", "Alternative booking room")
			proposal := f.request(f.event.EndTime.Add(30 * time.Minute))
			proposal.Msg.RoomId = room.ID
			var pendingID int32
			if !staff {
				created, err := f.svc.CreateBookingRequest(f.creatorCtx, proposal)
				if err != nil {
					t.Fatal(err)
				}
				pendingID = created.Msg.BookingRequest.Id
			}
			// The occupancy trigger remains authoritative if a conflicting booking
			// wins the slot after the service's preliminary conflict check.
			if _, err := f.db.ExecContext(f.staffCtx, `CREATE FUNCTION inject_booking_conflict() RETURNS TRIGGER LANGUAGE plpgsql AS $$
				BEGIN RAISE EXCEPTION USING ERRCODE = '23P01', MESSAGE = 'injected concurrent occupancy'; END;
				$$;
				CREATE TRIGGER inject_booking_conflict BEFORE INSERT ON room_bookings
				FOR EACH ROW EXECUTE FUNCTION inject_booking_conflict()`); err != nil {
				t.Fatal(err)
			}
			var err error
			if staff {
				_, err = f.svc.CreateBookingRequest(f.staffCtx, proposal)
			} else {
				_, err = f.svc.ApproveBookingRequest(f.staffCtx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: pendingID}))
			}
			if connect.CodeOf(err) != connect.CodeFailedPrecondition {
				t.Fatalf("database occupancy conflict must be failed_precondition: %v", err)
			}
			f.assertOriginalPreserved(t)
			if pendingID != 0 {
				var request models.BookingRequest
				if err := f.db.NewSelect().Model(&request).Where("id = ?", pendingID).Scan(f.staffCtx); err != nil {
					t.Fatal(err)
				}
				if request.Status != models.BookingRequestStatusPending {
					t.Fatalf("database occupancy conflict changed request status: %s", request.Status)
				}
			}
		})
	}
}

type bookingEventLockSignal struct {
	started chan struct{}
}

func (h *bookingEventLockSignal) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	if strings.Contains(event.Query, `FROM "events"`) && strings.Contains(event.Query, "FOR KEY SHARE") {
		close(h.started)
	}
	return ctx
}

func (*bookingEventLockSignal) AfterQuery(context.Context, *bun.QueryEvent) {}

func TestLinkedBookingApprovalLocksEventBeforeRequest(t *testing.T) {
	for _, auto := range []bool{false, true} {
		t.Run(fmt.Sprintf("auto_%t", auto), func(t *testing.T) {
			f := newBookingReplacementFixture(t)
			proposal := f.request(f.event.EndTime.Add(30 * time.Minute))
			requestID := f.original.BookingRequest.Id
			if !auto {
				created, err := f.svc.CreateBookingRequest(f.creatorCtx, proposal)
				if err != nil {
					t.Fatal(err)
				}
				requestID = created.Msg.BookingRequest.Id
			}
			ctx, cancel := context.WithTimeout(f.staffCtx, 10*time.Second)
			defer cancel()
			eventTx, err := f.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = eventTx.Rollback() }()
			var id int32
			if err := eventTx.NewRaw("SELECT id FROM events WHERE id = ? FOR UPDATE", f.event.ID).Scan(ctx, &id); err != nil {
				t.Fatal(err)
			}
			hook := &bookingEventLockSignal{started: make(chan struct{})}
			f.db.AddQueryHook(hook)
			done := make(chan error, 1)
			go func() {
				var err error
				if auto {
					_, err = f.svc.CreateBookingRequest(ctx, proposal)
				} else {
					_, err = f.svc.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: requestID}))
				}
				done <- err
			}()
			select {
			case <-hook.started:
			case err := <-done:
				t.Fatalf("approval finished without first locking linked event: %v", err)
			case <-ctx.Done():
				t.Fatal("approval never attempted to lock the linked event")
			}
			if err := eventTx.NewRaw("SELECT id FROM booking_requests WHERE id = ? FOR SHARE NOWAIT", requestID).Scan(ctx, &id); err != nil {
				t.Fatalf("approval locked request while waiting for event, reversing event-save lock order: %v", err)
			}
			var roomID string
			if err := eventTx.NewRaw("SELECT id FROM rooms WHERE id = ? FOR UPDATE NOWAIT", f.room.ID).Scan(ctx, &roomID); err != nil {
				t.Fatalf("approval locked room while waiting for event: %v", err)
			}
			if err := eventTx.Commit(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("approval after event save releases its lock: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("approval did not complete after event lock was released")
			}
			f.assertOriginalPreserved(t)
		})
	}
}
