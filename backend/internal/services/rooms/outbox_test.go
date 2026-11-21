//go:build integration

package rooms

import (
	"testing"

	"connectrpc.com/connect"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestBookingReviewRollsBackWhenDurableEnqueueFails(t *testing.T) {
	for _, action := range []string{"approve", "reject"} {
		t.Run(action, func(t *testing.T) {
			db := newTestDB(t)
			kratosID := "outbox-booking-review"
			actorID := seedAuthUser(t, db, kratosID)
			org := seedOrganization(t, db, "Outbox Club")
			event := seedEvent(t, db, actorID, org.ID, "Outbox Event")
			room := seedRoom(t, db, "101", "Test Room")
			request := seedBookingRequest(t, db, actorID, org.ID, event.ID, room.ID, "Booking")
			ctx := authedContext(kratosID)
			if _, err := db.ExecContext(ctx, "ALTER TABLE domain_event_outbox ADD CONSTRAINT reject_new_events CHECK (false) NOT VALID"); err != nil {
				t.Fatal(err)
			}
			svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), domainevents.NewDurablePublisher(db, &capturePublisher{}))
			var mutationErr error
			if action == "approve" {
				_, mutationErr = svc.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: request.ID, EventId: &event.ID}))
			} else {
				_, mutationErr = svc.RejectBookingRequest(ctx, connect.NewRequest(&roomsv1.RejectBookingRequestRequest{Id: request.ID, Reason: "Test"}))
			}
			if connect.CodeOf(mutationErr) != connect.CodeInternal {
				t.Fatalf("review should reject failed enqueue, got %v", mutationErr)
			}
			if err := db.NewSelect().Model(&request).WherePK().Scan(ctx); err != nil {
				t.Fatal(err)
			}
			if request.Status != models.BookingRequestStatusPending {
				t.Fatalf("review escaped rollback: %q", request.Status)
			}
			count, err := db.NewSelect().Model((*models.RoomBooking)(nil)).Count(ctx)
			if err != nil || count != 0 {
				t.Fatalf("booking escaped rollback: count=%d err=%v", count, err)
			}
		})
	}
}
