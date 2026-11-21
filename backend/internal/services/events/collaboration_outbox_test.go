//go:build integration

package events

import (
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestCollaborationAndCancellationRollBackFailedEnqueue(t *testing.T) {
	for _, action := range []string{"invite", "accept", "decline", "cancel"} {
		t.Run(action, func(t *testing.T) {
			db := newTestDB(t)
			kratosID := "outbox-collaboration"
			actorID := seedAuthUser(t, db, kratosID)
			host := seedOrganization(t, db, "Host")
			invited := seedOrganization(t, db, "Invited")
			if action == "accept" {
				invited = seedSundayOpenPublicationHost(t, db)
			}
			event := seedDraftEvent(t, db, actorID, host.ID, "Test Event", "Test Room")
			ctx := authedContext(kratosID)
			enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
			plain := NewService(db, nil, enforcer, nil, nil, "", "")
			var token string
			if action == "accept" || action == "decline" {
				response, err := plain.InviteOrganizationToCollaborate(ctx, connect.NewRequest(&eventsv1.InviteOrganizationToCollaborateRequest{EventId: event.ID, OrganizationId: invited.ID}))
				if err != nil {
					t.Fatal(err)
				}
				token = response.Msg.Invite.Token
			}
			if _, err := db.ExecContext(ctx, "ALTER TABLE domain_event_outbox ADD CONSTRAINT reject_new_events CHECK (false) NOT VALID"); err != nil {
				t.Fatal(err)
			}
			svc := NewService(db, nil, enforcer, nil, domainevents.NewDurablePublisher(db, &capturePublisher{}), "", "")
			var mutationErr error
			switch action {
			case "invite":
				_, mutationErr = svc.InviteOrganizationToCollaborate(ctx, connect.NewRequest(&eventsv1.InviteOrganizationToCollaborateRequest{EventId: event.ID, OrganizationId: invited.ID}))
			case "accept":
				_, mutationErr = svc.AcceptCollaborationInvite(ctx, connect.NewRequest(&eventsv1.AcceptCollaborationInviteRequest{Token: token}))
			case "decline":
				_, mutationErr = svc.DeclineCollaborationInvite(ctx, connect.NewRequest(&eventsv1.DeclineCollaborationInviteRequest{Token: token}))
			case "cancel":
				_, mutationErr = svc.CancelEvent(ctx, connect.NewRequest(&eventsv1.CancelEventRequest{EventId: event.ID}))
			}
			if connect.CodeOf(mutationErr) != connect.CodeInternal {
				t.Fatalf("mutation should reject failed enqueue, got %v", mutationErr)
			}
			if action == "cancel" {
				if err := db.NewSelect().Model(&event).WherePK().Scan(ctx); err != nil {
					t.Fatal(err)
				}
				if event.Status != models.EventStatusDraft || event.CancelledAt != nil {
					t.Fatalf("cancellation escaped rollback: %+v", event)
				}
			} else if action == "invite" {
				count, err := db.NewSelect().Model((*models.EventCollaborationInvite)(nil)).Where("event_id = ?", event.ID).Count(ctx)
				if err != nil || count != 0 {
					t.Fatalf("invitation escaped rollback: count=%d err=%v", count, err)
				}
			} else {
				var invite models.EventCollaborationInvite
				if err := db.NewSelect().Model(&invite).Where("token = ?", token).Scan(ctx); err != nil {
					t.Fatal(err)
				}
				if invite.Status != models.CollaborationInviteStatusPending || invite.RespondedAt != nil {
					t.Fatalf("response escaped rollback: %+v", invite)
				}
				count, err := db.NewSelect().Model((*models.EventCollaborator)(nil)).Where("event_id = ?", event.ID).Count(ctx)
				if err != nil || count != 0 {
					t.Fatalf("collaborator escaped rollback: count=%d err=%v", count, err)
				}
				if err := db.NewSelect().Model(&event).WherePK().Scan(ctx); err != nil || event.PublishOnSundayOpen {
					t.Fatalf("publication escaped failed acceptance rollback: event=%v err=%v", event, err)
				}
			}
		})
	}
}
