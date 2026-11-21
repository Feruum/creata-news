//go:build integration

package events

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestCollaborationInvitationResponseLifecycle(t *testing.T) {
	for _, action := range []string{"accept", "decline", "revoke", "expired"} {
		t.Run(action, func(t *testing.T) {
			db := newTestDB(t)
			actor := seedAuthUser(t, db, "invite-workflow-user")
			host := seedOrganization(t, db, "Host club")
			guest := seedOrganization(t, db, "Guest club")
			event := seedDraftEvent(t, db, actor, host.ID, "Shared workshop", "Hall")
			ctx := authedContext("invite-workflow-user")
			svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, domainevents.NewDurablePublisher(db, nil), "", "")
			inviteRequest := connect.NewRequest(&eventsv1.InviteOrganizationToCollaborateRequest{EventId: event.ID, OrganizationId: guest.ID})
			invitation, err := svc.InviteOrganizationToCollaborate(ctx, inviteRequest)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := svc.InviteOrganizationToCollaborate(ctx, inviteRequest); connect.CodeOf(err) != connect.CodeAlreadyExists {
				t.Fatalf("duplicate pending invite accepted: %v", err)
			}
			token := invitation.Msg.Invite.Token
			expectedStatus := models.CollaborationInviteStatusAccepted
			switch action {
			case "accept":
				accepted, err := svc.AcceptCollaborationInvite(ctx, connect.NewRequest(&eventsv1.AcceptCollaborationInviteRequest{Token: token}))
				if err != nil {
					t.Fatal(err)
				}
				addedAt, err := time.Parse(time.RFC3339, accepted.Msg.Collaborator.AddedAt)
				if err != nil || addedAt.Before(time.Now().Add(-time.Minute)) {
					t.Fatalf("collaborator has no current addition timestamp: %v", accepted.Msg.Collaborator)
				}
			case "decline":
				expectedStatus = models.CollaborationInviteStatusDeclined
				if _, err := svc.DeclineCollaborationInvite(ctx, connect.NewRequest(&eventsv1.DeclineCollaborationInviteRequest{Token: token})); err != nil {
					t.Fatal(err)
				}
			case "revoke":
				expectedStatus = models.CollaborationInviteStatusRevoked
				if _, err := svc.RevokeCollaborationInvite(ctx, connect.NewRequest(&eventsv1.RevokeCollaborationInviteRequest{InviteId: invitation.Msg.Invite.Id})); err != nil {
					t.Fatal(err)
				}
			case "expired":
				expectedStatus = models.CollaborationInviteStatusPending
				if _, err := db.ExecContext(ctx, "UPDATE event_collaboration_invites SET expires_at=NOW()-interval '1 minute' WHERE token=?", token); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := svc.AcceptCollaborationInvite(ctx, connect.NewRequest(&eventsv1.AcceptCollaborationInviteRequest{Token: token})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
				t.Fatalf("repeated or invalid acceptance accepted: %v", err)
			}
			var invite models.EventCollaborationInvite
			if err := db.NewSelect().Model(&invite).Where("token=?", token).Scan(ctx); err != nil || invite.Status != expectedStatus {
				t.Fatalf("invitation response not persisted: invite=%v err=%v", invite, err)
			}
			collaborators, err := db.NewSelect().Model((*models.EventCollaborator)(nil)).Where("event_id=?", event.ID).Count(ctx)
			wanted := 0
			if action == "accept" {
				wanted = 1
			}
			if err != nil || collaborators != wanted {
				t.Fatalf("unexpected collaborators: got %d, want %d, err=%v", collaborators, wanted, err)
			}
			if _, err := db.ExecContext(ctx, "UPDATE events SET status='published' WHERE id=?", event.ID); err != nil {
				t.Fatal(err)
			}
			listed, err := svc.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{OrganizationId: &guest.ID}))
			if err != nil || int(listed.Msg.Total) != wanted || len(listed.Msg.Events) != wanted {
				t.Fatalf("collaboration response not reflected in club events: got %v, want %d, err=%v", listed, wanted, err)
			}
		})
	}
}
