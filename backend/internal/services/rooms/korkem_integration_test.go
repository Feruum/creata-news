//go:build integration

package rooms

import (
	"testing"

	"connectrpc.com/connect"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestKorkemRequestsApprovalConflictsAndAvailability(t *testing.T) {
	db := newTestDB(t)
	seedAuthUser(t, db, "korkem-student")
	ctx := authedContext("korkem-student")
	room := seedRoom(t, db, "K.201", "Room 201")
	cafe := seedRoom(t, db, "K.CAFE", "Cafe")
	if _, err := db.NewUpdate().Model(&room).Set("block = ?", "K").Set("floor = 2").WherePK().Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.NewUpdate().Model(&cafe).Set("block = ?", "K").Set("floor = 0").Set("is_bookable = false").WherePK().Exec(ctx); err != nil {
		t.Fatal(err)
	}
	student := NewService(db, nil, authz.NewEnforcer(db, nil), nil)
	staff := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
	request := func(id string) *connect.Request[roomsv1.CreateBookingRequestRequest] {
		return connect.NewRequest(&roomsv1.CreateBookingRequestRequest{RoomId: id, Title: "Korkem workshop", StartTime: "2026-10-20T05:00:00Z", EndTime: "2026-10-20T06:00:00Z"})
	}
	created, err := student.CreateBookingRequest(ctx, request("к.201"))
	if err != nil {
		t.Fatal(err)
	}
	if created.Msg.AutoApproved || created.Msg.BookingRequest.RoomId != "K.201" {
		t.Fatalf("incorrect student request: %v", created.Msg)
	}
	approved, err := staff.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: created.Msg.BookingRequest.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if approved.Msg.Booking.RoomId != "K.201" {
		t.Fatal("approval changed campus room")
	}
	conflicting, err := student.CreateBookingRequest(ctx, request("k.201"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = staff.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: conflicting.Msg.BookingRequest.Id})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("overlap must block approval: %v", err)
	}
	check, err := staff.CheckRoomsTimeSlot(ctx, connect.NewRequest(&roomsv1.CheckRoomsTimeSlotRequest{RoomIds: []string{"к.201", "K.CAFE"}, StartTime: "2026-10-20T05:00:00Z", EndTime: "2026-10-20T06:00:00Z"}))
	if err != nil {
		t.Fatal(err)
	}
	if check.Msg.Rooms["K.201"].IsAvailable || check.Msg.Rooms["K.201"].ConflictingBooking == nil {
		t.Fatal("Korkem booking was not included in availability")
	}
	if _, err = student.CreateBookingRequest(ctx, request("K.CAFE")); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("cafe must not accept bookings: %v", err)
	}
	var count int
	if err := db.NewSelect().Model((*models.BookingRequest)(nil)).Where("room_id = ?", "K.CAFE").ColumnExpr("count(*)").Scan(ctx, &count); err != nil || count != 0 {
		t.Fatalf("cafe booking rows=%d error=%v", count, err)
	}
}

func TestKorkemApprovalRejectsRoomsDisabledAfterRequest(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		column string
	}{
		{name: "not_bookable", column: "is_bookable"},
		{name: "inactive", column: "is_active"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db := newTestDB(t)
			seedAuthUser(t, db, "korkem-disabled-room-student")
			ctx := authedContext("korkem-disabled-room-student")
			room := seedRoom(t, db, "K.201", "Room 201")
			if _, err := db.NewUpdate().Model(&room).Set("block = ?", "K").Set("floor = 2").WherePK().Exec(ctx); err != nil {
				t.Fatal(err)
			}
			student := NewService(db, nil, authz.NewEnforcer(db, nil), nil)
			staff := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
			created, err := student.CreateBookingRequest(ctx, connect.NewRequest(&roomsv1.CreateBookingRequestRequest{
				RoomId: room.ID, Title: "Korkem workshop", StartTime: "2026-10-20T05:00:00Z", EndTime: "2026-10-20T06:00:00Z",
			}))
			if err != nil {
				t.Fatal(err)
			}
			if created.Msg.AutoApproved || created.Msg.BookingRequest.Status != roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_PENDING {
				t.Fatalf("student request must remain pending: %v", created.Msg)
			}
			if _, err := db.NewUpdate().Model(&room).Set(testCase.column+" = ?", false).WherePK().Exec(ctx); err != nil {
				t.Fatal(err)
			}

			_, err = staff.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: created.Msg.BookingRequest.Id}))
			if connect.CodeOf(err) != connect.CodeFailedPrecondition {
				t.Errorf("approval of a disabled room must fail with failed_precondition: %v", err)
			}
			var pending models.BookingRequest
			if err := db.NewSelect().Model(&pending).Where("id = ?", created.Msg.BookingRequest.Id).Scan(ctx); err != nil {
				t.Fatal(err)
			}
			if pending.Status != models.BookingRequestStatusPending || pending.ReviewedBy != nil || pending.ReviewedAt != nil {
				t.Errorf("rejected approval changed the pending request: %+v", pending)
			}
			count, err := db.NewSelect().Model((*models.RoomBooking)(nil)).Where("room_id = ?", room.ID).Count(ctx)
			if err != nil || count != 0 {
				t.Errorf("disabled room booking rows=%d error=%v", count, err)
			}
		})
	}
}
