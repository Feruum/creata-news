package events

import (
	"bytes"
	"connectrpc.com/connect"
	"context"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLegacyImagePresignCannotBypassValidation(t *testing.T) {
	s := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := s.GetEventImageUploadUrl(context.Background(), connect.NewRequest(&eventsv1.GetEventImageUploadUrlRequest{}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("legacy upload must be closed, got %v", err)
	}
}

func TestImageUploadRequiresAuthentication(t *testing.T) {
	s := &Service{authz: authz.NewEnforcer(nil, nil)}
	_, err := s.UploadEventImage(context.Background(), connect.NewRequest(&eventsv1.UploadEventImageRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("authentication must precede decode/storage, got %v", err)
	}
}

func TestImageRPCRejectsBodyBeforeUnboundedDecode(t *testing.T) {
	_, handler := eventsv1connect.NewEventsServiceHandler(&Service{authz: authz.NewEnforcer(nil, nil)}, connect.WithReadMaxBytes(storage.ImageRPCReadMaxBytes))
	server := httptest.NewServer(handler)
	defer server.Close()
	client := eventsv1connect.NewEventsServiceClient(server.Client(), server.URL, connect.WithProtoJSON())
	_, err := client.UploadEventImage(context.Background(), connect.NewRequest(&eventsv1.UploadEventImageRequest{Image: bytes.Repeat([]byte{1}, storage.ImageRPCReadMaxBytes), ContentType: "image/png", OrganizationId: 1}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("oversized JSON request reached handler: %v", err)
	}
	response, err := server.Client().Post(server.URL+eventsv1connect.EventsServiceUploadEventImageProcedure, "application/json", bytes.NewBufferString(`{"image":"AA==","contentType":"image/png","organizationId":1}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusRequestEntityTooLarge {
		t.Fatal("small request rejected by body bound")
	}
}
