package events

import (
	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"testing"
)

func TestEventOwnershipCannotBeChangedByUpdate(t *testing.T) {
	event := models.Event{UserID: 7, OrganizationID: 3}
	other := int32(8)
	for _, req := range []*eventsv1.UpdateEventRequest{{UserId: &other}, {OrganizationId: &other}} {
		if err := validateEventOwnershipUpdate(event, req); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("ownership transfer accepted: %v", err)
		}
	}
	same := int32(3)
	if err := validateEventOwnershipUpdate(event, &eventsv1.UpdateEventRequest{OrganizationId: &same}); err != nil {
		t.Fatal(err)
	}
}
