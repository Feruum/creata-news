package organizations

import (
	"connectrpc.com/connect"
	"context"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"testing"
)

func TestLegacyImagePresignCannotBypassValidation(t *testing.T) {
	s := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := s.GetOrganizationImageUploadUrl(context.Background(), connect.NewRequest(&eventsv1.GetOrganizationImageUploadUrlRequest{}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("legacy upload must be closed, got %v", err)
	}
}

func TestImageUploadRequiresAuthentication(t *testing.T) {
	s := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := s.UploadOrganizationImage(context.Background(), connect.NewRequest(&eventsv1.UploadOrganizationImageRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("authentication must precede decode/storage, got %v", err)
	}
}
