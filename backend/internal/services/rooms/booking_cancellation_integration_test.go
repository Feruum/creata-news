//go:build integration

package rooms

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/models"
	eventservice "github.com/studyverse/ems-backend/internal/services/events"
)

func TestCancelBookingCancelsLinkedApprovedRequest(t *testing.T) {
	f := newBookingReplacementFixture(t)
	request := connect.NewRequest(&roomsv1.CancelRoomBookingRequest{Id: f.original.Booking.Id})
	response, err := f.svc.CancelBooking(f.staffCtx, request)
	if err != nil || !response.Msg.Success {
		t.Fatalf("cancel approved reservation: response=%v, err=%v", response, err)
	}
	var linked models.BookingRequest
	if err := f.db.NewSelect().Model(&linked).Where("id = ?", f.original.BookingRequest.Id).Scan(f.staffCtx); err != nil {
		t.Fatal(err)
	}
	if linked.Status != models.BookingRequestStatusCancelled {
		t.Fatalf("cancelled reservation retained publishable request status: %s", linked.Status)
	}
	exists, err := f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("id = ?", f.original.Booking.Id).Exists(f.staffCtx)
	if err != nil || exists {
		t.Fatalf("cancelled reservation still exists: exists=%t, err=%v", exists, err)
	}
	if _, err := f.svc.CancelBooking(f.staffCtx, request); err != nil {
		t.Fatalf("repeated cancellation must remain idempotent: %v", err)
	}
}

func TestCancelBookingPreservesUnlinkedReservationBehavior(t *testing.T) {
	f := newBookingReplacementFixture(t)
	unlinked := models.RoomBooking{
		RoomID: f.room.ID, CreatedBy: f.event.UserID, Title: "Individual reservation",
		StartTime: f.event.EndTime, EndTime: f.event.EndTime.Add(time.Hour),
	}
	if _, err := f.db.NewInsert().Model(&unlinked).Returning("*").Exec(f.staffCtx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.CancelBooking(f.staffCtx, connect.NewRequest(&roomsv1.CancelRoomBookingRequest{Id: unlinked.ID})); err != nil {
		t.Fatalf("cancel unlinked reservation: %v", err)
	}
	exists, err := f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("id = ?", unlinked.ID).Exists(f.staffCtx)
	if err != nil || exists {
		t.Fatalf("unlinked reservation was not removed: exists=%t, err=%v", exists, err)
	}
	f.assertOriginalPreserved(t)
}

func TestCancelledReplacementCannotBeSavedToPublishedEvent(t *testing.T) {
	f := newBookingReplacementFixture(t)
	end := f.event.EndTime.Add(30 * time.Minute)
	created, err := f.svc.CreateBookingRequest(f.staffCtx, f.request(end))
	if err != nil || !created.Msg.AutoApproved || created.Msg.Booking == nil {
		t.Fatalf("approve extension: response=%v, err=%v", created, err)
	}
	if _, err := f.svc.CancelBooking(f.staffCtx, connect.NewRequest(&roomsv1.CancelRoomBookingRequest{Id: created.Msg.Booking.Id})); err != nil {
		t.Fatal(err)
	}
	events := eventservice.NewService(f.db, nil, f.svc.authz, nil, nil, "", "")
	endText := end.Format(time.RFC3339)
	_, err = events.UpdateEvent(f.creatorCtx, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: f.event.ID, EndTime: &endText}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("cancelled approved replacement must block event save: %v", err)
	}
	f.assertOriginalPreserved(t)
}

func TestCancelBookingRollsBackBothReservationAndRequestOnFailure(t *testing.T) {
	for _, failure := range []string{"request_update", "reservation_delete"} {
		t.Run(failure, func(t *testing.T) {
			f := newBookingReplacementFixture(t)
			injectBookingCancellationFailure(t, f, failure)
			_, err := f.svc.CancelBooking(f.staffCtx, connect.NewRequest(&roomsv1.CancelRoomBookingRequest{Id: f.original.Booking.Id}))
			if connect.CodeOf(err) != connect.CodeInternal {
				t.Fatalf("injected %s failure must reject cancellation: %v", failure, err)
			}
			f.assertOriginalPreserved(t)
		})
	}
}

func injectBookingCancellationFailure(t *testing.T, f bookingReplacementFixture, failure string) {
	t.Helper()
	query := "ALTER TABLE booking_requests ADD CONSTRAINT forbid_cancellation CHECK (status <> 'cancelled') NOT VALID"
	if failure == "reservation_delete" {
		query = `CREATE FUNCTION forbid_booking_delete() RETURNS TRIGGER LANGUAGE plpgsql AS $$
			BEGIN RAISE EXCEPTION 'injected booking delete failure'; END;
			$$;
			CREATE TRIGGER forbid_booking_delete BEFORE DELETE ON room_bookings
			FOR EACH ROW EXECUTE FUNCTION forbid_booking_delete()`
	}
	if _, err := f.db.ExecContext(f.staffCtx, query); err != nil {
		t.Fatal(err)
	}
}

func TestCancelBookingRequestReleasesApprovedReservation(t *testing.T) {
	f := newBookingReplacementFixture(t)
	unrelated := models.RoomBooking{
		RoomID: f.room.ID, CreatedBy: f.event.UserID, Title: "Unrelated individual reservation",
		StartTime: f.event.EndTime, EndTime: f.event.EndTime.Add(time.Hour),
	}
	if _, err := f.db.NewInsert().Model(&unrelated).Returning("*").Exec(f.staffCtx); err != nil {
		t.Fatal(err)
	}
	request := connect.NewRequest(&roomsv1.CancelBookingRequestRequest{Id: f.original.BookingRequest.Id})
	response, err := f.svc.CancelBookingRequest(f.staffCtx, request)
	if err != nil || !response.Msg.Success {
		t.Fatalf("cancel approved request: response=%v, err=%v", response, err)
	}
	var linked models.BookingRequest
	if err := f.db.NewSelect().Model(&linked).Where("id = ?", request.Msg.Id).Scan(f.staffCtx); err != nil {
		t.Fatal(err)
	}
	if linked.Status != models.BookingRequestStatusCancelled {
		t.Fatalf("cancelled request status = %s, want cancelled", linked.Status)
	}
	exists, err := f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("booking_request_id = ?", request.Msg.Id).Exists(f.staffCtx)
	if err != nil || exists {
		t.Fatalf("cancelled request retained ghost occupancy: exists=%t, err=%v", exists, err)
	}
	exists, err = f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("id = ?", unrelated.ID).Exists(f.staffCtx)
	if err != nil || !exists {
		t.Fatalf("request cancellation removed unrelated occupancy: exists=%t, err=%v", exists, err)
	}
	if _, err := f.svc.CancelBookingRequest(f.staffCtx, request); err != nil {
		t.Fatalf("repeated request cancellation must be idempotent: %v", err)
	}
}

func TestCancelBookingRequestPreservesRequesterAndUnknownIDSemantics(t *testing.T) {
	f := newBookingReplacementFixture(t)
	// The event's creator cannot cancel a booking request made by another user.
	if _, err := f.svc.CancelBookingRequest(f.creatorCtx, connect.NewRequest(&roomsv1.CancelBookingRequestRequest{Id: f.original.BookingRequest.Id})); err != nil {
		t.Fatalf("unowned request keeps the previous no-op success response: %v", err)
	}
	f.assertOriginalPreserved(t)
	if _, err := f.svc.CancelBookingRequest(f.staffCtx, connect.NewRequest(&roomsv1.CancelBookingRequestRequest{Id: f.original.BookingRequest.Id + 1000})); err != nil {
		t.Fatalf("unknown request keeps the previous no-op success response: %v", err)
	}
	f.assertOriginalPreserved(t)
}

func TestCancelBookingRequestRollsBackBothReservationAndRequestOnFailure(t *testing.T) {
	for _, failure := range []string{"request_update", "reservation_delete"} {
		t.Run(failure, func(t *testing.T) {
			f := newBookingReplacementFixture(t)
			injectBookingCancellationFailure(t, f, failure)
			_, err := f.svc.CancelBookingRequest(f.staffCtx, connect.NewRequest(&roomsv1.CancelBookingRequestRequest{Id: f.original.BookingRequest.Id}))
			if connect.CodeOf(err) != connect.CodeInternal {
				t.Fatalf("injected %s failure must reject request cancellation: %v", failure, err)
			}
			f.assertOriginalPreserved(t)
		})
	}
}
