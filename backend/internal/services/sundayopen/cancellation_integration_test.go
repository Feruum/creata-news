//go:build integration

package sundayopen

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"connectrpc.com/connect"
	pbv1 "github.com/studyverse/ems-backend/gen/sundayopen/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestGuestCancellationCapabilityChangesAfterReactivation(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	for _, statement := range []string{
		`CREATE TABLE events (LIKE public.events INCLUDING ALL)`,
		`CREATE TABLE organization_slugs (LIKE public.organization_slugs INCLUDING ALL)`,
		`INSERT INTO organization_slugs(organization_id,slug,is_current) VALUES(1,'sunday-open',true)`,
		`CREATE TABLE event_registrations (LIKE public.event_registrations INCLUDING ALL)`,
		`CREATE TABLE external_registrations (LIKE public.external_registrations INCLUDING ALL)`,
		`INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,capacity,check_in_secret,check_in_token,publish_on_sunday_open) VALUES(3,'Guest cancellation','{}',1,1,NOW(),NOW()+INTERVAL '1 hour','{}','published',1,'test-secret','test-token',true)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	s := NewService(db, authz.NewEnforcer(db, nil), strings.Repeat("s", 32))
	register := func() string {
		t.Helper()
		response, err := s.RegisterExternally(ctx, connect.NewRequest(&pbv1.RegisterExternallyRequest{EventId: 3, Phone: "phone", Email: "mail", FullName: "Guest"}))
		if err != nil {
			t.Fatal(err)
		}
		if response.Msg.CancellationToken == "" {
			t.Fatal("missing cancellation capability")
		}
		return response.Msg.CancellationToken
	}
	cancel := func(token string) error {
		_, err := s.CancelExternalRegistration(ctx, connect.NewRequest(&pbv1.CancelExternalRegistrationRequest{EventId: 3, Phone: "phone", CancellationToken: token}))
		return err
	}
	oldToken := register()
	if err := cancel(oldToken); err != nil {
		t.Fatal(err)
	}
	newToken := register()
	if oldToken == newToken {
		t.Fatal("reactivation reused cancellation capability")
	}
	if err := cancel(oldToken); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("old token cancelled new booking: %v", err)
	}
	if err := cancel(newToken); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyPhoneCapabilityCannotTakeOverAccountRegistration(t *testing.T) {
	db := publicEventsFixture(t)
	ctx := context.Background()
	secret := strings.Repeat("s", 32)
	if _, err := db.ExecContext(ctx, `INSERT INTO external_registrations(event_id,sunday_open_account_id,full_name,phone,email,status,snapshot) VALUES(1,'00000000-0000-0000-0000-000000000001','Victim','+77001111111','victim@example.test','cancelled','{"title":"Victim event snapshot"}')`); err != nil {
		t.Fatal(err)
	}
	var before models.ExternalRegistration
	if err := db.NewSelect().Model(&before).Where("event_id=1").Scan(ctx); err != nil {
		t.Fatal(err)
	}
	s := NewService(db, nil, secret)
	response, err := s.RegisterExternally(ctx, connect.NewRequest(&pbv1.RegisterExternallyRequest{EventId: 1, FullName: "Attacker", Phone: "+77001111111", Email: "attacker@example.test"}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists || response != nil {
		t.Fatalf("legacy signup restored account booking or minted token; error: %v", err)
	}
	token, err := signCancellationToken(secret, before.ID, 1, "+77001111111", before.RegisteredAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE external_registrations SET status='registered' WHERE id=?`, before.ID); err != nil {
		t.Fatal(err)
	}
	_, err = s.CancelExternalRegistration(ctx, connect.NewRequest(&pbv1.CancelExternalRegistrationRequest{EventId: 1, Phone: "+77001111111", CancellationToken: token}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("legacy capability cancelled account booking: %v", err)
	}
	var after models.ExternalRegistration
	if err := db.NewSelect().Model(&after).Where("id=?", before.ID).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if after.FullName != before.FullName || after.Email != before.Email || after.Status != "registered" || !after.RegisteredAt.Equal(before.RegisteredAt) || string(after.Snapshot) != string(before.Snapshot) {
		a, _ := json.Marshal(after)
		t.Fatalf("legacy API mutated protected account row: %s", a)
	}
}
