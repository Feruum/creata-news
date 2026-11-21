//go:build integration

package events

import (
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestInviteOrganizationToCollaboratePublishesCollaborationInvited(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-collaboration-invite"
	actorID := seedAuthUser(t, db, kratosID)
	hostOrg := seedOrganization(t, db, "Host Organization")
	invitedOrg := seedOrganization(t, db, "Invited Organization")
	event := seedDraftEvent(t, db, actorID, hostOrg.ID, "AI Hackathon", "Innovation Center")

	cap := &capturePublisher{}
	pub := domainevents.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, nil, pub, "https://map.studyverse.test", "test-reminder-secret")
	ctx := authedContext(kratosID)

	resp, err := svc.InviteOrganizationToCollaborate(ctx, connect.NewRequest(&eventsv1.InviteOrganizationToCollaborateRequest{
		EventId:        event.ID,
		OrganizationId: invitedOrg.ID,
	}))
	if err != nil {
		t.Fatalf("InviteOrganizationToCollaborate() error = %v", err)
	}
	if resp.Msg == nil || resp.Msg.Invite == nil {
		t.Fatal("expected invite response, got nil")
	}
	if len(cap.topics) != 1 {
		t.Fatalf("publish call count = %d, want 1", len(cap.topics))
	}
	if cap.topics[0] != domainevents.TopicCollaborationInvited {
		t.Fatalf("published topic = %q, want %q", cap.topics[0], domainevents.TopicCollaborationInvited)
	}
	if len(cap.msgs) != 1 || cap.msgs[0] == nil {
		t.Fatal("expected one published message")
	}

	var got domainevents.CollaborationInvitedPayload
	if err := json.Unmarshal(cap.msgs[0].Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if got.EventID != event.ID {
		t.Fatalf("payload EventID = %d, want %d", got.EventID, event.ID)
	}
	if got.EventTitle != event.Title {
		t.Fatalf("payload EventTitle = %q, want %q", got.EventTitle, event.Title)
	}
	if got.OrganizationID != invitedOrg.ID {
		t.Fatalf("payload OrganizationID = %d, want %d", got.OrganizationID, invitedOrg.ID)
	}
	if got.HostOrganizationID != hostOrg.ID {
		t.Fatalf("payload HostOrganizationID = %d, want %d", got.HostOrganizationID, hostOrg.ID)
	}
	if got.InviteToken == "" {
		t.Fatal("expected non-empty invite token in payload")
	}
	if got.InviteToken != resp.Msg.Invite.Token {
		t.Fatalf("payload InviteToken = %q, response Invite.Token = %q", got.InviteToken, resp.Msg.Invite.Token)
	}
	if got.MapAppURL != "https://map.studyverse.test" {
		t.Fatalf("payload MapAppURL = %q, want %q", got.MapAppURL, "https://map.studyverse.test")
	}
	if got.ActorID != actorID {
		t.Fatalf("payload ActorID = %d, want %d", got.ActorID, actorID)
	}
}
