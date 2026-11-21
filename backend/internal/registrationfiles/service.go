// Package registrationfiles owns private registration attachment admission and binding.
package registrationfiles

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"github.com/studyverse/ems-backend/internal/storage"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/encoding/protojson"
)

type Owner struct {
	UserID    int32
	AccountID string
}

func (o Owner) valid() bool {
	if o.UserID > 0 {
		return o.AccountID == ""
	}
	if o.UserID != 0 || o.AccountID == "" {
		return false
	}
	id, e := uuid.Parse(o.AccountID)
	return e == nil && id != uuid.Nil
}
func (o Owner) key() string {
	if o.UserID > 0 {
		return fmt.Sprintf("registration-files:user:%d", o.UserID)
	}
	return "registration-files:account:" + o.AccountID
}
func (o Owner) matches(u *upload) bool {
	return o.valid() && ((o.UserID > 0 && u.OwnerUserID == o.UserID && u.OwnerAccountID == "") || (o.AccountID != "" && u.OwnerAccountID == o.AccountID && u.OwnerUserID == 0))
}

type Service struct {
	db    *bun.DB
	store storage.PrivateProvider
}

func New(db *bun.DB, store storage.PrivateProvider) *Service { return &Service{db: db, store: store} }
func (s *Service) Available() bool                           { return s != nil && s.db != nil && s.store != nil }
func failure(code connect.Code, message string) error {
	return connect.NewError(code, errors.New(message))
}
func (s *Service) check(owner Owner) error {
	if !owner.valid() {
		return failure(connect.CodeUnauthenticated, "A valid authenticated account is required")
	}
	if !s.Available() {
		return failure(connect.CodeUnavailable, "Registration file storage is unavailable")
	}
	return nil
}

type upload struct {
	bun.BaseModel  `bun:"table:event_registration_uploads,alias:u"`
	ID             string `bun:"id,pk"`
	EventID        int32
	FormVersionID  string
	FieldID        string
	OwnerUserID    int32  `bun:",nullzero"`
	OwnerAccountID string `bun:",nullzero"`
	FileName       string
	ContentType    string
	SizeBytes      int64
	StagingKey     string
	ObjectKey      string
	State          string
	ResponseID     int64 `bun:",nullzero"`
	CreatedAt      time.Time
	ExpiresAt      time.Time
}

func requireOwner(ctx context.Context, db bun.IDB, owner Owner) error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id=?)"
	var id interface{} = owner.UserID
	if owner.AccountID != "" {
		query = "SELECT EXISTS(SELECT 1 FROM sunday_open_accounts WHERE id=?::uuid)"
		id = owner.AccountID
	}
	if err := db.NewRaw(query, id).Scan(ctx, &exists); err != nil {
		return fmt.Errorf("verify upload owner: %w", err)
	}
	if !exists {
		return failure(connect.CodeUnauthenticated, "Account not found")
	}
	return nil
}

func published(ctx context.Context, db bun.IDB, eventID int32, version string, owner Owner) (*pb.RegistrationForm, error) {
	if eventID <= 0 {
		return nil, failure(connect.CodeInvalidArgument, "Invalid event")
	}
	if _, err := uuid.Parse(version); err != nil {
		return nil, failure(connect.CodeInvalidArgument, "Invalid form version")
	}
	audience := "e.publish_on_map"
	if owner.AccountID != "" {
		audience = "e.publish_on_sunday_open"
	}
	var definition string
	err := db.NewRaw(`SELECT v.definition FROM event_registration_form_versions v JOIN event_registration_forms f ON f.event_id=v.event_id AND f.published_version_id=v.id JOIN events e ON e.id=v.event_id WHERE e.id=? AND v.id=?::uuid AND e.status='published' AND e.completed_at IS NULL AND e.end_time>now() AND `+audience+`=true FOR SHARE OF e,f,v`, eventID, version).Scan(ctx, &definition)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, failure(connect.CodeFailedPrecondition, "Registration form is unavailable or no longer current")
	}
	if err != nil {
		return nil, fmt.Errorf("load current file form: %w", err)
	}
	form := new(pb.RegistrationForm)
	if err = protojson.Unmarshal([]byte(definition), form); err != nil {
		return nil, fmt.Errorf("decode registration form: %w", err)
	}
	return form, nil
}
func fieldByID(form *pb.RegistrationForm, id string) *pb.FormField {
	for _, field := range form.Fields {
		if field.Id == id {
			return field
		}
	}
	return nil
}

func (s *Service) Begin(ctx context.Context, owner Owner, req *pb.BeginFileUploadRequest) (*pb.BeginFileUploadResponse, error) {
	if err := s.check(owner); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, failure(connect.CodeInvalidArgument, "Missing upload request")
	}
	var response *pb.BeginFileUploadResponse
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended(?,0))", owner.key()); err != nil {
			return err
		}
		if err := requireOwner(ctx, tx, owner); err != nil {
			return err
		}
		form, err := published(ctx, tx, req.EventId, req.FormVersionId, owner)
		if err != nil {
			return err
		}
		field := fieldByID(form, req.FieldId)
		name := safeFilename(req.FileName)
		if err = validateMetadata(field, name, req.ContentType, req.SizeBytes); err != nil {
			return failure(connect.CodeInvalidArgument, err.Error())
		}
		budget, err := admissionBudget(form)
		if err != nil {
			return failure(connect.CodeFailedPrecondition, "Published file rules are invalid")
		}
		var outstanding, recent int
		if err = tx.NewRaw(`SELECT count(*) FILTER(WHERE state IN ('uploading','ready') AND expires_at>now()), count(*) FILTER(WHERE created_at>now()-interval '10 minutes') FROM event_registration_uploads WHERE owner_user_id IS NOT DISTINCT FROM ?::integer AND owner_account_id IS NOT DISTINCT FROM ?::uuid`, nullableUser(owner), nullableAccount(owner)).Scan(ctx, &outstanding, &recent); err != nil {
			return err
		}
		if outstanding >= budget || recent >= budget {
			return failure(connect.CodeResourceExhausted, "Too many outstanding or recent uploads")
		}
		// maxFiles limits the submitted selection. Abandoned/replaced uploads
		// have a separate quota so a failed PUT cannot consume the only slot.
		var eventOutstanding int
		if err = tx.NewRaw(`SELECT count(*) FROM event_registration_uploads WHERE event_id=? AND owner_user_id IS NOT DISTINCT FROM ?::integer AND owner_account_id IS NOT DISTINCT FROM ?::uuid AND state IN ('uploading','ready') AND expires_at>now()`, req.EventId, nullableUser(owner), nullableAccount(owner)).Scan(ctx, &eventOutstanding); err != nil {
			return err
		}
		if eventOutstanding >= budget {
			return failure(connect.CodeResourceExhausted, "Too many outstanding event uploads")
		}
		now := time.Now().UTC()
		id := uuid.NewString()
		key := "staging/" + id
		signed, err := s.store.SignUpload(ctx, key, req.ContentType, 5*time.Minute)
		if err != nil {
			return fmt.Errorf("sign private upload: %w", err)
		}
		u := &upload{ID: id, EventID: req.EventId, FormVersionID: req.FormVersionId, FieldID: req.FieldId, OwnerUserID: owner.UserID, OwnerAccountID: owner.AccountID, FileName: name, ContentType: req.ContentType, SizeBytes: req.SizeBytes, StagingKey: key, ObjectKey: "final/" + id + "/" + uuid.NewString(), State: "uploading", CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
		if _, err = tx.NewInsert().Model(u).Exec(ctx); err != nil {
			return fmt.Errorf("reserve private upload: %w", err)
		}
		response = &pb.BeginFileUploadResponse{UploadId: id, UploadUrl: signed.UploadURL, Headers: signed.Headers, ExpiresAt: now.Add(5 * time.Minute).Format(time.RFC3339)}
		return nil
	})
	return response, err
}

func admissionBudget(form *pb.RegistrationForm) (int, error) {
	if len(form.Fields) > 50 {
		return 0, fmt.Errorf("form exceeds question limit")
	}
	capacity := 0
	for _, field := range form.Fields {
		if field != nil && field.Type == pb.FormFieldType_FORM_FIELD_TYPE_FILE {
			_, _, count, err := limits(field.FileRules)
			if err != nil {
				return 0, err
			}
			capacity += count
		}
	}
	return min(300, max(30, 2*capacity)), nil
}
func nullableUser(owner Owner) interface{} {
	if owner.UserID == 0 {
		return nil
	}
	return owner.UserID
}
func nullableAccount(owner Owner) interface{} {
	if owner.AccountID == "" {
		return nil
	}
	return owner.AccountID
}
func getUpload(ctx context.Context, db bun.IDB, id string, lock bool) (*upload, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, failure(connect.CodeInvalidArgument, "Invalid upload ID")
	}
	u := new(upload)
	q := db.NewSelect().Model(u).Where("u.id=?::uuid", id)
	if lock {
		q = q.For("UPDATE")
	}
	err := q.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, failure(connect.CodeNotFound, "File not found")
	}
	return u, err
}
func attachment(u *upload) *pb.RegistrationAttachment {
	return &pb.RegistrationAttachment{Id: u.ID, FieldId: u.FieldID, FileName: u.FileName, ContentType: u.ContentType, SizeBytes: u.SizeBytes}
}

func (s *Service) Complete(ctx context.Context, owner Owner, req *pb.CompleteFileUploadRequest) (*pb.CompleteFileUploadResponse, error) {
	if err := s.check(owner); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, failure(connect.CodeInvalidArgument, "Missing upload request")
	}
	initial, err := getUpload(ctx, s.db, req.UploadId, false)
	if err != nil {
		return nil, err
	}
	if !owner.matches(initial) {
		return nil, failure(connect.CodePermissionDenied, "File access denied")
	}
	var response *pb.CompleteFileUploadResponse
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		form, err := published(ctx, tx, initial.EventID, initial.FormVersionID, owner)
		if err != nil {
			return err
		}
		u, err := getUpload(ctx, tx, req.UploadId, true)
		if err != nil {
			return err
		}
		if !owner.matches(u) || !u.ExpiresAt.After(time.Now()) || u.State != "uploading" {
			return failure(connect.CodeFailedPrecondition, "File upload is expired or already completed")
		}
		field := fieldByID(form, u.FieldID)
		if err = validateMetadata(field, u.FileName, u.ContentType, u.SizeBytes); err != nil {
			return failure(connect.CodeInvalidArgument, err.Error())
		}
		data, err := s.store.Read(ctx, u.StagingKey, u.SizeBytes)
		if err != nil {
			return failure(connect.CodeInvalidArgument, "Uploaded file cannot be read or exceeds its declared size")
		}
		if int64(len(data)) != u.SizeBytes {
			return failure(connect.CodeInvalidArgument, "Uploaded file size does not match")
		}
		if err = validateBytes(u.FileName, u.ContentType, data); err != nil {
			return failure(connect.CodeInvalidArgument, err.Error())
		}
		// A new unpredictable final key is never signed for PUT. Replaying a valid
		// staging PUT after completion cannot change the accepted attachment.
		// Reserve the final key before any storage write, so a lost storage or
		// database response cannot orphan an object outside expiry cleanup.
		finalKey := u.ObjectKey
		if finalKey == "" {
			return failure(connect.CodeFailedPrecondition, "File upload reservation is invalid")
		}
		if err = s.store.Write(ctx, finalKey, u.ContentType, u.FileName, data); err != nil {
			// A prior exclusive create may have succeeded while its response was
			// lost. Only accept that immutable object if it contains these same
			// validated bytes. The staging object may have changed since then.
			existing, readErr := s.store.Read(ctx, finalKey, u.SizeBytes)
			if readErr != nil || !bytes.Equal(existing, data) {
				return fmt.Errorf("write validated private attachment: %w", err)
			}
		}
		result, err := tx.NewUpdate().Model((*upload)(nil)).Set("object_key=?", finalKey).Set("state='ready'").Where("id=?::uuid AND state='uploading'", u.ID).Exec(ctx)
		if err != nil {
			return err
		}
		if n, _ := result.RowsAffected(); n != 1 {
			return failure(connect.CodeAborted, "File completion changed concurrently")
		}
		u.ObjectKey = finalKey
		u.State = "ready"
		response = &pb.CompleteFileUploadResponse{Attachment: attachment(u)}
		return nil
	})
	// Keep the staging reservation until its 24h expiry. A signed staging PUT can
	// recreate it for five minutes, so cleanup must delete it again after expiry.
	return response, err
}

func (s *Service) Download(ctx context.Context, owner Owner, id string, canManage func(context.Context, int32) bool) (*pb.GetFileDownloadResponse, error) {
	if err := s.check(owner); err != nil {
		return nil, err
	}
	u, err := getUpload(ctx, s.db, id, false)
	if err != nil {
		return nil, err
	}
	if !owner.matches(u) && (owner.UserID <= 0 || canManage == nil || !canManage(ctx, u.EventID)) {
		return nil, failure(connect.CodePermissionDenied, "File access denied")
	}
	if u.ObjectKey == "" || (u.State != "ready" && u.State != "attached") || (u.State != "attached" && !u.ExpiresAt.After(time.Now())) {
		return nil, failure(connect.CodeFailedPrecondition, "File is not ready or has expired")
	}
	signed, err := s.store.SignDownload(ctx, u.ObjectKey, safeFilename(u.FileName), time.Minute)
	if err != nil {
		return nil, fmt.Errorf("sign private download: %w", err)
	}
	return &pb.GetFileDownloadResponse{Url: signed, ExpiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339)}, nil
}

func validateAttachment(u *upload, eventID int32, versionID string, owner Owner, field *pb.FormField, now time.Time) error {
	if !owner.matches(u) || u.EventID != eventID || u.FormVersionID != versionID || field == nil || u.FieldID != field.Id || u.State != "ready" || u.ResponseID != 0 || u.ObjectKey == "" || !u.ExpiresAt.After(now) {
		return failure(connect.CodeInvalidArgument, "File is foreign, expired, not ready, or already attached")
	}
	if err := validateMetadata(field, u.FileName, u.ContentType, u.SizeBytes); err != nil {
		return failure(connect.CodeInvalidArgument, err.Error())
	}
	return nil
}

func (s *Service) ValidateAndAttach(ctx context.Context, tx bun.Tx, eventID int32, versionID string, owner Owner, answers []*pb.RegistrationAnswer, responseID int64) error {
	hasFiles := false
	for _, answer := range answers {
		if answer != nil && answer.GetFiles() != nil {
			hasFiles = true
		}
	}
	if !hasFiles {
		return nil
	}
	if err := s.check(owner); err != nil {
		return err
	}
	if responseID <= 0 {
		return failure(connect.CodeInvalidArgument, "Invalid response")
	}
	form, err := published(ctx, tx, eventID, versionID, owner)
	if err != nil {
		return err
	}
	var responseMatches bool
	err = tx.NewRaw(`SELECT EXISTS(SELECT 1 FROM event_registration_responses r
 LEFT JOIN event_registrations ir ON ir.id=r.registration_id
 LEFT JOIN external_registrations xr ON xr.id=r.external_registration_id
 WHERE r.id=? AND r.event_id=? AND r.form_version_id=?::uuid AND r.is_current=true
 AND ((r.registration_id IS NOT NULL AND r.external_registration_id IS NULL AND ir.event_id=r.event_id AND ir.user_id=?::integer)
 OR (r.external_registration_id IS NOT NULL AND r.registration_id IS NULL AND xr.event_id=r.event_id AND xr.sunday_open_account_id=?::uuid)))`, responseID, eventID, versionID, nullableUser(owner), nullableAccount(owner)).Scan(ctx, &responseMatches)
	if err != nil {
		return fmt.Errorf("verify attachment response identity: %w", err)
	}
	if !responseMatches {
		return failure(connect.CodeInvalidArgument, "Response does not match file owner, event, or form version")
	}
	seen := map[string]bool{}
	fields := map[string]bool{}
	for _, answer := range answers {
		if answer == nil || answer.GetFiles() == nil {
			continue
		}
		field := fieldByID(form, answer.FieldId)
		if field == nil || field.Type != pb.FormFieldType_FORM_FIELD_TYPE_FILE || fields[field.Id] {
			return failure(connect.CodeInvalidArgument, "Invalid or duplicate file question")
		}
		fields[field.Id] = true
		_, _, max, err := limits(field.FileRules)
		if err != nil {
			return err
		}
		ids := answer.GetFiles().UploadIds
		if len(ids) > max {
			return failure(connect.CodeInvalidArgument, "Too many files for this question")
		}
		for _, id := range ids {
			if seen[id] {
				return failure(connect.CodeInvalidArgument, "A file cannot be attached more than once")
			}
			seen[id] = true
			u, err := getUpload(ctx, tx, id, true)
			if err != nil {
				return err
			}
			if err = validateAttachment(u, eventID, versionID, owner, field, time.Now()); err != nil {
				return err
			}
			result, err := tx.NewUpdate().Model((*upload)(nil)).Set("state='attached'").Set("response_id=?", responseID).Where("id=?::uuid AND state='ready' AND response_id IS NULL", id).Exec(ctx)
			if err != nil {
				return err
			}
			if n, _ := result.RowsAffected(); n != 1 {
				return failure(connect.CodeAborted, "File was attached concurrently")
			}
		}
	}
	return nil
}

// Start runs bounded cleanup until shutdown. Use a lifecycle context owned by main.
func (s *Service) Start(ctx context.Context) {
	if !s.Available() {
		return
	}
	go func() {
		s.cleanup(ctx)
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanup(ctx)
			}
		}
	}()
}
func (s *Service) cleanup(ctx context.Context) {
	// A row remains until every object delete succeeds, making transient storage
	// failures retryable. Row locks prevent cleanup racing a registration commit.
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		uploads := []upload{}
		if err := tx.NewSelect().Model(&uploads).Where("u.expires_at<now() AND u.state<>'attached'").OrderExpr("u.expires_at").Limit(100).For("UPDATE SKIP LOCKED").Scan(ctx); err != nil {
			return err
		}
		for _, u := range uploads {
			if err := s.store.Delete(ctx, u.StagingKey); err != nil {
				continue
			}
			if u.ObjectKey != "" {
				if err := s.store.Delete(ctx, u.ObjectKey); err != nil {
					continue
				}
			}
			if _, err := tx.NewDelete().Model((*upload)(nil)).Where("id=?::uuid AND state<>'attached'", u.ID).Exec(ctx); err != nil {
				return err
			}
		}
		// Attached objects remain private; their obsolete staging object is harmless
		// but must also be removed once its signed write window has elapsed.
		attached := []upload{}
		if err := tx.NewSelect().Model(&attached).Where("u.state='attached' AND u.expires_at<now() AND u.staging_key<>''").Limit(100).For("UPDATE SKIP LOCKED").Scan(ctx); err != nil {
			return err
		}
		for _, u := range attached {
			if err := s.store.Delete(ctx, u.StagingKey); err != nil {
				continue
			}
			if _, err := tx.NewUpdate().Model((*upload)(nil)).Set("staging_key='' ").Where("id=?::uuid", u.ID).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil && ctx.Err() == nil {
		slog.Warn("Registration upload cleanup failed", "error", err)
	}
}
