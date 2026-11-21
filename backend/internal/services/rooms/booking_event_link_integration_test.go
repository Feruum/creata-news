//go:build integration

package rooms

import (
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/models"
	eventservice "github.com/studyverse/ems-backend/internal/services/events"
)

func TestApproveBookingRequestRejectsChangingExistingEventLink(t *testing.T) {
	f := newBookingReplacementFixture(t)
	otherEvent := seedEvent(t, f.db, f.event.UserID, f.event.OrganizationID, "Other linked event")
	request := f.request(f.event.EndTime)
	request.Msg.EventId = &otherEvent.ID
	created, err := f.svc.CreateBookingRequest(f.creatorCtx, request)
	if err != nil {
		t.Fatal(err)
	}
	if created.Msg.AutoApproved || created.Msg.BookingRequest.Status != roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_PENDING {
		t.Fatalf("conflicting request must start pending: %+v", created.Msg)
	}
	// The supplied ID must not let the other event's request borrow this event's
	// exemption from conflicts with its own existing reservation.
	_, err = f.svc.ApproveBookingRequest(f.staffCtx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{
		Id: created.Msg.BookingRequest.Id, EventId: &f.event.ID,
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("an existing event link cannot be reassigned during approval: %v", err)
	}
	var stored models.BookingRequest
	if err := f.db.NewSelect().Model(&stored).Where("id = ?", created.Msg.BookingRequest.Id).Scan(f.staffCtx); err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.BookingRequestStatusPending || stored.EventID == nil || *stored.EventID != otherEvent.ID ||
		stored.ReviewedBy != nil || stored.ReviewedAt != nil {
		t.Fatalf("rejected override mutated the original request: %+v", stored)
	}
	count, err := f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("room_id = ?", f.room.ID).Count(f.staffCtx)
	if err != nil || count != 1 {
		t.Fatalf("rejected override created a reservation: count=%d, err=%v", count, err)
	}
	f.assertOriginalPreserved(t)
}

func TestApproveBookingRequestLinksUnlinkedRequestAtomically(t *testing.T) {
	for _, failInsert := range []bool{false, true} {
		t.Run(fmt.Sprintf("insert_failure_%t", failInsert), func(t *testing.T) {
			f := newBookingReplacementFixture(t)
			end := f.event.EndTime.Add(30 * time.Minute)
			request := f.request(end)
			request.Msg.EventId, request.Msg.OrganizationId = nil, nil
			created, err := f.svc.CreateBookingRequest(f.creatorCtx, request)
			if err != nil {
				t.Fatal(err)
			}
			if created.Msg.AutoApproved || created.Msg.BookingRequest.EventId != nil {
				t.Fatalf("individual request must start pending and unlinked: %+v", created.Msg)
			}
			if failInsert {
				if _, err := f.db.ExecContext(f.staffCtx, `CREATE FUNCTION reject_linked_booking() RETURNS TRIGGER LANGUAGE plpgsql AS $$
					BEGIN RAISE EXCEPTION USING ERRCODE = '23P01', MESSAGE = 'injected occupancy conflict'; END;
					$$;
					CREATE TRIGGER reject_linked_booking BEFORE INSERT ON room_bookings
					FOR EACH ROW EXECUTE FUNCTION reject_linked_booking()`); err != nil {
					t.Fatal(err)
				}
			}
			approved, approvalErr := f.svc.ApproveBookingRequest(f.staffCtx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{
				Id: created.Msg.BookingRequest.Id, EventId: &f.event.ID,
			}))
			var stored models.BookingRequest
			if err := f.db.NewSelect().Model(&stored).Where("id = ?", created.Msg.BookingRequest.Id).Scan(f.staffCtx); err != nil {
				t.Fatal(err)
			}
			if failInsert {
				if connect.CodeOf(approvalErr) != connect.CodeFailedPrecondition {
					t.Fatalf("occupancy failure must reject linked approval: %v", approvalErr)
				}
				if stored.EventID != nil || stored.Status != models.BookingRequestStatusPending || stored.ReviewedBy != nil || stored.ReviewedAt != nil {
					t.Fatalf("failed reservation insert retained link or approval state: %+v", stored)
				}
				count, err := f.db.NewSelect().Model((*models.RoomBooking)(nil)).Where("room_id = ?", f.room.ID).Count(f.staffCtx)
				if err != nil || count != 1 {
					t.Fatalf("failed linked approval changed original occupancy: count=%d, err=%v", count, err)
				}
				f.assertOriginalPreserved(t)
				return
			}
			if approvalErr != nil {
				t.Fatalf("approve individual request linked to event: %v", approvalErr)
			}
			if approved.Msg.BookingRequest.EventId == nil || *approved.Msg.BookingRequest.EventId != f.event.ID ||
				approved.Msg.Booking.EventId == nil || *approved.Msg.Booking.EventId != f.event.ID {
				t.Errorf("approval response must link both request and reservation to the event: %+v", approved.Msg)
			}
			if stored.EventID == nil || *stored.EventID != f.event.ID || stored.Status != models.BookingRequestStatusApproved {
				t.Errorf("approved request did not persist its assigned event link: %+v", stored)
			}
			var booking models.RoomBooking
			if err := f.db.NewSelect().Model(&booking).Where("booking_request_id = ?", stored.ID).Scan(f.staffCtx); err != nil {
				t.Fatal(err)
			}
			if booking.EventID == nil || *booking.EventID != f.event.ID {
				t.Errorf("persisted reservation lost assigned event link: %+v", booking)
			}
			f.assertOriginalPreserved(t)
			events := eventservice.NewService(f.db, nil, f.svc.authz, nil, nil, "", "")
			endText := end.Format(time.RFC3339)
			readiness, err := events.GetEventPublishReadiness(f.creatorCtx, connect.NewRequest(&eventsv1.GetEventPublishReadinessRequest{
				EventId: f.event.ID, ProposedEndTime: &endText,
			}))
			if err != nil || readiness.Msg.State != eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_READY ||
				readiness.Msg.BookingRequestId == nil || *readiness.Msg.BookingRequestId != stored.ID {
				t.Fatalf("assigned event must see the approved request in proposed readiness: response=%v, err=%v", readiness, err)
			}
			if _, err := events.UpdateEvent(f.creatorCtx, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: f.event.ID, EndTime: &endText})); err != nil {
				t.Fatalf("assigned approved request must permit saving the proposed schedule: %v", err)
			}
		})
	}
}
