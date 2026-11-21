package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/search"
)

// fakeSearchClient records calls for assertion.
type fakeSearchClient struct {
	indexedUser *search.UserDocument
	indexedOrg  *search.OrganizationDocument
	indexedEvt  *search.EventDocument
	deletedIdx  string
	deletedID   int32
}

func (f *fakeSearchClient) IndexUser(_ context.Context, doc *search.UserDocument) error {
	f.indexedUser = doc
	return nil
}

func (f *fakeSearchClient) IndexOrganization(_ context.Context, doc *search.OrganizationDocument) error {
	f.indexedOrg = doc
	return nil
}

func (f *fakeSearchClient) IndexEvent(_ context.Context, doc *search.EventDocument) error {
	f.indexedEvt = doc
	return nil
}

func (f *fakeSearchClient) DeleteDocument(_ context.Context, index string, id int32) error {
	f.deletedIdx = index
	f.deletedID = id
	return nil
}

func TestHandleUserCreatedIndexesUser(t *testing.T) {
	client := &fakeSearchClient{}
	handler := handleUserCreated(HandlerDeps{SearchClient: client})

	payload, _ := json.Marshal(events.UserCreatedPayload{UserID: 7, Username: "alice", Email: "alice@example.com"})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if client.indexedUser == nil {
		t.Fatal("expected user document to be indexed")
	}
	if client.indexedUser.ID != 7 {
		t.Fatalf("indexedUser.ID = %d, want 7", client.indexedUser.ID)
	}
	if client.indexedUser.Username != "alice" {
		t.Fatalf("indexedUser.Username = %q, want %q", client.indexedUser.Username, "alice")
	}
	if client.indexedUser.Email != "alice@example.com" {
		t.Fatalf("indexedUser.Email = %q, want %q", client.indexedUser.Email, "alice@example.com")
	}
}

func TestHandleUserDeletedDeletesFromIndexUsers(t *testing.T) {
	client := &fakeSearchClient{}
	handler := handleUserDeleted(HandlerDeps{SearchClient: client})

	payload, _ := json.Marshal(events.UserDeletedPayload{UserID: 42})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if client.deletedIdx != search.IndexUsers {
		t.Fatalf("deletedIdx = %q, want %q", client.deletedIdx, search.IndexUsers)
	}
	if client.deletedID != 42 {
		t.Fatalf("deletedID = %d, want 42", client.deletedID)
	}
}

func TestHandleOrganizationCreatedIndexesOrganization(t *testing.T) {
	client := &fakeSearchClient{}
	handler := handleOrganizationCreated(HandlerDeps{SearchClient: client})

	payload, _ := json.Marshal(events.OrganizationCreatedPayload{
		OrganizationID:    9,
		OrganizationTitle: "Robotics Club",
		TypeID:            2,
		ActorID:           7,
	})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if client.indexedOrg == nil {
		t.Fatal("expected organization document to be indexed")
	}
	if client.indexedOrg.ID != 9 {
		t.Fatalf("indexedOrg.ID = %d, want 9", client.indexedOrg.ID)
	}
	if client.indexedOrg.Title != "Robotics Club" {
		t.Fatalf("indexedOrg.Title = %q, want %q", client.indexedOrg.Title, "Robotics Club")
	}
	if client.indexedOrg.OrganizationTypeID != 2 {
		t.Fatalf("indexedOrg.OrganizationTypeID = %d, want 2", client.indexedOrg.OrganizationTypeID)
	}
}

func TestHandleEventUpdatedCancelledDoesNotIndex(t *testing.T) {
	client := &fakeSearchClient{}
	handler := handleEventUpdated(HandlerDeps{SearchClient: client})

	payload, _ := json.Marshal(events.EventUpdatedPayload{
		EventID:           5,
		Title:             "Cancelled Event",
		OrganizationID:    1,
		OrganizationTitle: "Some Org",
		Status:            "cancelled",
		StartTime:         "2026-04-01T10:00:00Z",
		EndTime:           "2026-04-01T12:00:00Z",
		ActorID:           3,
	})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if client.indexedEvt != nil {
		t.Fatal("expected cancelled event NOT to be indexed")
	}
}

func TestHandleEventDeletedDeletesFromIndexEvents(t *testing.T) {
	client := &fakeSearchClient{}
	handler := handleEventDeleted(HandlerDeps{SearchClient: client})

	payload, _ := json.Marshal(events.EventDeletedPayload{EventID: 99, Title: "Old Event", OrganizationID: 1, ActorID: 2})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if client.deletedIdx != search.IndexEvents {
		t.Fatalf("deletedIdx = %q, want %q", client.deletedIdx, search.IndexEvents)
	}
	if client.deletedID != 99 {
		t.Fatalf("deletedID = %d, want 99", client.deletedID)
	}
}

func TestHandleUserUpdatedIndexesUser(t *testing.T) {
	client := &fakeSearchClient{}
	handler := handleUserUpdated(HandlerDeps{SearchClient: client})

	payload, _ := json.Marshal(events.UserUpdatedPayload{UserID: 15, Username: "bob", Email: "bob@example.com"})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if client.indexedUser == nil || client.indexedUser.ID != 15 {
		t.Fatal("expected user document to be indexed with ID 15")
	}
}

func TestHandleOrganizationDeletedDeletesFromIndex(t *testing.T) {
	client := &fakeSearchClient{}
	handler := handleOrganizationDeleted(HandlerDeps{SearchClient: client})

	payload, _ := json.Marshal(events.OrganizationDeletedPayload{OrganizationID: 33, OrganizationTitle: "Old Org", ActorID: 1})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if client.deletedIdx != search.IndexOrganizations {
		t.Fatalf("deletedIdx = %q, want %q", client.deletedIdx, search.IndexOrganizations)
	}
	if client.deletedID != 33 {
		t.Fatalf("deletedID = %d, want 33", client.deletedID)
	}
}

func TestHandleEventCreatedIndexesEvent(t *testing.T) {
	client := &fakeSearchClient{}
	handler := handleEventCreated(HandlerDeps{SearchClient: client})

	payload, _ := json.Marshal(events.EventCreatedPayload{
		EventID:           10,
		Title:             "New Event",
		OrganizationID:    2,
		OrganizationTitle: "Cool Org",
		Status:            "published",
		StartTime:         "2026-04-01T10:00:00Z",
		EndTime:           "2026-04-01T12:00:00Z",
		Location:          "Room 101",
		ActorID:           5,
	})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if client.indexedEvt == nil {
		t.Fatal("expected event document to be indexed")
	}
	if client.indexedEvt.ID != 10 {
		t.Fatalf("indexedEvt.ID = %d, want 10", client.indexedEvt.ID)
	}
	if client.indexedEvt.Title != "New Event" {
		t.Fatalf("indexedEvt.Title = %q, want %q", client.indexedEvt.Title, "New Event")
	}
	if client.indexedEvt.OrganizationTitle != "Cool Org" {
		t.Fatalf("indexedEvt.OrganizationTitle = %q, want %q", client.indexedEvt.OrganizationTitle, "Cool Org")
	}
}

func TestDraftCreatedIsRemovedFromPublicSearch(t *testing.T) {
	client := &fakeSearchClient{}
	payload, _ := json.Marshal(events.EventCreatedPayload{EventID: 10, Status: "draft", Title: "private draft"})
	if err := handleEventCreated(HandlerDeps{SearchClient: client})(message.NewMessage("draft", payload)); err != nil {
		t.Fatal(err)
	}
	if client.indexedEvt != nil || client.deletedIdx != search.IndexEvents || client.deletedID != 10 {
		t.Fatalf("draft escaped search: %+v", client)
	}
}

func TestHandleEventUpdatedNonCancelledIndexesEvent(t *testing.T) {
	client := &fakeSearchClient{}
	handler := handleEventUpdated(HandlerDeps{SearchClient: client})

	payload, _ := json.Marshal(events.EventUpdatedPayload{
		EventID:           5,
		Title:             "Active Event",
		OrganizationID:    1,
		OrganizationTitle: "Some Org",
		Status:            "published",
		StartTime:         "2026-04-01T10:00:00Z",
		EndTime:           "2026-04-01T12:00:00Z",
		Location:          "Auditorium",
		ActorID:           3,
	})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if client.indexedEvt == nil {
		t.Fatal("expected non-cancelled event to be indexed")
	}
	if client.indexedEvt.ID != 5 {
		t.Fatalf("indexedEvt.ID = %d, want 5", client.indexedEvt.ID)
	}
}

func TestHandleSearchNilClientReturnsNil(t *testing.T) {
	// All search handlers should return nil when SearchClient is nil.
	handler := handleUserCreated(HandlerDeps{SearchClient: nil})

	payload, _ := json.Marshal(events.UserCreatedPayload{UserID: 1, Username: "x", Email: "x@x.com"})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}
}
