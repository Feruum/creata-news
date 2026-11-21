// Package registrationforms owns published registration requirements and immutable
// response attempts shared by campus and Sunday Open registrations.
package registrationforms

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"github.com/studyverse/ems-backend/gen/registrationforms/v1/registrationformsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/registrationfiles"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Service struct {
	registrationformsv1connect.UnimplementedRegistrationFormsServiceHandler
	db        *bun.DB
	authz     *authz.Enforcer
	files     *registrationfiles.Service
	publisher *events.Publisher
}

func NewService(db *bun.DB, authz *authz.Enforcer, files *registrationfiles.Service) *Service {
	return &Service{db: db, authz: authz, files: files}
}
func (s *Service) WithPublisher(p *events.Publisher) *Service { s.publisher = p; return s }
func (s *Service) filesAvailable() bool                       { return s.files != nil && s.files.Available() }
func internal(err error) error {
	if err == nil {
		return nil
	}
	var ce *connect.Error
	if errors.As(err, &ce) {
		return err
	}
	return connect.NewError(connect.CodeInternal, err)
}
func (s *Service) require(ctx context.Context, eventID int32, permission string) (int32, error) {
	if eventID <= 0 {
		return 0, invalid("event_id must be positive")
	}
	return s.authz.RequireResourcePermission(ctx, "event", strconv.Itoa(int(eventID)), permission)
}

func (s *Service) published(ctx context.Context, db bun.IDB, eventID int32) (*pb.RegistrationForm, error) {
	var row struct{ Definition json.RawMessage }
	err := db.NewRaw(`SELECT v.definition FROM event_registration_forms f JOIN event_registration_form_versions v ON v.id=f.published_version_id WHERE f.event_id=?`, eventID).Scan(ctx, &row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, internal(err)
	}
	form := new(pb.RegistrationForm)
	if err := protojson.Unmarshal(row.Definition, form); err != nil {
		return nil, internal(err)
	}
	form.FilesAvailable = s.filesAvailable()
	return form, nil
}
func (s *Service) HasPublishedForm(ctx context.Context, db bun.IDB, eventID int32) (bool, error) {
	f, err := s.published(ctx, db, eventID)
	return f != nil, err
}

func (s *Service) GetPublishedForm(ctx context.Context, req *connect.Request[pb.GetPublishedFormRequest]) (*connect.Response[pb.GetPublishedFormResponse], error) {
	if req.Msg.EventId <= 0 {
		return nil, invalid("event_id must be positive")
	}
	var row struct {
		Status                            string
		PublishOnMap, PublishOnSundayOpen bool
	}
	err := s.db.NewRaw(`SELECT status,publish_on_map,publish_on_sunday_open FROM events WHERE id=?`, req.Msg.EventId).Scan(ctx, &row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("event not found"))
	}
	if err != nil {
		return nil, internal(err)
	}
	visible := false
	switch req.Msg.Audience {
	case pb.FormAudience_FORM_AUDIENCE_MAP:
		visible = row.PublishOnMap
	case pb.FormAudience_FORM_AUDIENCE_SUNDAY_OPEN:
		visible = row.PublishOnSundayOpen
	default:
		return nil, invalid("form audience is required")
	}
	if row.Status != "published" || !visible {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("event not found"))
	}
	f, err := s.published(ctx, s.db, req.Msg.EventId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.GetPublishedFormResponse{Form: f}), nil
}

func (s *Service) GetFormDraft(ctx context.Context, req *connect.Request[pb.GetFormDraftRequest]) (*connect.Response[pb.GetFormDraftResponse], error) {
	id, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	resource := strconv.Itoa(int(req.Msg.EventId))
	edit := s.authz.CheckResourcePermission(ctx, id, "event", resource, "edit")
	manage := s.authz.CheckResourcePermission(ctx, id, "event", resource, "manage_registrations")
	if !edit && !manage {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("event access required"))
	}
	f, err := s.draft(ctx, s.db, req.Msg.EventId)
	if err != nil {
		return nil, err
	}
	published, err := s.published(ctx, s.db, req.Msg.EventId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.GetFormDraftResponse{Form: f, PublishedForm: published, CanEdit: edit, CanManageRegistrations: manage}), nil
}
func (s *Service) draft(ctx context.Context, db bun.IDB, eventID int32) (*pb.RegistrationForm, error) {
	var row struct {
		Draft    json.RawMessage
		Revision int32
	}
	err := db.NewRaw(`SELECT draft,revision FROM event_registration_forms WHERE event_id=?`, eventID).Scan(ctx, &row)
	f := &pb.RegistrationForm{EventId: eventID, ApprovalMode: pb.ApprovalMode_APPROVAL_MODE_AUTOMATIC}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, internal(err)
	}
	if err == nil {
		if err := protojson.Unmarshal(row.Draft, f); err != nil {
			return nil, internal(err)
		}
		f.Revision = row.Revision
	}
	f.FilesAvailable = s.filesAvailable()
	return f, nil
}
func (s *Service) SaveFormDraft(ctx context.Context, req *connect.Request[pb.SaveFormDraftRequest]) (*connect.Response[pb.SaveFormDraftResponse], error) {
	if req.Msg.Form == nil {
		return nil, invalid("form is required")
	}
	id, err := s.require(ctx, req.Msg.Form.EventId, "edit")
	if err != nil {
		return nil, err
	}
	f := proto.Clone(req.Msg.Form).(*pb.RegistrationForm)
	f.VersionId = ""
	f.FilesAvailable = s.filesAvailable()
	if err := ValidateDefinition(f, false); err != nil {
		return nil, err
	}
	if f.ApprovalMode == pb.ApprovalMode_APPROVAL_MODE_UNSPECIFIED {
		f.ApprovalMode = pb.ApprovalMode_APPROVAL_MODE_AUTOMATIC
	}
	f.Revision = req.Msg.ExpectedRevision + 1
	raw, err := protojson.Marshal(f)
	if err != nil {
		return nil, internal(err)
	}
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO event_registration_forms(event_id,draft,revision,updated_by) VALUES(?,'{}',0,?) ON CONFLICT(event_id) DO NOTHING`, f.EventId, id); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE event_registration_forms SET draft=?::jsonb,revision=revision+1,updated_by=?,updated_at=NOW() WHERE event_id=? AND revision=?`, string(raw), id, f.EventId, req.Msg.ExpectedRevision)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return connect.NewError(connect.CodeAborted, errors.New("form draft changed; reload before saving"))
		}
		return nil
	})
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&pb.SaveFormDraftResponse{Form: f}), nil
}
func (s *Service) PublishRegistrationForm(ctx context.Context, req *connect.Request[pb.PublishRegistrationFormRequest]) (*connect.Response[pb.PublishRegistrationFormResponse], error) {
	id, err := s.require(ctx, req.Msg.EventId, "edit")
	if err != nil {
		return nil, err
	}
	var f *pb.RegistrationForm
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var eventID int32
		if err := tx.NewRaw(`SELECT id FROM events WHERE id=? FOR UPDATE`, req.Msg.EventId).Scan(ctx, &eventID); err != nil {
			return err
		}
		var revision int32
		if err := tx.NewRaw(`SELECT revision FROM event_registration_forms WHERE event_id=? FOR UPDATE`, req.Msg.EventId).Scan(ctx, &revision); errors.Is(err, sql.ErrNoRows) {
			return precondition("save a draft before publishing")
		} else if err != nil {
			return err
		}
		if revision != req.Msg.ExpectedRevision {
			return connect.NewError(connect.CodeAborted, errors.New("form draft changed; reload before publishing"))
		}
		var err error
		f, err = s.draft(ctx, tx, req.Msg.EventId)
		if err != nil {
			return err
		}
		return publishFormVersion(ctx, tx, f, id)
	})
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&pb.PublishRegistrationFormResponse{Form: f}), nil
}

func publishFormVersion(ctx context.Context, tx bun.Tx, f *pb.RegistrationForm, userID int32) error {
	if err := ValidateDefinition(f, true); err != nil {
		return err
	}
	f.VersionId = uuid.NewString()
	raw, err := protojson.Marshal(f)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO event_registration_form_versions(id,event_id,definition,created_by) VALUES(?::uuid,?,?::jsonb,?)`, f.VersionId, f.EventId, string(raw), userID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE event_registration_forms SET published_version_id=?::uuid,updated_by=?,updated_at=NOW() WHERE event_id=?`, f.VersionId, userID, f.EventId)
	return err
}

// PrepareHermesPaidForm runs in the new draft's transaction. A pre-existing form
// must never be overwritten by an authoring retry.
func (s *Service) PrepareHermesPaidForm(ctx context.Context, tx bun.Tx, eventID int32) error {
	var creator int32
	if err := tx.NewRaw(`SELECT user_id FROM events WHERE id=? AND status='draft'`, eventID).Scan(ctx, &creator); err != nil {
		return err
	}
	f := &pb.RegistrationForm{EventId: eventID, Revision: 1, FilesAvailable: s.filesAvailable(),
		ApprovalMode: pb.ApprovalMode_APPROVAL_MODE_MANUAL,
		Fields: []*pb.FormField{{Id: "payment-receipt", Type: pb.FormFieldType_FORM_FIELD_TYPE_FILE,
			Label: "Загрузите квитанцию об оплате", Required: true, FileRules: &pb.FormFileRules{
				Extensions: []string{"pdf", "jpg", "jpeg", "png", "webp"}, MaxBytes: 10 << 20, MaxFiles: 1,
			}}}}
	raw, err := protojson.Marshal(f)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO event_registration_forms(event_id,draft,revision,updated_by) VALUES(?,?::jsonb,1,?)`, eventID, string(raw), creator); err != nil {
		return err
	}
	return publishFormVersion(ctx, tx, f, creator)
}

type Prepared struct {
	EventID int32
	Status  string
	Form    *pb.RegistrationForm
	Answers []*pb.RegistrationAnswer
}

// Prepare runs under the caller's event lock and registration transaction.
func (s *Service) Prepare(ctx context.Context, tx bun.Tx, eventID int32, submission *pb.FormSubmission) (*Prepared, error) {
	f, err := s.published(ctx, tx, eventID)
	if err != nil {
		return nil, err
	}
	p := &Prepared{EventID: eventID, Status: "registered", Form: f}
	if f == nil {
		if submission != nil && (submission.FormVersionId != "" || len(submission.Answers) > 0) {
			return nil, precondition("registration form changed; reload this event and try again")
		}
		return p, nil
	}
	if submission == nil || submission.FormVersionId != f.VersionId {
		return nil, precondition("registration form changed or answers are missing; reload this event and complete the current form")
	}
	p.Answers, err = ValidateAnswers(f, submission.Answers)
	if err != nil {
		return nil, err
	}
	if f.ApprovalMode == pb.ApprovalMode_APPROVAL_MODE_MANUAL {
		p.Status = "pending_approval"
	}
	return p, nil
}
func registrationColumn(source pb.RegistrationSource) (string, error) {
	switch source {
	case pb.RegistrationSource_REGISTRATION_SOURCE_MAP:
		return "registration_id", nil
	case pb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN:
		return "external_registration_id", nil
	}
	return "", invalid("registration source is required")
}
func (s *Service) Record(ctx context.Context, tx bun.Tx, p *Prepared, owner registrationfiles.Owner, source pb.RegistrationSource, id int32, contact *pb.ResponseContact) (int64, error) {
	column, err := registrationColumn(source)
	if err != nil {
		return 0, err
	}
	if p == nil || contact == nil {
		return 0, invalid("prepared registration and contact are required")
	}
	if p.Form != nil {
		email, err := mail.ParseAddress(contact.Email)
		if strings.TrimSpace(contact.FullName) == "" || err != nil || email.Address != contact.Email {
			return 0, precondition("complete your name and email in your profile before registering")
		}
	}
	if s.files == nil {
		for _, answer := range p.Answers {
			if len(answer.GetFiles().GetUploadIds()) > 0 {
				return 0, precondition("private file storage is unavailable")
			}
		}
	}
	raw, err := protojson.Marshal(&pb.FormSubmission{Answers: p.Answers})
	if err != nil {
		return 0, err
	}
	var wrapper struct {
		Answers json.RawMessage `json:"answers"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return 0, err
	}
	if len(wrapper.Answers) == 0 {
		wrapper.Answers = json.RawMessage("[]")
	}
	contacts, err := protojson.Marshal(contact)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE event_registration_responses SET is_current=false WHERE `+column+`=? AND is_current=true`, id); err != nil {
		return 0, err
	}
	version := ""
	if p.Form != nil {
		version = p.Form.VersionId
	}
	var responseID int64
	err = tx.NewRaw(`INSERT INTO event_registration_responses(event_id,form_version_id,`+column+`,answers,contacts,status) VALUES(?,NULLIF(?,'')::uuid,?,?::jsonb,?::jsonb,?) RETURNING id`, p.EventID, version, id, string(wrapper.Answers), string(contacts), p.Status).Scan(ctx, &responseID)
	if err != nil {
		return 0, err
	}
	if s.files != nil && p.Form != nil {
		if err := s.files.ValidateAndAttach(ctx, tx, p.EventID, version, owner, p.Answers, responseID); err != nil {
			return 0, err
		}
	}
	return responseID, nil
}
func (s *Service) Cancel(ctx context.Context, tx bun.Tx, source pb.RegistrationSource, id int32) error {
	column, err := registrationColumn(source)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE event_registration_responses SET status='cancelled' WHERE `+column+`=? AND is_current=true`, id)
	return err
}

func (s *Service) BeginUploadForOwner(ctx context.Context, owner registrationfiles.Owner, req *pb.BeginFileUploadRequest) (*pb.BeginFileUploadResponse, error) {
	if !s.filesAvailable() {
		return nil, precondition("private file storage is unavailable")
	}
	return s.files.Begin(ctx, owner, req)
}
func (s *Service) CompleteUploadForOwner(ctx context.Context, owner registrationfiles.Owner, req *pb.CompleteFileUploadRequest) (*pb.CompleteFileUploadResponse, error) {
	if !s.filesAvailable() {
		return nil, precondition("private file storage is unavailable")
	}
	return s.files.Complete(ctx, owner, req)
}
func (s *Service) DownloadForOwner(ctx context.Context, owner registrationfiles.Owner, id string) (*pb.GetFileDownloadResponse, error) {
	if !s.filesAvailable() {
		return nil, precondition("private file storage is unavailable")
	}
	return s.files.Download(ctx, owner, id, func(ctx context.Context, eventID int32) bool {
		if owner.UserID == 0 {
			return false
		}
		return s.authz.CheckResourcePermission(ctx, owner.UserID, "event", fmt.Sprint(eventID), "manage_registrations")
	})
}
func (s *Service) BeginFileUpload(ctx context.Context, req *connect.Request[pb.BeginFileUploadRequest]) (*connect.Response[pb.BeginFileUploadResponse], error) {
	id, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.BeginUploadForOwner(ctx, registrationfiles.Owner{UserID: id}, req.Msg)
	if err != nil {
		return nil, internal(err)
	}
	response := connect.NewResponse(res)
	response.Header().Set("Cache-Control", "private, no-store")
	return response, nil
}
func (s *Service) CompleteFileUpload(ctx context.Context, req *connect.Request[pb.CompleteFileUploadRequest]) (*connect.Response[pb.CompleteFileUploadResponse], error) {
	id, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.CompleteUploadForOwner(ctx, registrationfiles.Owner{UserID: id}, req.Msg)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(res), nil
}
func (s *Service) GetFileDownload(ctx context.Context, req *connect.Request[pb.GetFileDownloadRequest]) (*connect.Response[pb.GetFileDownloadResponse], error) {
	id, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.DownloadForOwner(ctx, registrationfiles.Owner{UserID: id}, req.Msg.UploadId)
	if err != nil {
		return nil, internal(err)
	}
	response := connect.NewResponse(res)
	response.Header().Set("Cache-Control", "private, no-store")
	return response, nil
}
