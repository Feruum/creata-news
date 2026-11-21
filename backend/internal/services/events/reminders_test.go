package events

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
)

func TestGetPendingReminders_RejectsMissingInternalSecret(t *testing.T) {
	svc := &Service{internalReminderSecret: "top-secret"}
	_, err := svc.GetPendingReminders(context.Background(), connect.NewRequest(&eventsv1.GetPendingRemindersRequest{
		ReminderType: "day_before",
	}))
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected unauthenticated, got %v", connect.CodeOf(err))
	}
}

func TestMarkRemindersSent_RejectsMissingInternalSecret(t *testing.T) {
	svc := &Service{internalReminderSecret: "top-secret"}
	_, err := svc.MarkRemindersSent(context.Background(), connect.NewRequest(&eventsv1.MarkRemindersSentRequest{}))
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected unauthenticated, got %v", connect.CodeOf(err))
	}
}

func TestGetPendingReminders_AllowsValidInternalSecret(t *testing.T) {
	// Test the auth gate directly — no DB or handler execution needed.
	req := connect.NewRequest(&eventsv1.GetPendingRemindersRequest{ReminderType: "day_before"})
	req.Header().Set("X-Internal-Secret", "top-secret")

	if err := auth.RequireInternalSecret(req, "top-secret"); err != nil {
		t.Fatalf("expected valid secret to be accepted, got: %v", err)
	}
}

func TestGetPendingReminders_RejectsWrongInternalSecret(t *testing.T) {
	svc := &Service{internalReminderSecret: "top-secret"}
	req := connect.NewRequest(&eventsv1.GetPendingRemindersRequest{ReminderType: "day_before"})
	req.Header().Set("X-Internal-Secret", "wrong-secret")

	_, err := svc.GetPendingReminders(context.Background(), req)
	if err == nil {
		t.Fatal("expected unauthenticated error for wrong secret")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected unauthenticated, got %v", connect.CodeOf(err))
	}
}

func TestGetPendingReminders_RejectsWhenSecretNotConfigured(t *testing.T) {
	svc := &Service{internalReminderSecret: ""}
	req := connect.NewRequest(&eventsv1.GetPendingRemindersRequest{ReminderType: "day_before"})
	req.Header().Set("X-Internal-Secret", "some-secret")

	_, err := svc.GetPendingReminders(context.Background(), req)
	if err == nil {
		t.Fatal("expected internal error when secret not configured")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("expected internal, got %v", connect.CodeOf(err))
	}
}
