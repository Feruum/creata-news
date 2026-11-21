//go:build integration

package events

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

func seedSundayOpenPublicationHost(t *testing.T, db *bun.DB) models.Organization {
	t.Helper()
	org := seedOrganization(t, db, "Sunday Open host")
	if _, err := db.NewInsert().Model(&models.OrganizationSlug{
		OrganizationID: org.ID,
		Slug:           "sunday-open",
		IsCurrent:      true,
	}).Exec(context.Background()); err != nil {
		t.Fatal(err)
	}
	return org
}

func TestCreateEventPublicationFollowsHost(t *testing.T) {
	for _, host := range []string{"club", "sunday", "renamed-sunday"} {
		t.Run(host, func(t *testing.T) {
			db := newTestDB(t)
			seedAuthUser(t, db, "publication-creator")
			// A matching display name alone must not grant Sunday Open publication.
			org := seedOrganization(t, db, "Sunday Open")
			if host != "club" {
				org = seedSundayOpenPublicationHost(t, db)
			}
			ctx := authedContext("publication-creator")
			if host == "renamed-sunday" {
				if _, err := db.ExecContext(ctx, "UPDATE organization_slugs SET is_current=false WHERE organization_id=?", org.ID); err != nil {
					t.Fatal(err)
				}
				if _, err := db.ExecContext(ctx, "INSERT INTO organization_slugs(organization_id,slug,is_current) VALUES(?,'renamed-host',true)", org.ID); err != nil {
					t.Fatal(err)
				}
				if _, err := db.ExecContext(ctx, "UPDATE organizations SET title='Renamed host' WHERE id=?", org.ID); err != nil {
					t.Fatal(err)
				}
			}
			svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
			on, off := true, false
			for i, targets := range [][2]*bool{{nil, nil}, {&on, &off}, {&off, &on}, {&on, &on}, {&off, &off}} {
				for _, draft := range []bool{false, true} {
					t.Run(fmt.Sprintf("flags-%d/draft-%t", i, draft), func(t *testing.T) {
						start := time.Now().Add(time.Hour)
						req := &eventsv1.CreateEventRequest{
							Title: fmt.Sprintf("Automatic host event %d draft %t", i, draft), OrganizationId: org.ID,
							StartTime: start.Format(time.RFC3339), EndTime: start.Add(time.Hour).Format(time.RFC3339),
							StructuredLocation: &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_ONLINE},
							PublishOnMap:       targets[0], PublishOnSundayOpen: targets[1],
						}
						if draft {
							status := eventsv1.EventStatus_EVENT_STATUS_DRAFT
							req.Status = &status
						}
						created, err := svc.CreateEvent(ctx, connect.NewRequest(req))
						if err != nil {
							t.Fatal(err)
						}
						wantSunday := host != "club"
						if !created.Msg.Event.PublishOnMap || created.Msg.Event.PublishOnSundayOpen != wantSunday {
							t.Fatalf("host did not determine publication: %v", created.Msg.Event)
						}
						wantStatus := eventsv1.EventStatus_EVENT_STATUS_PUBLISHED
						if draft {
							wantStatus = eventsv1.EventStatus_EVENT_STATUS_DRAFT
						}
						if created.Msg.Event.Status != wantStatus {
							t.Fatalf("publication assignment changed event status: %v", created.Msg.Event.Status)
						}
					})
				}
			}
		})
	}
}

func TestPublishEventReassignsLegacyDraftTargetsFromHost(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "publication-publisher")
	club := seedOrganization(t, db, "Club host")
	sunday := seedSundayOpenPublicationHost(t, db)
	ctx := authedContext("publication-publisher")
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	for _, org := range []models.Organization{club, sunday} {
		for _, targets := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
			event := seedDraftEvent(t, db, actor, org.ID, "Legacy draft", "Off campus")
			if _, err := db.ExecContext(ctx, "UPDATE events SET publish_on_map=?,publish_on_sunday_open=? WHERE id=?", targets[0], targets[1], event.ID); err != nil {
				t.Fatal(err)
			}
			published, err := svc.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: event.ID}))
			if err != nil {
				t.Fatal(err)
			}
			wantSunday := org.ID == sunday.ID
			if !published.Msg.Event.PublishOnMap || published.Msg.Event.PublishOnSundayOpen != wantSunday {
				t.Fatalf("published legacy targets instead of host targets: %v", published.Msg.Event)
			}
			var stored models.Event
			if err := db.NewSelect().Model(&stored).Where("id=?", event.ID).Scan(ctx); err != nil {
				t.Fatal(err)
			}
			if !stored.PublishOnMap || stored.PublishOnSundayOpen != wantSunday || stored.Status != models.EventStatusPublished || stored.PublishedAt == nil {
				t.Fatalf("publication targets and status were not persisted together: %v", stored)
			}
		}
	}
}

func TestUpdateEventPreservesStoredPublicationTargets(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "publication-editor")
	club := seedOrganization(t, db, "Legacy club")
	ctx := authedContext("publication-editor")
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	on, off := true, false
	for _, status := range []models.EventStatus{models.EventStatusDraft, models.EventStatusPublished} {
		for _, targets := range [][2]bool{{true, false}, {false, true}, {true, true}, {false, false}} {
			event := seedDraftEvent(t, db, actor, club.ID, "Legacy event", "Off campus")
			if _, err := db.ExecContext(ctx, "UPDATE events SET status=?,publish_on_map=?,publish_on_sunday_open=? WHERE id=?", status, targets[0], targets[1], event.ID); err != nil {
				t.Fatal(err)
			}
			for _, requested := range [][2]*bool{{nil, nil}, {&on, &off}, {&off, &on}, {&on, &on}, {&off, &off}} {
				description := "An ordinary event edit"
				updated, err := svc.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{
					Id: event.ID, Description: &description, PublishOnMap: requested[0], PublishOnSundayOpen: requested[1],
				}))
				if err != nil {
					t.Fatal(err)
				}
				if updated.Msg.Event.PublishOnMap != targets[0] || updated.Msg.Event.PublishOnSundayOpen != targets[1] {
					t.Fatalf("ordinary edit changed stored targets %v: %v", targets, updated.Msg.Event)
				}
				var stored models.Event
				if err := db.NewSelect().Model(&stored).Where("id=?", event.ID).Scan(ctx); err != nil {
					t.Fatal(err)
				}
				if stored.PublishOnMap != targets[0] || stored.PublishOnSundayOpen != targets[1] || stored.Status != status {
					t.Fatalf("ordinary edit changed stored publication: %v", stored)
				}
			}
		}
	}
}

func TestAcceptedCollaborationPublicationFollowsAllHosts(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "publication-collaborator")
	club := seedOrganization(t, db, "Collaborating club")
	sunday := seedSundayOpenPublicationHost(t, db)
	unrelated := seedOrganization(t, db, "Sunday Open")
	ctx := authedContext("publication-collaborator")
	if _, err := db.ExecContext(ctx, "UPDATE organization_slugs SET is_current=false WHERE organization_id=?", sunday.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO organization_slugs(organization_id,slug,is_current) VALUES(?,'sunday-open-2',true)", sunday.ID); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	for _, status := range []models.EventStatus{models.EventStatusDraft, models.EventStatusPublished} {
		for _, pair := range [][2]models.Organization{{sunday, club}, {club, sunday}, {club, unrelated}} {
			host, collaborator := pair[0], pair[1]
			t.Run(fmt.Sprintf("%s/host-%d/guest-%d", status, host.ID, collaborator.ID), func(t *testing.T) {
				event := seedDraftEvent(t, db, actor, host.ID, "Shared event", "Off campus")
				primarySunday := host.ID == sunday.ID
				if _, err := db.ExecContext(ctx, "UPDATE events SET status=?,publish_on_map=true,publish_on_sunday_open=? WHERE id=?", status, primarySunday, event.ID); err != nil {
					t.Fatal(err)
				}
				if err := db.NewSelect().Model(&event).WherePK().Scan(ctx); err != nil {
					t.Fatal(err)
				}
				invitation, err := svc.InviteOrganizationToCollaborate(ctx, connect.NewRequest(&eventsv1.InviteOrganizationToCollaborateRequest{EventId: event.ID, OrganizationId: collaborator.ID}))
				if err != nil {
					t.Fatal(err)
				}
				assertCollaborationPublication(t, db, event, true, primarySunday)
				if _, err := svc.AcceptCollaborationInvite(ctx, connect.NewRequest(&eventsv1.AcceptCollaborationInviteRequest{Token: invitation.Msg.Invite.Token})); err != nil {
					t.Fatal(err)
				}
				wantSunday := primarySunday || collaborator.ID == sunday.ID
				assertCollaborationPublication(t, db, event, true, wantSunday)
				if status == models.EventStatusDraft {
					published, err := svc.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: event.ID}))
					if err != nil {
						t.Fatal(err)
					}
					if published.Msg.Event.OrganizationId != host.ID || !published.Msg.Event.PublishOnMap || published.Msg.Event.PublishOnSundayOpen != wantSunday {
						t.Fatalf("accepted hosts not reflected in publication: %v", published.Msg.Event)
					}
					if err := db.NewSelect().Model(&event).WherePK().Scan(ctx); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := svc.RemoveEventCollaborator(ctx, connect.NewRequest(&eventsv1.RemoveEventCollaboratorRequest{EventId: event.ID, OrganizationId: collaborator.ID})); err != nil {
					t.Fatal(err)
				}
				assertCollaborationPublication(t, db, event, true, primarySunday)
			})
		}
	}
}

func assertCollaborationPublication(t *testing.T, db *bun.DB, before models.Event, onMap, onSunday bool) {
	t.Helper()
	var stored models.Event
	if err := db.NewSelect().Model(&stored).Where("id=?", before.ID).Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	before.PublishOnMap = onMap
	before.PublishOnSundayOpen = onSunday
	if !reflect.DeepEqual(stored, before) {
		t.Fatalf("collaboration must only update publication targets: got %+v, want %+v", stored, before)
	}
}

func TestSundayOpenCollaborationPublicationRollsBackWithMutation(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "publication-rollback")
	club := seedOrganization(t, db, "Club host")
	sunday := seedSundayOpenPublicationHost(t, db)
	event := seedDraftEvent(t, db, actor, club.ID, "Atomic collaboration", "Off campus")
	ctx := authedContext("publication-rollback")
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	invitation, err := svc.InviteOrganizationToCollaborate(ctx, connect.NewRequest(&eventsv1.InviteOrganizationToCollaborateRequest{EventId: event.ID, OrganizationId: sunday.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE events ADD CONSTRAINT reject_collaboration_publication CHECK (NOT publish_on_sunday_open) NOT VALID"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AcceptCollaborationInvite(ctx, connect.NewRequest(&eventsv1.AcceptCollaborationInviteRequest{Token: invitation.Msg.Invite.Token})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("acceptance must fail when publication cannot be saved: %v", err)
	}
	var invite models.EventCollaborationInvite
	if err := db.NewSelect().Model(&invite).Where("id=?", invitation.Msg.Invite.Id).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	count, err := db.NewSelect().Model((*models.EventCollaborator)(nil)).Where("event_id=?", event.ID).Count(ctx)
	if err != nil || count != 0 || invite.Status != models.CollaborationInviteStatusPending || invite.RespondedAt != nil {
		t.Fatalf("failed publication must roll back acceptance: invite=%v collaborators=%d err=%v", invite, count, err)
	}
	assertCollaborationPublication(t, db, event, true, false)
	if _, err := db.ExecContext(ctx, "ALTER TABLE events DROP CONSTRAINT reject_collaboration_publication"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AcceptCollaborationInvite(ctx, connect.NewRequest(&eventsv1.AcceptCollaborationInviteRequest{Token: invitation.Msg.Invite.Token})); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE events ADD CONSTRAINT reject_collaboration_publication CHECK (publish_on_sunday_open) NOT VALID"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RemoveEventCollaborator(ctx, connect.NewRequest(&eventsv1.RemoveEventCollaboratorRequest{EventId: event.ID, OrganizationId: sunday.ID})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("removal must fail when publication cannot be saved: %v", err)
	}
	count, err = db.NewSelect().Model((*models.EventCollaborator)(nil)).Where("event_id=?", event.ID).Count(ctx)
	if err != nil || count != 1 {
		t.Fatalf("failed publication must roll back collaborator removal: count=%d err=%v", count, err)
	}
	assertCollaborationPublication(t, db, event, true, true)
	if _, err := svc.RevokeCollaborationInvite(ctx, connect.NewRequest(&eventsv1.RevokeCollaborationInviteRequest{InviteId: invitation.Msg.Invite.Id})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("revocation must fail when publication cannot be saved: %v", err)
	}
	if err := db.NewSelect().Model(&invite).WherePK().Scan(ctx); err != nil || invite.Status != models.CollaborationInviteStatusAccepted {
		t.Fatalf("failed publication must roll back consent revocation: invite=%v err=%v", invite, err)
	}
	assertCollaborationPublication(t, db, event, true, true)
}

func TestSundayOpenCollaborationRequiresCurrentAcceptedConsent(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "publication-consent")
	club := seedOrganization(t, db, "Club host")
	sunday := seedSundayOpenPublicationHost(t, db)
	ctx := authedContext("publication-consent")
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	for _, action := range []string{"direct", "pending", "declined", "revoked", "expired", "accepted-expired", "accepted-revoked", "removed-readded"} {
		t.Run(action, func(t *testing.T) {
			event := seedDraftEvent(t, db, actor, club.ID, "Consent "+action, "Off campus")
			if action == "direct" {
				if _, err := svc.AddEventCollaborator(ctx, connect.NewRequest(&eventsv1.AddEventCollaboratorRequest{EventId: event.ID, OrganizationId: sunday.ID})); err != nil {
					t.Fatal(err)
				}
			} else {
				invitation, err := svc.InviteOrganizationToCollaborate(ctx, connect.NewRequest(&eventsv1.InviteOrganizationToCollaborateRequest{EventId: event.ID, OrganizationId: sunday.ID}))
				if err != nil {
					t.Fatal(err)
				}
				token := invitation.Msg.Invite.Token
				switch action {
				case "declined":
					if _, err := svc.DeclineCollaborationInvite(ctx, connect.NewRequest(&eventsv1.DeclineCollaborationInviteRequest{Token: token})); err != nil {
						t.Fatal(err)
					}
				case "revoked":
					if _, err := svc.RevokeCollaborationInvite(ctx, connect.NewRequest(&eventsv1.RevokeCollaborationInviteRequest{InviteId: invitation.Msg.Invite.Id})); err != nil {
						t.Fatal(err)
					}
				case "expired":
					if _, err := db.ExecContext(ctx, "UPDATE event_collaboration_invites SET expires_at=NOW()-interval '1 day' WHERE id=?", invitation.Msg.Invite.Id); err != nil {
						t.Fatal(err)
					}
					if _, err := svc.AcceptCollaborationInvite(ctx, connect.NewRequest(&eventsv1.AcceptCollaborationInviteRequest{Token: token})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
						t.Fatalf("expired consent accepted: %v", err)
					}
				case "accepted-expired", "accepted-revoked", "removed-readded":
					if _, err := svc.AcceptCollaborationInvite(ctx, connect.NewRequest(&eventsv1.AcceptCollaborationInviteRequest{Token: token})); err != nil {
						t.Fatal(err)
					}
					assertCollaborationPublication(t, db, event, true, true)
					if action == "accepted-expired" {
						if _, err := db.ExecContext(ctx, "UPDATE event_collaboration_invites SET expires_at=NOW()-interval '1 day' WHERE id=?", invitation.Msg.Invite.Id); err != nil {
							t.Fatal(err)
						}
					} else if action == "accepted-revoked" {
						if _, err := svc.RevokeCollaborationInvite(ctx, connect.NewRequest(&eventsv1.RevokeCollaborationInviteRequest{InviteId: invitation.Msg.Invite.Id})); err != nil {
							t.Fatal(err)
						}
						assertCollaborationPublication(t, db, event, true, false)
					} else {
						if _, err := svc.RemoveEventCollaborator(ctx, connect.NewRequest(&eventsv1.RemoveEventCollaboratorRequest{EventId: event.ID, OrganizationId: sunday.ID})); err != nil {
							t.Fatal(err)
						}
						if _, err := svc.AddEventCollaborator(ctx, connect.NewRequest(&eventsv1.AddEventCollaboratorRequest{EventId: event.ID, OrganizationId: sunday.ID})); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
			published, err := svc.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: event.ID}))
			wantSunday := action == "accepted-expired"
			if err != nil || !published.Msg.Event.PublishOnMap || published.Msg.Event.PublishOnSundayOpen != wantSunday {
				t.Fatalf("only current accepted consent may publish on Sunday Open: event=%v err=%v", published, err)
			}
		})
	}
}

func TestClubCannotCreateEventForSundayOpenHost(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "publication-club-manager")
	club := seedOrganization(t, db, "Managed club")
	sunday := seedSundayOpenPublicationHost(t, db)
	ctx := authedContext("publication-club-manager")
	if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id,role) VALUES(?,'none')", actor); err != nil {
		t.Fatal(err)
	}
	var roleID int32
	if err := db.NewRaw("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'publication-club-manager','Club manager',true) RETURNING id", club.OrganizationTypeID).Scan(ctx, &roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'create_event')", roleID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO organization_members(organization_id,user_id,role_template_id) VALUES(?,?,?)", club.ID, actor, roleID); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	start := time.Now().Add(time.Hour)
	on := true
	req := &eventsv1.CreateEventRequest{
		Title: "Club manager event", OrganizationId: club.ID,
		StartTime: start.Format(time.RFC3339), EndTime: start.Add(time.Hour).Format(time.RFC3339),
		StructuredLocation:  &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_ONLINE},
		PublishOnSundayOpen: &on,
	}
	created, err := svc.CreateEvent(ctx, connect.NewRequest(req))
	if err != nil {
		t.Fatalf("club manager could not create for their own club: %v", err)
	}
	invitation, err := svc.InviteOrganizationToCollaborate(ctx, connect.NewRequest(&eventsv1.InviteOrganizationToCollaborateRequest{EventId: created.Msg.Event.Id, OrganizationId: sunday.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AcceptCollaborationInvite(ctx, connect.NewRequest(&eventsv1.AcceptCollaborationInviteRequest{Token: invitation.Msg.Invite.Token})); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("club manager accepted consent on behalf of Sunday Open: %v", err)
	}
	var stored models.Event
	if err := db.NewSelect().Model(&stored).Where("id=?", created.Msg.Event.Id).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if stored.PublishOnSundayOpen || stored.OrganizationID != club.ID || stored.UserID != actor {
		t.Fatalf("unauthorized acceptance changed publication or ownership: %v", stored)
	}
	req.OrganizationId = sunday.ID
	if _, err := svc.CreateEvent(ctx, connect.NewRequest(req)); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("club manager created an event for Sunday Open: %v", err)
	}
}
