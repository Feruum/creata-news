package rooms

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/authz"
)

func TestGetBookingRequest_RequiresPlatformStaff(t *testing.T) {
	svc := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := svc.GetBookingRequest(context.Background(), connect.NewRequest(&roomsv1.GetBookingRequestRequest{Id: 1}))
	if err == nil {
		t.Fatal("expected auth failure")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied {
		t.Fatalf("expected unauthenticated or permission_denied, got %v", code)
	}
}

func TestListBookingRequests_RequiresPlatformStaff(t *testing.T) {
	svc := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := svc.ListBookingRequests(context.Background(), connect.NewRequest(&roomsv1.ListBookingRequestsRequest{}))
	if err == nil {
		t.Fatal("expected auth failure")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied {
		t.Fatalf("expected unauthenticated or permission_denied, got %v", code)
	}
}

func TestListBookings_RequiresPlatformStaff(t *testing.T) {
	svc := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := svc.ListBookings(context.Background(), connect.NewRequest(&roomsv1.ListBookingsRequest{}))
	if err == nil {
		t.Fatal("expected auth failure")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied {
		t.Fatalf("expected unauthenticated or permission_denied, got %v", code)
	}
}
