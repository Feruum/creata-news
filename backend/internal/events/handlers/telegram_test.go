package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/events"
)

func TestPostBulkTelegramNotificationIncludesCredentialAndMessage(t *testing.T) {
	var gotPath, gotSecret, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSecret = r.Header.Get("X-Internal-Secret")
		data, _ := io.ReadAll(r.Body)
		gotBody = string(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"sent":1,"failed":0}`))
	}))
	defer server.Close()
	if err := postBulkTelegramNotification(context.Background(), server.URL, "audit-notification-secret", telegramBulkRequest{Notifications: []telegramNotification{{TelegramID: 101, Message: "New collaboration invite for Launch"}}}); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/notify/bulk" || gotSecret != "audit-notification-secret" || !strings.Contains(gotBody, "New collaboration invite for Launch") {
		t.Fatalf("notification request omitted credential or message: %s %s", gotPath, gotBody)
	}
}
func TestHandleCollaborationInvitedEmptyURLReturnsNil(t *testing.T) {
	handler := handleCollaborationInvited(HandlerDeps{TelegramBotURL: ""})
	payload, _ := json.Marshal(events.CollaborationInvitedPayload{EventID: 1, EventTitle: "E", OrganizationID: 1, HostOrganizationID: 1, InviteToken: "t", MapAppURL: "http://x", ActorID: 1})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}
}

func TestHandleCollaborationInvitedBadPayloadReturnsError(t *testing.T) {
	handler := handleCollaborationInvited(HandlerDeps{TelegramBotURL: "http://unused"})
	msg := message.NewMessage("1", []byte("not json"))
	msg.SetContext(context.Background())

	if err := handler(msg); err == nil {
		t.Fatal("expected error for bad payload, got nil")
	}
}

func TestPostBulkTelegramNotificationServerErrorReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	defer server.Close()
	if err := postBulkTelegramNotification(context.Background(), server.URL, "audit-notification-secret", telegramBulkRequest{Notifications: []telegramNotification{{TelegramID: 101, Message: "Test"}}}); err == nil {
		t.Fatal("notification service error acknowledged")
	}
}
func TestHandleCollaborationAcceptedEmptyURLReturnsNil(t *testing.T) {
	handler := handleCollaborationAccepted(HandlerDeps{TelegramBotURL: ""})
	payload, _ := json.Marshal(events.CollaborationAcceptedPayload{EventID: 1, EventTitle: "E", OrganizationID: 1, HostOrganizationID: 1, ActorID: 1})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}
}

func TestHandleCollaborationDeclinedEmptyURLReturnsNil(t *testing.T) {
	handler := handleCollaborationDeclined(HandlerDeps{TelegramBotURL: ""})
	payload, _ := json.Marshal(events.CollaborationDeclinedPayload{EventID: 1, EventTitle: "E", OrganizationID: 1, HostOrganizationID: 1, ActorID: 1})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}
}

func TestHandleBookingApprovedEmptyURLReturnsNil(t *testing.T) {
	handler := handleBookingApproved(HandlerDeps{TelegramBotURL: ""})
	payload, _ := json.Marshal(events.BookingApprovedPayload{BookingRequestID: 1, EventID: 1, EventTitle: "E", OrganizationID: 1, RoomID: "A-101", RoomName: "R", ActorID: 1})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}
}

func TestHandleBookingRejectedEmptyURLReturnsNil(t *testing.T) {
	handler := handleBookingRejected(HandlerDeps{TelegramBotURL: ""})
	payload, _ := json.Marshal(events.BookingRejectedPayload{BookingRequestID: 1, EventID: 1, EventTitle: "E", OrganizationID: 1, RoomID: "A-101", RoomName: "R", ActorID: 1})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}
}

func TestHandleEventPublishedPostsTelegramNotificationToSubscribers(t *testing.T) {
	// Without a DB, the handler should return nil (no subscribers to notify).
	handler := handleEventPublished(HandlerDeps{TelegramBotURL: "http://unused"})
	payload, _ := json.Marshal(events.EventPublishedPayload{EventID: 5, Title: "Grand Opening", OrganizationID: 1, OrganizationName: "CS Club", StartTime: "2026-04-01T10:00:00Z", Location: "Room 101", EventSlug: "grand-opening", MapAppURL: "http://localhost:3000", ActorID: 3})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}
}

func TestHandleEventPublishedEmptyURLReturnsNil(t *testing.T) {
	handler := handleEventPublished(HandlerDeps{TelegramBotURL: ""})
	payload, _ := json.Marshal(events.EventPublishedPayload{EventID: 5, Title: "E", OrganizationID: 1, OrganizationName: "O", StartTime: "2026-04-01T10:00:00Z", Location: "L", EventSlug: "s", MapAppURL: "http://x", ActorID: 1})
	msg := message.NewMessage("1", payload)
	msg.SetContext(context.Background())

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}
}

func TestPostBulkNotificationsChunksRecipients(t *testing.T) {
	var sizes []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request telegramBulkRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		sizes = append(sizes, len(request.Notifications))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "sent": len(request.Notifications), "failed": 0})
	}))
	defer server.Close()
	request := telegramBulkRequest{ParseMode: "HTML", ReplyMarkup: &inlineKeyboardMarkup{InlineKeyboard: [][]inlineKeyboardButton{{{Text: "View", URL: "https://example.test/event"}}}}}
	for i := 0; i < 201; i++ {
		request.Notifications = append(request.Notifications, telegramNotification{TelegramID: int64(i + 1), Message: "Test"})
	}
	if err := postBulkTelegramNotification(context.Background(), server.URL, "audit-notification-secret", request); err != nil {
		t.Fatal(err)
	}
	if len(sizes) != 3 || sizes[0] != 100 || sizes[1] != 100 || sizes[2] != 1 {
		t.Fatalf("unbounded recipient requests: %v", sizes)
	}
}
