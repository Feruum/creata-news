package registrationfiles

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"github.com/studyverse/ems-backend/internal/storage"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"google.golang.org/protobuf/encoding/protojson"
)

type memoryStore struct {
	objects       map[string][]byte
	signedKeys    []string
	failWriteOnce bool
	failReadOnce  bool
}

func (m *memoryStore) SignUpload(_ context.Context, key, mime string, d time.Duration) (*storage.UploadURLResult, error) {
	m.signedKeys = append(m.signedKeys, key)
	return &storage.UploadURLResult{UploadURL: "https://private.test/" + key, Headers: map[string]string{"Content-Type": mime}}, nil
}
func (m *memoryStore) Read(_ context.Context, key string, max int64) ([]byte, error) {
	if m.failReadOnce {
		m.failReadOnce = false
		return nil, errors.New("storage temporarily unavailable")
	}
	b, ok := m.objects[key]
	if !ok || int64(len(b)) > max {
		return nil, errors.New("missing or oversized")
	}
	return bytes.Clone(b), nil
}
func (m *memoryStore) Write(_ context.Context, key, mime, name string, data []byte) error {
	if m.objects[key] != nil {
		return errors.New("immutable key exists")
	}
	m.objects[key] = bytes.Clone(data)
	if m.failWriteOnce {
		m.failWriteOnce = false
		m.failReadOnce = true
		return errors.New("write succeeded but response was lost")
	}
	return nil
}
func (m *memoryStore) Delete(_ context.Context, key string) error { delete(m.objects, key); return nil }
func (m *memoryStore) SignDownload(_ context.Context, key, name string, d time.Duration) (string, error) {
	if d != time.Minute {
		return "", errors.New("download expiry")
	}
	return "https://private.test/" + key, nil
}

func testDatabase(t *testing.T) *bun.DB {
	t.Helper()
	dsn := os.Getenv("REGISTRATION_FILES_TEST_DSN")
	if dsn == "" {
		t.Skip("set REGISTRATION_FILES_TEST_DSN to run isolated Postgres integration checks")
	}
	admin := bun.NewDB(sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn))), pgdialect.New())
	schema := "registration_files_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	db := bun.NewDB(sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn), pgdriver.WithConnParams(map[string]any{"search_path": schema}))), pgdialect.New())
	t.Cleanup(func() {
		_ = db.Close()
		_, _ = admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		_ = admin.Close()
	})
	ddl := `CREATE TABLE users(id integer PRIMARY KEY); INSERT INTO users VALUES(7),(8);
 CREATE TABLE sunday_open_accounts(id uuid PRIMARY KEY);
 CREATE TABLE events(id integer PRIMARY KEY,status text DEFAULT 'published',publish_on_map boolean DEFAULT true,publish_on_sunday_open boolean DEFAULT true,end_time timestamptz DEFAULT now()+interval '1 day',completed_at timestamptz);
 INSERT INTO events(id) VALUES(2),(3);
 CREATE TABLE event_registration_form_versions(id uuid PRIMARY KEY,event_id integer,definition jsonb);
 CREATE TABLE event_registration_forms(event_id integer PRIMARY KEY,published_version_id uuid);
 CREATE TABLE event_registrations(id integer PRIMARY KEY,event_id integer,user_id integer);
 CREATE TABLE external_registrations(id integer PRIMARY KEY,event_id integer,sunday_open_account_id uuid);
 CREATE TABLE event_registration_responses(id bigint PRIMARY KEY,event_id integer,form_version_id uuid,registration_id integer,external_registration_id integer,is_current boolean DEFAULT true);
 CREATE TABLE event_registration_uploads(id uuid PRIMARY KEY,event_id integer,form_version_id uuid,field_id text,owner_user_id integer,owner_account_id uuid,file_name text,content_type text,size_bytes bigint,staging_key text,object_key text,state text,response_id bigint,created_at timestamptz,expires_at timestamptz);`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPrivateUploadLifecycleIntegration(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	store := &memoryStore{objects: map[string][]byte{}}
	s := New(db, store)
	version := uuid.NewString()
	owner := Owner{UserID: 7}
	definition := `{"fields":[{"id":"file","type":"FORM_FIELD_TYPE_FILE","fileRules":{"extensions":["pdf"],"maxFiles":3}}]}`
	if _, err := db.ExecContext(ctx, `INSERT INTO event_registration_form_versions VALUES(?::uuid,2,?::jsonb); INSERT INTO event_registration_forms VALUES(2,?::uuid); INSERT INTO event_registrations VALUES(1,2,7),(2,2,8); INSERT INTO event_registration_responses(id,event_id,form_version_id,registration_id) VALUES(1,2,?::uuid,1),(2,2,?::uuid,2)`, version, definition, version, version, version); err != nil {
		t.Fatal(err)
	}
	pdf := []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%EOF")
	begin := func() *pb.BeginFileUploadResponse {
		t.Helper()
		r, e := s.Begin(ctx, owner, &pb.BeginFileUploadRequest{EventId: 2, FormVersionId: version, FieldId: "file", FileName: "../../evidence.pdf", ContentType: "application/pdf", SizeBytes: int64(len(pdf))})
		if e != nil {
			t.Fatal(e)
		}
		store.objects["staging/"+r.UploadId] = bytes.Clone(pdf)
		return r
	}
	first := begin()
	if _, err := s.Complete(ctx, Owner{UserID: 8}, &pb.CompleteFileUploadRequest{UploadId: first.UploadId}); err == nil {
		t.Fatal("foreign completion accepted")
	}
	store.failWriteOnce = true
	if _, err := s.Complete(ctx, owner, &pb.CompleteFileUploadRequest{UploadId: first.UploadId}); err == nil {
		t.Fatal("expected injected write failure")
	}
	reserved, err := getUpload(ctx, db, first.UploadId, false)
	if err != nil {
		t.Fatal(err)
	}
	if reserved.ObjectKey == "" || store.objects[reserved.ObjectKey] == nil {
		t.Fatal("failed completion orphaned its final object outside the cleanup reservation")
	}
	completed, err := s.Complete(ctx, owner, &pb.CompleteFileUploadRequest{UploadId: first.UploadId})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Attachment.FileName != "evidence.pdf" {
		t.Fatal(completed.Attachment.FileName)
	}
	u, err := getUpload(ctx, db, first.UploadId, false)
	if err != nil {
		t.Fatal(err)
	}
	if u.ObjectKey == u.StagingKey || !strings.HasPrefix(u.ObjectKey, "final/") {
		t.Fatal("final key must differ from staging")
	}
	store.objects[u.StagingKey] = []byte("replayed signed PUT")
	if !bytes.Equal(store.objects[u.ObjectKey], pdf) {
		t.Fatal("staging replay changed accepted bytes")
	}
	if _, err = s.Download(ctx, Owner{UserID: 8}, u.ID, nil); err == nil {
		t.Fatal("foreign download accepted")
	}
	if _, err = s.Download(ctx, Owner{UserID: 8}, u.ID, func(context.Context, int32) bool { return true }); err != nil {
		t.Fatal(err)
	}
	answers := []*pb.RegistrationAnswer{{FieldId: "file", Value: &pb.RegistrationAnswer_Files{Files: &pb.FileSelection{UploadIds: []string{u.ID}}}}}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return s.ValidateAndAttach(ctx, tx, 2, version, owner, answers, 2)
	})
	if err == nil {
		t.Fatal("attached file to another participant's response")
	}
	rollback := errors.New("rollback registration")
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := s.ValidateAndAttach(ctx, tx, 2, version, owner, answers, 1); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	u, _ = getUpload(ctx, db, u.ID, false)
	if u.State != "ready" || u.ResponseID != 0 {
		t.Fatal("attachment escaped rollback")
	}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return s.ValidateAndAttach(ctx, tx, 2, version, owner, answers, 1)
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return s.ValidateAndAttach(ctx, tx, 2, version, owner, answers, 1)
	})
	if err == nil {
		t.Fatal("already-bound attachment accepted")
	}
	expired := begin()
	if _, err = db.ExecContext(ctx, "UPDATE event_registration_uploads SET expires_at=now()-interval '1 second' WHERE id=?::uuid", expired.UploadId); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Complete(ctx, owner, &pb.CompleteFileUploadRequest{UploadId: expired.UploadId}); err == nil {
		t.Fatal("expired completion accepted")
	}
	s.cleanup(ctx)
	if _, ok := store.objects["staging/"+expired.UploadId]; ok {
		t.Fatal("expired staging retained")
	}
	if store.objects[u.ObjectKey] == nil {
		t.Fatal("attached object deleted")
	}
	for _, mutation := range []string{"publish_on_map=false", "status='cancelled'", "end_time=now()-interval '1 day'", "completed_at=now()"} {
		if _, err = db.ExecContext(ctx, "UPDATE events SET "+mutation+" WHERE id=2"); err != nil {
			t.Fatal(err)
		}
		if _, err = s.Begin(ctx, owner, &pb.BeginFileUploadRequest{EventId: 2, FormVersionId: version, FieldId: "file", FileName: "ok.pdf", ContentType: "application/pdf", SizeBytes: 80}); err == nil {
			t.Fatal("closed or hidden event accepted: " + mutation)
		}
		_, _ = db.ExecContext(ctx, "UPDATE events SET publish_on_map=true,status='published',end_time=now()+interval '1 day',completed_at=NULL WHERE id=2")
	}
	for name, request := range map[string]*pb.BeginFileUploadRequest{"old version": {EventId: 2, FormVersionId: uuid.NewString(), FieldId: "file", FileName: "ok.pdf", ContentType: "application/pdf", SizeBytes: 80}, "unknown field": {EventId: 2, FormVersionId: version, FieldId: "other", FileName: "ok.pdf", ContentType: "application/pdf", SizeBytes: 80}, "unknown event": {EventId: 33, FormVersionId: version, FieldId: "file", FileName: "ok.pdf", ContentType: "application/pdf", SizeBytes: 80}} {
		if _, err := s.Begin(ctx, owner, request); err == nil {
			t.Fatal(fmt.Sprintf("accepted %s", name))
		}
	}
}

func TestBeginAllowsReplacingQuestionFileIntegration(t *testing.T) {
	db := testDatabase(t)
	ctx := context.Background()
	store := &memoryStore{objects: map[string][]byte{}}
	s := New(db, store)
	owner := Owner{UserID: 7}
	version := uuid.NewString()
	definition := `{"fields":[{"id":"file","type":"FORM_FIELD_TYPE_FILE","fileRules":{"extensions":["pdf"],"maxFiles":1}}]}`
	if _, err := db.ExecContext(ctx, `INSERT INTO event_registration_form_versions VALUES(?::uuid,2,?::jsonb); INSERT INTO event_registration_forms VALUES(2,?::uuid); INSERT INTO event_registrations VALUES(1,2,7); INSERT INTO event_registration_responses(id,event_id,form_version_id,registration_id) VALUES(1,2,?::uuid,1)`, version, definition, version, version); err != nil {
		t.Fatal(err)
	}
	pdf := []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%EOF")
	begin := func() *pb.BeginFileUploadResponse {
		t.Helper()
		r, err := s.Begin(ctx, owner, &pb.BeginFileUploadRequest{EventId: 2, FormVersionId: version, FieldId: "file", FileName: "evidence.pdf", ContentType: "application/pdf", SizeBytes: int64(len(pdf))})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	complete := func(r *pb.BeginFileUploadResponse) {
		t.Helper()
		store.objects["staging/"+r.UploadId] = bytes.Clone(pdf)
		if _, err := s.Complete(ctx, owner, &pb.CompleteFileUploadRequest{UploadId: r.UploadId}); err != nil {
			t.Fatal(err)
		}
	}
	original := begin()
	complete(original)
	// Removing a ready selection in the browser must permit a new reservation.
	failed := begin()
	store.objects["staging/"+failed.UploadId] = bytes.Repeat([]byte("x"), len(pdf))
	if _, err := s.Complete(ctx, owner, &pb.CompleteFileUploadRequest{UploadId: failed.UploadId}); err == nil {
		t.Fatal("invalid file accepted")
	}
	replacement := begin()
	complete(replacement)
	answers := []*pb.RegistrationAnswer{{FieldId: "file", Value: &pb.RegistrationAnswer_Files{Files: &pb.FileSelection{UploadIds: []string{original.UploadId, replacement.UploadId}}}}}
	if err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return s.ValidateAndAttach(ctx, tx, 2, version, owner, answers, 1)
	}); err == nil {
		t.Fatal("selected more than question maxFiles")
	}
	answers[0].GetFiles().UploadIds = []string{replacement.UploadId}
	if err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return s.ValidateAndAttach(ctx, tx, 2, version, owner, answers, 1)
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 27; i++ {
		begin()
	}
	if _, err := s.Begin(ctx, owner, &pb.BeginFileUploadRequest{EventId: 2, FormVersionId: version, FieldId: "file", FileName: "evidence.pdf", ContentType: "application/pdf", SizeBytes: int64(len(pdf))}); err == nil {
		t.Fatal("unbounded owner upload reservations")
	}
}

func TestAdmissionBudgetSupportsPublishedFileCapacityIntegration(t *testing.T) {
	for _, count := range []int{13, 33} {
		t.Run(fmt.Sprintf("%d_questions", count), func(t *testing.T) {
			db := testDatabase(t)
			ctx := context.Background()
			store := &memoryStore{objects: map[string][]byte{}}
			s := New(db, store)
			version := uuid.NewString()
			form := &pb.RegistrationForm{}
			for i := 0; i < count; i++ {
				form.Fields = append(form.Fields, &pb.FormField{Id: fmt.Sprintf("file_%d", i), Required: true, Type: pb.FormFieldType_FORM_FIELD_TYPE_FILE, FileRules: &pb.FormFileRules{Extensions: []string{"pdf"}, MaxFiles: 1}})
			}
			definition, err := protojson.Marshal(form)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.ExecContext(ctx, `INSERT INTO event_registration_form_versions VALUES(?::uuid,2,?::jsonb); INSERT INTO event_registration_forms VALUES(2,?::uuid)`, version, string(definition), version); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < count+1; i++ {
				fieldID := fmt.Sprintf("file_%d", i%count)
				if _, err := s.Begin(ctx, Owner{UserID: 7}, &pb.BeginFileUploadRequest{EventId: 2, FormVersionId: version, FieldId: fieldID, FileName: "evidence.pdf", ContentType: "application/pdf", SizeBytes: 80}); err != nil {
					t.Fatalf("published capacity or replacement blocked at %d: %v", i+1, err)
				}
			}
		})
	}
}
