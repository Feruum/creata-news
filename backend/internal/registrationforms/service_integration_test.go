//go:build integration

package registrationforms

import (
	"context"
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/registrationfiles"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

func formsFixture(t *testing.T) (*Service, context.Context) {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL must name a disposable PostgreSQL database")
	}
	db := testsupport.IsolatedDB(t)
	ctx := context.WithValue(context.Background(), auth.UserIDKey, "forms-owner")
	ddl := `CREATE TABLE users (LIKE public.users INCLUDING ALL);
 CREATE TABLE events (LIKE public.events INCLUDING ALL);
 CREATE TABLE event_registrations(id SERIAL PRIMARY KEY,event_id INTEGER,user_id INTEGER,status TEXT,registered_at TIMESTAMPTZ DEFAULT NOW(),updated_at TIMESTAMPTZ DEFAULT NOW());
 CREATE TABLE sunday_open_accounts(id UUID PRIMARY KEY,telegram_username TEXT);
 CREATE TABLE external_registrations(id SERIAL PRIMARY KEY,event_id INTEGER,full_name TEXT,email TEXT,phone TEXT,status TEXT,registered_at TIMESTAMPTZ DEFAULT NOW(),checked_in_at TIMESTAMPTZ,sunday_open_account_id UUID,snapshot JSONB DEFAULT '{}',updated_at TIMESTAMPTZ DEFAULT NOW());
 CREATE TABLE event_attendance(registration_id INTEGER,checked_in_at TIMESTAMPTZ);
 CREATE TABLE event_registration_forms(event_id INTEGER PRIMARY KEY,draft JSONB NOT NULL DEFAULT '{}',revision INTEGER NOT NULL DEFAULT 0,published_version_id UUID,updated_by INTEGER,updated_at TIMESTAMPTZ DEFAULT NOW());
 CREATE TABLE event_registration_form_versions(id UUID PRIMARY KEY,event_id INTEGER,definition JSONB,created_by INTEGER,created_at TIMESTAMPTZ DEFAULT NOW());
 CREATE TABLE event_registration_responses(id BIGSERIAL PRIMARY KEY,event_id INTEGER,form_version_id UUID,registration_id INTEGER,external_registration_id INTEGER,answers JSONB DEFAULT '[]',contacts JSONB DEFAULT '{}',status TEXT,is_current BOOLEAN DEFAULT TRUE,notes TEXT DEFAULT '',reviewed_by INTEGER,reviewed_at TIMESTAMPTZ,submitted_at TIMESTAMPTZ DEFAULT NOW());
 CREATE UNIQUE INDEX current_map ON event_registration_responses(registration_id) WHERE is_current=true;
 CREATE UNIQUE INDEX current_external ON event_registration_responses(external_registration_id) WHERE is_current=true;
 CREATE TABLE event_registration_response_reads(event_id INTEGER,user_id INTEGER,latest_response_id BIGINT DEFAULT 0,PRIMARY KEY(event_id,user_id));
 CREATE TABLE event_registration_uploads(id UUID,field_id TEXT,file_name TEXT,content_type TEXT,size_bytes BIGINT,response_id BIGINT,state TEXT,created_at TIMESTAMPTZ);
 INSERT INTO users(id,kratos_id,username,email,password) VALUES(1,'forms-owner','Owner','owner@example.test','test');
 INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,capacity,check_in_secret,check_in_token,publish_on_map,publish_on_sunday_open) VALUES(1,'Forms','{}',1,1,NOW(),NOW()+interval '1 day','{}','published',1,'forms-secret','forms-token',true,true);`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	return NewService(db, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil), ctx
}

func TestRegistrationContactsUseLinkedTelegramAccounts(t *testing.T) {
	s, ctx := formsFixture(t)
	if _, err := s.db.ExecContext(ctx, `UPDATE users SET telegram_username='map_person' WHERE id=1;
 INSERT INTO sunday_open_accounts(id,telegram_username) VALUES('00000000-0000-0000-0000-000000000099','sunday_person');
 INSERT INTO event_registrations(id,event_id,user_id,status) VALUES(1,1,1,'registered');
 INSERT INTO external_registrations(id,event_id,full_name,email,status,sunday_open_account_id) VALUES
 (1,1,'Sunday','sunday@example.test','registered','00000000-0000-0000-0000-000000000099'),
 (2,1,'Guest','guest@example.test','registered',NULL),
 (3,2,'Other event','other@example.test','registered','00000000-0000-0000-0000-000000000099');
 INSERT INTO event_registration_responses(event_id,registration_id,contacts,status) VALUES
 (1,1,'{"fullName":"Saved name","email":"saved@example.test"}','registered');`); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		name     string
		source   pb.RegistrationSource
		id       int32
		username string
	}{
		{"map", pb.RegistrationSource_REGISTRATION_SOURCE_MAP, 1, "map_person"},
		{"sunday", pb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN, 1, "sunday_person"},
		{"guest", pb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN, 2, ""},
	} {
		t.Run(entry.name, func(t *testing.T) {
			detail, err := s.GetEventResponse(ctx, connect.NewRequest(&pb.GetEventResponseRequest{
				EventId: 1, Registration: &pb.RegistrationRef{Source: entry.source, Id: entry.id},
			}))
			if err != nil {
				t.Fatal(err)
			}
			if got := detail.Msg.Response.Contact.TelegramUsername; got != entry.username {
				t.Fatalf("telegram username = %q, want %q", got, entry.username)
			}
			if entry.name == "map" && detail.Msg.Response.Contact.FullName != "Saved name" {
				t.Fatal("account enrichment replaced the saved registration contact")
			}
		})
	}
	listing, err := s.ListEventResponses(ctx, connect.NewRequest(&pb.ListEventResponsesRequest{EventId: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if listing.Msg.Total != 3 {
		t.Fatalf("list escaped the event scope: %d", listing.Msg.Total)
	}
	if _, err := s.GetEventResponse(ctx, connect.NewRequest(&pb.GetEventResponseRequest{
		EventId: 1, Registration: &pb.RegistrationRef{Source: pb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN, Id: 3},
	})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("another event's contact was accessible: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE users SET telegram_username=NULL WHERE id=1;
 UPDATE event_registration_responses SET contacts=contacts||'{"telegramUsername":"stale_person"}'::jsonb WHERE registration_id=1`); err != nil {
		t.Fatal(err)
	}
	detail, err := s.GetEventResponse(ctx, connect.NewRequest(&pb.GetEventResponseRequest{
		EventId: 1, Registration: &pb.RegistrationRef{Source: pb.RegistrationSource_REGISTRATION_SOURCE_MAP, Id: 1},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if detail.Msg.Response.Contact.TelegramUsername != "" {
		t.Fatal("unlinked Telegram account remained visible")
	}
}

func TestDraftPublishVersionsAndStaleSubmissions(t *testing.T) {
	s, ctx := formsFixture(t)
	f := &pb.RegistrationForm{EventId: 1, ApprovalMode: pb.ApprovalMode_APPROVAL_MODE_MANUAL, Fields: []*pb.FormField{{Id: "reason", Label: "Reason", Type: pb.FormFieldType_FORM_FIELD_TYPE_SHORT_TEXT, Required: true}}}
	saved, err := s.SaveFormDraft(ctx, connect.NewRequest(&pb.SaveFormDraftRequest{Form: f}))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Msg.Form.Revision != 1 {
		t.Fatal("revision did not advance")
	}
	if _, err := s.SaveFormDraft(ctx, connect.NewRequest(&pb.SaveFormDraftRequest{Form: f})); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale save accepted: %v", err)
	}
	published, err := s.PublishRegistrationForm(ctx, connect.NewRequest(&pb.PublishRegistrationFormRequest{EventId: 1, ExpectedRevision: 1}))
	if err != nil {
		t.Fatal(err)
	}
	original := published.Msg.Form.VersionId
	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error { _, err := s.Prepare(ctx, tx, 1, nil); return err }); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing form accepted: %v", err)
	}
	f.Fields[0].Label = "Updated reason"
	if _, err := s.SaveFormDraft(ctx, connect.NewRequest(&pb.SaveFormDraftRequest{Form: f, ExpectedRevision: 1})); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PublishRegistrationForm(ctx, connect.NewRequest(&pb.PublishRegistrationFormRequest{EventId: 1, ExpectedRevision: 2})); err != nil {
		t.Fatal(err)
	}
	var old string
	if err := s.db.NewRaw(`SELECT definition->'fields'->0->>'label' FROM event_registration_form_versions WHERE id=?::uuid`, original).Scan(ctx, &old); err != nil {
		t.Fatal(err)
	}
	if old != "Reason" {
		t.Fatal("published snapshot mutated")
	}
	if err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := s.Prepare(ctx, tx, 1, &pb.FormSubmission{FormVersionId: original})
		return err
	}); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale version accepted: %v", err)
	}
}

func TestPendingReviewCapacityHistoryAndSeenMarkers(t *testing.T) {
	s, ctx := formsFixture(t)
	f := &pb.RegistrationForm{EventId: 1, ApprovalMode: pb.ApprovalMode_APPROVAL_MODE_MANUAL, Fields: []*pb.FormField{{Id: "reason", Label: "Reason", Type: pb.FormFieldType_FORM_FIELD_TYPE_SHORT_TEXT, Required: true}}}
	if _, err := s.SaveFormDraft(ctx, connect.NewRequest(&pb.SaveFormDraftRequest{Form: f})); err != nil {
		t.Fatal(err)
	}
	published, err := s.PublishRegistrationForm(ctx, connect.NewRequest(&pb.PublishRegistrationFormRequest{EventId: 1, ExpectedRevision: 1}))
	if err != nil {
		t.Fatal(err)
	}
	var responseID int64
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		p, err := s.Prepare(ctx, tx, 1, &pb.FormSubmission{FormVersionId: published.Msg.Form.VersionId, Answers: []*pb.RegistrationAnswer{{FieldId: "reason", Value: &pb.RegistrationAnswer_Text{Text: "=1+1"}}}})
		if err != nil {
			return err
		}
		if p.Status != "pending_approval" {
			t.Fatal("manual registration confirmed")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO event_registrations(id,event_id,user_id,status) VALUES(1,1,1,?)`, p.Status); err != nil {
			return err
		}
		responseID, err = s.Record(ctx, tx, p, registrationfiles.Owner{UserID: 1}, pb.RegistrationSource_REGISTRATION_SOURCE_MAP, 1, &pb.ResponseContact{FullName: "=HYPERLINK(1)", Email: "participant@example.test"})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := &pb.RegistrationRef{Source: pb.RegistrationSource_REGISTRATION_SOURCE_MAP, Id: 1}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO external_registrations(id,event_id,full_name,email,phone,status) VALUES(9,1,'Capacity occupant','occupant@example.test','99999','registered')`); err != nil {
		t.Fatal(err)
	}
	review := func(action pb.RegistrationReviewAction) (*connect.Response[pb.ReviewEventRegistrationResponse], error) {
		return s.ReviewEventRegistration(ctx, connect.NewRequest(&pb.ReviewEventRegistrationRequest{EventId: 1, Registration: ref, ResponseId: strconv.FormatInt(responseID, 10), Action: action}))
	}
	if _, err := review(pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_APPROVE); connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("full event approved pending: %v", err)
	}
	detail, err := s.GetEventResponse(ctx, connect.NewRequest(&pb.GetEventResponseRequest{EventId: 1, Registration: ref}))
	if err != nil {
		t.Fatal(err)
	}
	if detail.Msg.Response.Status != "pending_approval" {
		t.Fatal("capacity failure changed status")
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM external_registrations WHERE id=9`); err != nil {
		t.Fatal(err)
	}
	if _, err := review(pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_REJECT); err != nil {
		t.Fatal(err)
	}
	if _, err := review(pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_REOPEN); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE events SET capacity=1 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err := review(pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_APPROVE); err != nil {
		t.Fatal(err)
	}
	listing, err := s.ListEventResponses(ctx, connect.NewRequest(&pb.ListEventResponsesRequest{EventId: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if listing.Msg.Total != 1 || listing.Msg.UnreadCount != 1 {
		t.Fatalf("incorrect list %v", listing.Msg)
	}
	if _, err := s.MarkResponsesSeen(ctx, connect.NewRequest(&pb.MarkResponsesSeenRequest{EventId: 1, LatestResponseId: "9223372036854775807"})); err != nil {
		t.Fatal(err)
	}
	var marker int64
	if err := s.db.NewRaw(`SELECT latest_response_id FROM event_registration_response_reads WHERE event_id=1 AND user_id=1`).Scan(ctx, &marker); err != nil {
		t.Fatal(err)
	}
	if marker != responseID {
		t.Fatal("seen cutoff escaped current event maximum")
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO external_registrations(id,event_id,full_name,email,phone,status) VALUES(1,1,'Legacy','legacy@example.test','12345','registered')`); err != nil {
		t.Fatal(err)
	}
	listing, err = s.ListEventResponses(ctx, connect.NewRequest(&pb.ListEventResponsesRequest{EventId: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if listing.Msg.Total != 2 {
		t.Fatal("legacy external entry omitted")
	}
	exported, err := s.ExportEventResponses(ctx, connect.NewRequest(&pb.ExportEventResponsesRequest{EventId: 1}))
	if err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(exported.Msg.Csv)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if records[1][2] != "'=HYPERLINK(1)" || records[1][9] != "'=1+1" {
		t.Fatalf("unsafe CSV: %v", records)
	}
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := s.Cancel(ctx, tx, ref.Source, ref.Id); err != nil {
			return err
		}
		p := &Prepared{EventID: 1, Status: "pending_approval", Form: published.Msg.Form}
		_, err := s.Record(ctx, tx, p, registrationfiles.Owner{UserID: 1}, ref.Source, ref.Id, &pb.ResponseContact{FullName: "New", Email: "participant@example.test"})
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE event_registrations SET status='pending_approval' WHERE id=1`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := review(pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_APPROVE); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale response reviewed: %v", err)
	}
	detail, err = s.GetEventResponse(ctx, connect.NewRequest(&pb.GetEventResponseRequest{EventId: 1, Registration: ref}))
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Msg.Response.History) != 1 || detail.Msg.Response.History[0].Status != "cancelled" {
		t.Fatal("prior response attempt was lost")
	}
}
