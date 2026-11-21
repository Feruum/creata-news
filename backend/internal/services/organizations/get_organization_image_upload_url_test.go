package organizations

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
)

func TestGetOrganizationImageUploadUrl_RejectsRetiredDirectUpload(t *testing.T) {
	svc := &Service{authz: authz.NewEnforcer(nil, nil)}
	response, err := svc.GetOrganizationImageUploadUrl(context.Background(), connect.NewRequest(&eventsv1.GetOrganizationImageUploadUrlRequest{
		OrganizationId: 1,
		Filename:       "logo.png",
		ContentType:    "image/png",
	}))
	if err == nil {
		t.Fatal("expected retired direct upload failure")
	}
	if response != nil || connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("retired endpoint must issue no upload credential, response=%v error=%v", response, err)
	}
}
