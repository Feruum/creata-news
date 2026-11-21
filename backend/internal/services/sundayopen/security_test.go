package sundayopen

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	pbv1 "github.com/studyverse/ems-backend/gen/sundayopen/v1"
	"github.com/studyverse/ems-backend/internal/authz"
)

func TestAnonymousCannotCancelByKnowingPhone(t *testing.T) {
	s := NewService(nil, authz.NewEnforcer(nil, nil))
	_, err := s.CancelExternalRegistration(context.Background(), connect.NewRequest(&pbv1.CancelExternalRegistrationRequest{EventId: 1, Phone: "+77000000000"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}
