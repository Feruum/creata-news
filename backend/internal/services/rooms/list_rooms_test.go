//go:build integration

package rooms

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestAdminCanListAndReactivateInactiveRooms(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	adminID := seedAuthUser(t, db, "room-catalog-admin")
	userID := seedAuthUser(t, db, "room-catalog-user")
	if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id, role) VALUES (?, 'administrator')", adminID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id, role) VALUES (?, 'none')", userID); err != nil {
		t.Fatal(err)
	}
	seedRoom(t, db, "101", "Catalog active")
	seedRoom(t, db, "102", "Catalog inactive")
	if _, err := db.ExecContext(ctx, "UPDATE rooms SET is_active = false WHERE id = '102'"); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
	search := "Catalog"
	for _, caller := range []context.Context{ctx, authedContext("room-catalog-user")} {
		_, err := svc.ListRooms(caller, connect.NewRequest(&roomsv1.ListRoomsRequest{IncludeInactive: true}))
		if err == nil {
			t.Fatal("non-admin listed inactive rooms")
		}
	}
	public, err := svc.ListRooms(ctx, connect.NewRequest(&roomsv1.ListRoomsRequest{Search: &search}))
	if err != nil || public.Msg.Total != 1 || len(public.Msg.Rooms) != 1 {
		t.Fatalf("public catalog should expose only the active room: %v", err)
	}
	adminCtx := authedContext("room-catalog-admin")
	admin, err := svc.ListRooms(adminCtx, connect.NewRequest(&roomsv1.ListRoomsRequest{
		Search: &search, IncludeInactive: true,
	}))
	if err != nil || admin.Msg.Total != 2 || len(admin.Msg.Rooms) != 2 {
		t.Fatalf("admin should be able to find the deactivated room: %v", err)
	}
	active := true
	if _, err := svc.UpdateRoom(adminCtx, connect.NewRequest(&roomsv1.UpdateRoomRequest{Id: "102", IsActive: &active})); err != nil {
		t.Fatal(err)
	}
	public, err = svc.ListRooms(ctx, connect.NewRequest(&roomsv1.ListRoomsRequest{Search: &search}))
	if err != nil || public.Msg.Total != 2 {
		t.Fatalf("reactivated room should return to the public catalog: %v", err)
	}
}
