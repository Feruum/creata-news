//go:build integration

package events

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/services"
	"github.com/studyverse/ems-backend/internal/services/eventattendance"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestPrelaunchOrganizationSubscriptionLifecycle(t *testing.T) {
	db := newTestDB(t)
	seedAuthUser(t, db, "club-subscriber")
	org := seedOrganization(t, db, "Campus Club")
	svc := NewService(db, nil, authz.NewEnforcer(db, nil), nil, nil, "", "")
	ctx := authedContext("club-subscriber")

	for i := 0; i < 2; i++ {
		if _, err := svc.SubscribeToOrganization(ctx, connect.NewRequest(&eventsv1.SubscribeToOrganizationRequest{OrganizationId: org.ID})); err != nil {
			t.Fatalf("subscribe: %v", err)
		}
	}
	count, err := svc.GetOrganizationSubscriberCount(ctx, connect.NewRequest(&eventsv1.GetOrganizationSubscriberCountRequest{OrganizationId: org.ID}))
	if err != nil || count.Msg.Count != 1 {
		t.Fatalf("duplicate subscription must count once: response=%v err=%v", count, err)
	}
	status, err := svc.GetOrganizationSubscriptionStatus(ctx, connect.NewRequest(&eventsv1.GetOrganizationSubscriptionStatusRequest{OrganizationId: org.ID}))
	if err != nil || !status.Msg.IsSubscribed {
		t.Fatalf("subscription status: response=%v err=%v", status, err)
	}
	if _, err := svc.UnsubscribeFromOrganization(ctx, connect.NewRequest(&eventsv1.UnsubscribeFromOrganizationRequest{OrganizationId: org.ID})); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	count, err = svc.GetOrganizationSubscriberCount(ctx, connect.NewRequest(&eventsv1.GetOrganizationSubscriberCountRequest{OrganizationId: org.ID}))
	if err != nil || count.Msg.Count != 0 {
		t.Fatalf("count after unsubscribe: response=%v err=%v", count, err)
	}
}

func TestPrelaunchEventRegistrationAndReminderLifecycle(t *testing.T) {
	db := newTestDB(t)
	kratosID := "prelaunch-event-admin"
	actorID := seedAuthUser(t, db, kratosID)
	org := seedOrganization(t, db, "Campus Club")
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, nil, nil, "https://example.test", "prelaunch-secret")
	registrations := services.NewEventRegistrationsService(db, nil, enforcer)
	ctx := context.WithValue(authedContext(kratosID), auth.UserEmailKey, "prelaunch_event_admin@test.local")
	draft := eventsv1.EventStatus_EVENT_STATUS_DRAFT
	start := time.Now().UTC().Add(time.Hour)
	created, err := svc.CreateEvent(ctx, connect.NewRequest(&eventsv1.CreateEventRequest{
		Title: "Campus Club prelaunch integration", Description: `[{"type":"p","children":[{"text":"Prelaunch integration description"}]}]`,
		OrganizationId: org.ID, StartTime: start.Format(time.RFC3339), EndTime: start.Add(time.Hour).Format(time.RFC3339),
		Status: &draft, StructuredLocation: &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_ONLINE, OnlineUrl: ptrString("https://example.test/meeting")},
	}))
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	eventID := created.Msg.Event.Id
	if created.Msg.Event.Status != draft {
		t.Fatalf("expected draft, got %v", created.Msg.Event.Status)
	}
	public, err := svc.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{}))
	if err != nil || len(public.Msg.Events) != 0 {
		t.Fatalf("draft in public list: response=%v err=%v", public, err)
	}
	if _, err := registrations.RegisterForEvent(ctx, connect.NewRequest(&eventsv1.RegisterForEventRequest{EventId: eventID, UserId: actorID})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("draft registration should be rejected, got %v", err)
	}
	if _, err := svc.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: eventID})); err != nil {
		t.Fatalf("publish: %v", err)
	}
	public, err = svc.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{}))
	if err != nil || len(public.Msg.Events) != 1 {
		t.Fatalf("published event missing: response=%v err=%v", public, err)
	}
	registration, err := registrations.RegisterForEvent(ctx, connect.NewRequest(&eventsv1.RegisterForEventRequest{EventId: eventID, UserId: actorID}))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := registrations.RegisterForEvent(ctx, connect.NewRequest(&eventsv1.RegisterForEventRequest{EventId: eventID, UserId: actorID})); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("duplicate registration should be rejected, got %v", err)
	}
	if _, err := db.NewRaw("UPDATE users SET telegram_id = 123456789 WHERE id = ?", actorID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	for _, window := range []struct {
		name   string
		offset time.Duration
	}{
		{"day_before", 24 * time.Hour}, {"hour_before", time.Hour}, {"15_min", 15 * time.Minute},
	} {
		t.Run(window.name, func(t *testing.T) {
			if _, err := db.NewRaw("UPDATE events SET start_time = ?, end_time = ? WHERE id = ?", time.Now().Add(window.offset), time.Now().Add(window.offset+time.Hour), eventID).Exec(ctx); err != nil {
				t.Fatal(err)
			}
			pending := connect.NewRequest(&eventsv1.GetPendingRemindersRequest{ReminderType: window.name})
			pending.Header().Set("X-Internal-Secret", "prelaunch-secret")
			response, err := svc.GetPendingReminders(ctx, pending)
			if err != nil || len(response.Msg.Reminders) != 1 {
				t.Fatalf("pending reminder: response=%v err=%v", response, err)
			}
			mark := connect.NewRequest(&eventsv1.MarkRemindersSentRequest{Reminders: []*eventsv1.SentReminder{{EventId: eventID, UserId: actorID, ReminderType: window.name}}})
			mark.Header().Set("X-Internal-Secret", "prelaunch-secret")
			if _, err := svc.MarkRemindersSent(ctx, mark); err != nil {
				t.Fatalf("acknowledge: %v", err)
			}
			response, err = svc.GetPendingReminders(ctx, pending)
			if err != nil || len(response.Msg.Reminders) != 0 {
				t.Fatalf("reminder repeated: response=%v err=%v", response, err)
			}
		})
	}
	attendance := eventattendance.NewService(db, enforcer, "https://example.test")
	if _, err := db.NewRaw("UPDATE events SET start_time = ?, end_time = ? WHERE id = ?", time.Now().Add(-time.Minute), time.Now().Add(time.Hour), eventID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	info, err := attendance.GetCheckInInfo(ctx, connect.NewRequest(&eventsv1.GetCheckInInfoRequest{EventId: eventID}))
	if err != nil || info.Msg.CheckInToken == nil || info.Msg.CurrentRotatingToken == nil {
		t.Fatalf("live QR info: response=%v err=%v", info, err)
	}
	invalid, err := attendance.SelfCheckIn(ctx, connect.NewRequest(&eventsv1.SelfCheckInRequest{Token: *info.Msg.CheckInToken, RotatingToken: "expired-code"}))
	if err != nil || invalid.Msg.Success {
		t.Fatalf("invalid QR should be rejected: response=%v err=%v", invalid, err)
	}
	for i := 0; i < 2; i++ {
		checked, err := attendance.SelfCheckIn(ctx, connect.NewRequest(&eventsv1.SelfCheckInRequest{Token: *info.Msg.CheckInToken, RotatingToken: *info.Msg.CurrentRotatingToken}))
		if err != nil || !checked.Msg.Success {
			t.Fatalf("self check-in: response=%v err=%v", checked, err)
		}
	}
	var attendanceCount int
	if err := db.NewRaw("SELECT count(*) FROM event_attendance WHERE registration_id = ?", registration.Msg.Registration.Id).Scan(ctx, &attendanceCount); err != nil || attendanceCount != 1 {
		t.Fatalf("QR retry must not duplicate attendance: count=%d err=%v", attendanceCount, err)
	}
	if _, err := registrations.CancelRegistration(ctx, connect.NewRequest(&eventsv1.CancelRegistrationRequest{RegistrationId: &registration.Msg.Registration.Id})); err != nil {
		t.Fatalf("cancel registration: %v", err)
	}
	if _, err := registrations.RegisterForEvent(ctx, connect.NewRequest(&eventsv1.RegisterForEventRequest{EventId: eventID, UserId: actorID})); err != nil {
		t.Fatalf("re-register: %v", err)
	}
	if _, err := svc.CancelEvent(ctx, connect.NewRequest(&eventsv1.CancelEventRequest{EventId: eventID})); err != nil {
		t.Fatalf("cancel event: %v", err)
	}
	if _, err := registrations.RegisterForEvent(ctx, connect.NewRequest(&eventsv1.RegisterForEventRequest{EventId: eventID, UserId: actorID})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("cancelled event registration should be rejected, got %v", err)
	}
}
