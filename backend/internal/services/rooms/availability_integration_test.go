//go:build integration

package rooms

import (
	"connectrpc.com/connect"
	"context"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"testing"
	"time"
)

func TestAvailabilityIncludesOvernightBookingOnCampusDate(t *testing.T) {
	db := newTestDB(t)
	room := seedRoom(t, db, "A-101", "QA room")
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO room_bookings(room_id,title,start_time,end_time,created_by)
		VALUES(?, 'Overnight', '2026-10-03T18:00:00Z', '2026-10-04T03:30:00Z', 1)`, room.ID); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, nil, nil)
	response, err := svc.GetRoomAvailability(ctx, connect.NewRequest(&roomsv1.GetRoomAvailabilityRequest{RoomId: room.ID, Date: "2026-10-04"}))
	if err != nil {
		t.Fatal(err)
	}
	slots := response.Msg.Slots
	if len(slots) != 13 {
		t.Fatalf("slot count = %d", len(slots))
	}
	if slots[0].StartTime != "2026-10-04T08:00:00+05:00" || slots[12].EndTime != "2026-10-04T21:00:00+05:00" {
		t.Fatal("slots use wrong date or timezone")
	}
	if slots[0].IsAvailable || slots[0].Booking == nil || !slots[1].IsAvailable {
		t.Fatal("overnight booking overlap is incorrect")
	}
	for _, slot := range slots {
		start, err := time.Parse(time.RFC3339, slot.StartTime)
		if err != nil || start.Day() != 4 {
			t.Fatalf("slot crosses day: %v", slot)
		}
	}
}
