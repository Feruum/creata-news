package sundayopen

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"connectrpc.com/connect"
	pbv1 "github.com/studyverse/ems-backend/gen/sundayopen/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestPublicEventDescriptionAndLocation(t *testing.T) {
	for _, tc := range []struct{ name, description, location, wantDescription, wantLocation string }{
		{"rich text campus", `[{"type":"p","children":[{"text":"Learn "},{"text":"together","bold":true}]}]`, `{"type":"campus_room","room_id":"A-101","room_name":"Workshop room","floor":"1"}`, "Learn together", "Workshop room, Floor 1"},
		{"numeric campus floor", `"Campus session"`, `{"type":"campus_room","room_id":"C1.1.221P","room_name":"Astana IT University, аудитория C1.1.221P","floor":1}`, "Campus session", "Astana IT University, аудитория C1.1.221P, Floor 1"},
		{"plain external", `"Design workshop"`, `{"type":"external","external_address":"Astana, 10 Main Street"}`, "Design workshop", "Astana, 10 Main Street"},
		{"online", `"Online session"`, `{"type":"online","online_platform":"Zoom","online_url":"https://example.test"}`, "Online session", "Zoom"},
		{"legacy", `"Session"`, `{"text":"Main hall"}`, "Session", "Main hall"},
		{"tiptap", `{"type":"doc","content":[{"type":"paragraph","attrs":{"id":"private-marker"},"content":[{"type":"text","text":"Learn "},{"type":"text","text":"together"}]},{"type":"paragraph","content":[{"type":"text","text":"Create"}]}]}`, `{}`, "Learn together\nCreate", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := sundayOpenEventRow{Event: models.Event{Description: json.RawMessage(tc.description), StructuredLocation: json.RawMessage(tc.location)}}
			got := dbSundayOpenEventToProto(row)
			if got.Description != tc.wantDescription || got.GetLocationText() != tc.wantLocation {
				t.Fatalf("public content got description=%q location=%q; want %q, %q", got.Description, got.GetLocationText(), tc.wantDescription, tc.wantLocation)
			}
		})
	}
}

func TestPublicRegistrationOpenFollowsLifecycle(t *testing.T) {
	categoryID := int32(2)
	categoryName := "Workshops"
	base := models.Event{Status: models.EventStatusPublished, EndTime: time.Now().Add(time.Hour), CategoryID: &categoryID}
	for _, tc := range []struct {
		name  string
		event models.Event
		open  bool
	}{
		{"ongoing", base, true},
		{"ended", func() models.Event { e := base; e.EndTime = time.Now().Add(-time.Hour); return e }(), false},
		{"completed", func() models.Event { e := base; now := time.Now(); e.CompletedAt = &now; return e }(), false},
		{"draft", func() models.Event { e := base; e.Status = models.EventStatusDraft; return e }(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := dbSundayOpenEventToProto(sundayOpenEventRow{Event: tc.event, CategoryName: &categoryName})
			if got.RegistrationOpen != tc.open || got.GetCategoryId() != categoryID || got.GetCategoryName() != categoryName {
				t.Fatalf("registrationOpen=%v category=%d/%q", got.RegistrationOpen, got.GetCategoryId(), got.GetCategoryName())
			}
		})
	}
}

func TestExternalListValidatesEventBeforeAuthentication(t *testing.T) {
	s := NewService(nil, authz.NewEnforcer(nil, nil))
	_, err := s.GetExternalRegistrations(context.Background(), connect.NewRequest(&pbv1.GetExternalRegistrationsRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid event ID, got %v", err)
	}
}

func TestAnonymousCannotReadExternalRegistrations(t *testing.T) {
	s := NewService(nil, authz.NewEnforcer(nil, nil))
	_, err := s.GetExternalRegistrations(context.Background(), connect.NewRequest(&pbv1.GetExternalRegistrationsRequest{EventId: 1}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestListRejectsInvalidFiltersBeforeDatabase(t *testing.T) {
	s := NewService(nil, nil)
	invalidCategory := int32(-1)
	for _, req := range []*pbv1.ListPublicEventsRequest{{Period: pbv1.PublicEventPeriod(99)}, {CategoryId: &invalidCategory}} {
		_, err := s.ListPublicEvents(context.Background(), connect.NewRequest(req))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("expected invalid filter, got %v", err)
		}
	}
}
