//go:build integration

package sundayopen

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/studyverse/ems-backend/gen/sundayopen/v1"
	"github.com/studyverse/ems-backend/internal/registrationforms"
)

func TestLegacySundayRegistrationCannotBypassPublishedForm(t *testing.T) {
	db := publicEventsFixture(t)
	ctx := context.Background()
	for _, table := range []string{"event_registration_forms", "event_registration_form_versions"} {
		if _, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO event_registration_form_versions(id,event_id,definition) VALUES('27c85842-e46a-4579-99e6-d5193a9a2584',1,'{"eventId":1,"versionId":"27c85842-e46a-4579-99e6-d5193a9a2584","approvalMode":"APPROVAL_MODE_MANUAL"}'); INSERT INTO event_registration_forms(event_id,draft,revision,published_version_id) VALUES(1,'{}',1,'27c85842-e46a-4579-99e6-d5193a9a2584')`); err != nil {
		t.Fatal(err)
	}
	s := NewService(db, nil).WithForms(registrationforms.NewService(db, nil, nil))
	_, err := s.RegisterExternally(ctx, connect.NewRequest(&pb.RegisterExternallyRequest{EventId: 1, FullName: "Legacy visitor", Phone: "+77010000000", Email: "visitor@example.test"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("contacts-only flow bypassed form: %v", err)
	}
	var count int
	if err := db.NewRaw(`SELECT count(*) FROM external_registrations WHERE event_id=1`).Scan(ctx, &count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("legacy flow reserved a seat before account form submission")
	}
}
