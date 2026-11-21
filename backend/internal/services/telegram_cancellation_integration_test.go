//go:build integration

package services

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestTelegramCancellationOnlyUsesTrustedLinkedSender(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	owner := seedAuthUser(t, db, "telegram-linked-owner")
	other := seedAuthUser(t, db, "telegram-other-owner")
	event, _, _ := seedEvent(t, db, owner)
	if _, err := db.ExecContext(ctx, "UPDATE users SET telegram_id=101 WHERE id=?", owner); err != nil {
		t.Fatal(err)
	}
	reg := models.EventRegistration{EventID: event, UserID: owner, Status: models.RegistrationStatusRegistered}
	otherReg := models.EventRegistration{EventID: event, UserID: other, Status: models.RegistrationStatusRegistered}
	for _, registration := range []*models.EventRegistration{&reg, &otherReg} {
		if _, err := db.NewInsert().Model(registration).Returning("*").Exec(ctx); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewEventRegistrationsService(db, events.NewDurablePublisher(db, nil), authz.NewEnforcer(db, nil), "telegram-workflow-internal-secret")
	request := func(secret, sender string, body *eventsv1.CancelRegistrationRequest) *connect.Request[eventsv1.CancelRegistrationRequest] {
		req := connect.NewRequest(body)
		req.Header().Set("X-Internal-Secret", secret)
		req.Header().Set("X-Telegram-User-Id", sender)
		return req
	}
	const secret = "telegram-workflow-internal-secret"
	for _, sender := range []string{"", "0", "-1", "abc", "999"} {
		if _, err := svc.CancelRegistration(ctx, request(secret, sender, &eventsv1.CancelRegistrationRequest{EventId: &event})); err == nil {
			t.Errorf("invalid or unlinked sender %q was accepted", sender)
		}
	}
	if _, err := svc.CancelRegistration(ctx, request("wrong-secret", "101", &eventsv1.CancelRegistrationRequest{EventId: &event})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("invalid internal credential accepted: %v", err)
	}
	for _, body := range []*eventsv1.CancelRegistrationRequest{
		{EventId: &event, UserId: &other}, {RegistrationId: &otherReg.ID},
	} {
		if _, err := svc.CancelRegistration(ctx, request(secret, "101", body)); err == nil {
			t.Fatal("trusted bot was allowed to cancel another user's registration")
		}
	}
	response, err := svc.CancelRegistration(ctx, request(secret, "101", &eventsv1.CancelRegistrationRequest{EventId: &event}))
	if err != nil || !response.Msg.Success {
		t.Fatalf("trusted linked sender cannot cancel own registration: response=%v err=%v", response, err)
	}
	for _, registration := range []*models.EventRegistration{&reg, &otherReg} {
		if err := db.NewSelect().Model(registration).WherePK().Scan(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if reg.Status != models.RegistrationStatusCancelled || otherReg.Status != models.RegistrationStatusRegistered {
		t.Fatalf("wrong registration cancelled: own=%s other=%s", reg.Status, otherReg.Status)
	}
	var payload struct {
		ParticipantUserID int32 `json:"participant_user_id"`
	}
	if err := db.NewRaw("SELECT (payload->>'participant_user_id')::integer FROM domain_event_outbox WHERE topic=?", events.TopicRegistrationCancelled).Scan(ctx, &payload.ParticipantUserID); err != nil || payload.ParticipantUserID != owner {
		t.Fatalf("cancellation must durably notify own user %d: payload=%v err=%v", owner, payload, err)
	}
	if _, err := svc.CancelRegistration(ctx, request(secret, fmt.Sprint(101), &eventsv1.CancelRegistrationRequest{EventId: &event})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("repeated cancellation should not duplicate durable event: %v", err)
	}
}
