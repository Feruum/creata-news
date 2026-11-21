package events

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
)

// TestUpdateEventOrganizer_UnauthorizedCallerIsDeniedBeforeMutation proves that
// an unauthenticated caller is rejected BEFORE any DB mutation is attempted.
// With authz wired but no authenticated context, RequireResourcePermission must
// fire before the UPDATE query. A nil db pointer ensures a panic if the handler
// reaches the mutation code path.
func TestUpdateEventOrganizer_UnauthorizedCallerIsDeniedBeforeMutation(t *testing.T) {
	svc := &Service{
		db:    nil,
		authz: authz.NewEnforcer(nil, nil),
	}

	_, err := svc.UpdateEventOrganizer(context.Background(), connect.NewRequest(&eventsv1.UpdateEventOrganizerRequest{
		Id:   999,
		Role: ptrString("viewer"),
	}))
	if err == nil {
		t.Fatal("expected permission/auth failure, got nil error")
	}
	code := connect.CodeOf(err)
	if code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied {
		t.Fatalf("expected unauthenticated or permission_denied, got %v: %v", code, err)
	}
}
