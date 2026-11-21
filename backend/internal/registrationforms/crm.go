package registrationforms

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/registration"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/encoding/protojson"
)

// Both branches are scoped before joining contacts. No global user directory
// permission or lookup is used by the event organizer's CRM.
const registrationsSQL = `SELECT er.id,1 AS source,er.status::text,er.registered_at AS submitted_at,
 COALESCE(NULLIF(trim(concat_ws(' ',u.first_name,u.last_name)),''),u.username,'') AS full_name,COALESCE(u.email,'') AS email,''::text AS phone,COALESCE(u.telegram_username,'') AS telegram_username,
 (SELECT min(checked_in_at) FROM event_attendance WHERE registration_id=er.id) AS checked_in_at,
 r.id AS response_id,r.contacts,r.notes,r.answers,r.form_version_id::text,v.definition
 FROM event_registrations er JOIN users u ON u.id=er.user_id
 LEFT JOIN event_registration_responses r ON r.registration_id=er.id AND r.is_current=true
 LEFT JOIN event_registration_form_versions v ON v.id=r.form_version_id WHERE er.event_id=?
 UNION ALL
 SELECT xr.id,2 AS source,xr.status::text,xr.registered_at,xr.full_name,xr.email,COALESCE(xr.phone,''),COALESCE(sa.telegram_username,''),xr.checked_in_at,
 r.id,r.contacts,r.notes,r.answers,r.form_version_id::text,v.definition
 FROM external_registrations xr LEFT JOIN sunday_open_accounts sa ON sa.id=xr.sunday_open_account_id
 LEFT JOIN event_registration_responses r ON r.external_registration_id=xr.id AND r.is_current=true
 LEFT JOIN event_registration_form_versions v ON v.id=r.form_version_id WHERE xr.event_id=?`

type responseRow struct {
	ID                            int32
	Source                        int32
	Status                        string
	SubmittedAt                   time.Time
	FullName, Email, Phone        string
	TelegramUsername              string
	CheckedInAt                   *time.Time
	ResponseID                    *int64
	Contacts, Answers, Definition json.RawMessage
	Notes                         *string
	FormVersionID                 *string
}

func decodeAnswers(raw json.RawMessage) ([]*pb.RegistrationAnswer, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	submission := new(pb.FormSubmission)
	if err := protojson.Unmarshal(append(append([]byte(`{"answers":`), raw...), '}'), submission); err != nil {
		return nil, err
	}
	return submission.Answers, nil
}
func (row responseRow) proto(readID int64) (*pb.EventResponse, error) {
	result := &pb.EventResponse{Registration: &pb.RegistrationRef{Source: pb.RegistrationSource(row.Source), Id: row.ID}, Contact: &pb.ResponseContact{FullName: row.FullName, Email: row.Email, Phone: row.Phone}, Status: row.Status, SubmittedAt: row.SubmittedAt.Format(time.RFC3339)}
	if row.ResponseID != nil {
		result.ResponseId = strconv.FormatInt(*row.ResponseID, 10)
		result.IsNew = *row.ResponseID > readID
	}
	if row.Notes != nil {
		result.Notes = *row.Notes
	}
	if row.CheckedInAt != nil {
		result.CheckedInAt = row.CheckedInAt.Format(time.RFC3339)
	}
	if len(row.Contacts) > 0 && string(row.Contacts) != "{}" {
		if err := protojson.Unmarshal(row.Contacts, result.Contact); err != nil {
			return nil, err
		}
	}
	// Telegram follows the current linked account, including unlinking, while
	// name/email/phone retain the contact snapshot submitted with the registration.
	result.Contact.TelegramUsername = row.TelegramUsername
	if len(row.Definition) > 0 {
		result.Form = new(pb.RegistrationForm)
		if err := protojson.Unmarshal(row.Definition, result.Form); err != nil {
			return nil, err
		}
	}
	var err error
	result.Answers, err = decodeAnswers(row.Answers)
	return result, err
}
func filters(q *bun.SelectQuery, search, status string, source pb.RegistrationSource) (*bun.SelectQuery, error) {
	if status != "" {
		switch status {
		case "registered", "cancelled", "waitlist", "pending_approval", "rejected":
		default:
			return nil, invalid("unsupported registration status")
		}
		q = q.Where("r.status=?", status)
	}
	if source != pb.RegistrationSource_REGISTRATION_SOURCE_UNSPECIFIED {
		if _, err := registrationColumn(source); err != nil {
			return nil, err
		}
		q = q.Where("r.source=?", int(source))
	}
	if term := strings.TrimSpace(search); term != "" {
		if len(term) > 300 {
			return nil, invalid("search is too long")
		}
		term = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(term)
		term = "%" + term + "%"
		q = q.Where(`(COALESCE(r.contacts->>'fullName',r.full_name) ILIKE ? OR COALESCE(r.contacts->>'email',r.email) ILIKE ? OR COALESCE(r.contacts->>'phone',r.phone) ILIKE ?)`, term, term, term)
	}
	return q, nil
}
func (s *Service) query(db bun.IDB, eventID int32) *bun.SelectQuery {
	return db.NewSelect().TableExpr("(?) AS r", db.NewRaw(registrationsSQL, eventID, eventID)).ColumnExpr("r.*")
}
func (s *Service) readMarker(ctx context.Context, eventID, userID int32) (int64, error) {
	var id int64
	err := s.db.NewRaw(`SELECT latest_response_id FROM event_registration_response_reads WHERE event_id=? AND user_id=?`, eventID, userID).Scan(ctx, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}
func (s *Service) ListEventResponses(ctx context.Context, req *connect.Request[pb.ListEventResponsesRequest]) (*connect.Response[pb.ListEventResponsesResponse], error) {
	id, err := s.require(ctx, req.Msg.EventId, "manage_registrations")
	if err != nil {
		return nil, err
	}
	marker, err := s.readMarker(ctx, req.Msg.EventId, id)
	if err != nil {
		return nil, internal(err)
	}
	q, err := filters(s.query(s.db, req.Msg.EventId), req.Msg.Search, req.Msg.Status, req.Msg.Source)
	if err != nil {
		return nil, err
	}
	page, limit := req.Msg.Page, req.Msg.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var rows []responseRow
	total, err := q.OrderExpr("r.submitted_at DESC,r.source,r.id DESC").Limit(int(limit)).Offset(int((int64(page)-1)*int64(limit))).ScanAndCount(ctx, &rows)
	if err != nil {
		return nil, internal(err)
	}
	result := &pb.ListEventResponsesResponse{Responses: []*pb.EventResponse{}, Total: int32(total)}
	for _, row := range rows {
		r, err := row.proto(marker)
		if err != nil {
			return nil, internal(err)
		}
		result.Responses = append(result.Responses, r)
	}
	var counts struct {
		Latest int64
		Unread int32
	}
	err = s.db.NewRaw(`SELECT COALESCE(MAX(id),0) AS latest,COUNT(*) FILTER(WHERE id>? AND is_current=true) AS unread FROM event_registration_responses WHERE event_id=?`, marker, req.Msg.EventId).Scan(ctx, &counts)
	if err != nil {
		return nil, internal(err)
	}
	result.LatestResponseId = strconv.FormatInt(counts.Latest, 10)
	result.UnreadCount = counts.Unread
	return connect.NewResponse(result), nil
}
func (s *Service) detail(ctx context.Context, db bun.IDB, eventID int32, ref *pb.RegistrationRef, marker int64) (*pb.EventResponse, error) {
	if ref == nil || ref.Id <= 0 {
		return nil, invalid("registration reference is required")
	}
	column, err := registrationColumn(ref.Source)
	if err != nil {
		return nil, err
	}
	var row responseRow
	if err := s.query(db, eventID).Where("r.source=?", int(ref.Source)).Where("r.id=?", ref.Id).Scan(ctx, &row); errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("registration not found"))
	} else if err != nil {
		return nil, err
	}
	result, err := row.proto(marker)
	if err != nil {
		return nil, err
	}
	var attempts []struct {
		ID                  int64
		Definition, Answers json.RawMessage
		Status              string
		SubmittedAt         time.Time
		ReviewedAt          *time.Time
		ReviewedBy          *int32
		IsCurrent           bool
	}
	err = db.NewRaw(`SELECT r.id,v.definition,r.answers,r.status,r.submitted_at,r.reviewed_at,r.reviewed_by,r.is_current FROM event_registration_responses r LEFT JOIN event_registration_form_versions v ON v.id=r.form_version_id WHERE r.event_id=? AND r.`+column+`=? ORDER BY r.id DESC`, eventID, ref.Id).Scan(ctx, &attempts)
	if err != nil {
		return nil, err
	}
	for _, attempt := range attempts {
		a := &pb.ResponseAttempt{Id: strconv.FormatInt(attempt.ID, 10), Status: attempt.Status, SubmittedAt: attempt.SubmittedAt.Format(time.RFC3339)}
		if len(attempt.Definition) > 0 {
			a.Form = new(pb.RegistrationForm)
			if err := protojson.Unmarshal(attempt.Definition, a.Form); err != nil {
				return nil, err
			}
		}
		a.Answers, err = decodeAnswers(attempt.Answers)
		if err != nil {
			return nil, err
		}
		if attempt.ReviewedAt != nil {
			a.ReviewedAt = attempt.ReviewedAt.Format(time.RFC3339)
		}
		if attempt.ReviewedBy != nil {
			a.ReviewedBy = *attempt.ReviewedBy
		}
		var files []struct {
			ID, FieldID, FileName, ContentType string
			SizeBytes                          int64
		}
		if err := db.NewRaw(`SELECT id::text,field_id,file_name,content_type,size_bytes FROM event_registration_uploads WHERE response_id=? AND state='attached' ORDER BY created_at,id`, attempt.ID).Scan(ctx, &files); err != nil {
			return nil, err
		}
		for _, file := range files {
			a.Attachments = append(a.Attachments, &pb.RegistrationAttachment{Id: file.ID, FieldId: file.FieldID, FileName: file.FileName, ContentType: file.ContentType, SizeBytes: file.SizeBytes})
		}
		if attempt.IsCurrent {
			result.Attachments = a.Attachments
		} else {
			result.History = append(result.History, a)
		}
	}
	return result, nil
}
func (s *Service) GetEventResponse(ctx context.Context, req *connect.Request[pb.GetEventResponseRequest]) (*connect.Response[pb.GetEventResponseResponse], error) {
	id, err := s.require(ctx, req.Msg.EventId, "manage_registrations")
	if err != nil {
		return nil, err
	}
	marker, err := s.readMarker(ctx, req.Msg.EventId, id)
	if err != nil {
		return nil, internal(err)
	}
	result, err := s.detail(ctx, s.db, req.Msg.EventId, req.Msg.Registration, marker)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&pb.GetEventResponseResponse{Response: result}), nil
}

func (s *Service) ReviewEventRegistration(ctx context.Context, req *connect.Request[pb.ReviewEventRegistrationRequest]) (*connect.Response[pb.ReviewEventRegistrationResponse], error) {
	actor, err := s.require(ctx, req.Msg.EventId, "manage_registrations")
	if err != nil {
		return nil, err
	}
	if req.Msg.Registration == nil {
		return nil, invalid("registration reference is required")
	}
	column, err := registrationColumn(req.Msg.Registration.Source)
	if err != nil {
		return nil, err
	}
	responseID, err := strconv.ParseInt(req.Msg.ResponseId, 10, 64)
	if err != nil || responseID <= 0 {
		return nil, invalid("current response_id is required")
	}
	var result *pb.EventResponse
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Event lock is acquired before response lock, matching all submission paths.
		var eventOpen bool
		var eventID int32
		if err := tx.NewRaw(`SELECT id FROM events WHERE id=? FOR UPDATE`, req.Msg.EventId).Scan(ctx, &eventID); err != nil {
			return err
		}
		var current struct {
			ID     int64
			Status string
		}
		if err := tx.NewRaw(`SELECT id,status FROM event_registration_responses WHERE event_id=? AND `+column+`=? AND is_current=true FOR UPDATE`, eventID, req.Msg.Registration.Id).Scan(ctx, &current); err != nil {
			return err
		}
		if current.ID != responseID {
			return connect.NewError(connect.CodeAborted, errors.New("a newer response exists; reload before reviewing"))
		}
		next, err := reviewTransition(current.Status, req.Msg.Action)
		if err != nil {
			return err
		}
		if next == "registered" || next == "pending_approval" {
			if err := tx.NewRaw(`SELECT status='published' AND completed_at IS NULL AND end_time>clock_timestamp() FROM events WHERE id=?`, eventID).Scan(ctx, &eventOpen); err != nil {
				return err
			}
			if !eventOpen {
				return precondition("event registration is closed")
			}
		}
		if next == "registered" {
			event, err := registration.LockOpenEvent(ctx, tx, eventID, time.Now)
			if err != nil {
				return err
			}
			if err := registration.EnsureCapacity(ctx, tx, event); err != nil {
				return err
			}
		}
		table := "event_registrations"
		if req.Msg.Registration.Source == pb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN {
			table = "external_registrations"
		}
		saved, err := tx.ExecContext(ctx, `UPDATE `+table+` SET status=?,updated_at=NOW() WHERE id=? AND event_id=? AND status::text=?`, next, req.Msg.Registration.Id, eventID, current.Status)
		if err != nil {
			return err
		}
		n, err := saved.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return precondition("registration changed; reload before reviewing")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE event_registration_responses SET status=?,reviewed_by=?,reviewed_at=NOW() WHERE id=?`, next, actor, responseID); err != nil {
			return err
		}
		if next == "registered" {
			if req.Msg.Registration.Source == pb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN {
				_, err = tx.ExecContext(ctx, `INSERT INTO sunday_open_notification_deliveries(delivery_key,account_id,registration_id,event_id,kind,payload,available_at) SELECT ?,sunday_open_account_id,id,event_id,'registered',snapshot,NOW() FROM external_registrations WHERE id=? AND sunday_open_account_id IS NOT NULL ON CONFLICT(delivery_key) DO NOTHING`, fmt.Sprintf("approved:%d", responseID), req.Msg.Registration.Id)
				if err != nil {
					return err
				}
			} else if publisher := s.publisher.InTransaction(tx); publisher != nil {
				var row struct {
					UserID, OrganizationID int32
					Title                  string
				}
				if err := tx.NewRaw(`SELECT er.user_id,e.organization_id,e.title FROM event_registrations er JOIN events e ON e.id=er.event_id WHERE er.id=?`, req.Msg.Registration.Id).Scan(ctx, &row); err != nil {
					return err
				}
				if err := publisher.PublishRegistrationCreated(ctx, events.RegistrationCreatedPayload{RegistrationID: req.Msg.Registration.Id, EventID: eventID, EventTitle: row.Title, OrganizationID: row.OrganizationID, ParticipantUserID: row.UserID, ActorID: actor}); err != nil {
					return err
				}
			}
		}
		result, err = s.detail(ctx, tx, eventID, req.Msg.Registration, responseID)
		return err
	})
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&pb.ReviewEventRegistrationResponse{Response: result}), nil
}
func (s *Service) SaveResponseNote(ctx context.Context, req *connect.Request[pb.SaveResponseNoteRequest]) (*connect.Response[pb.SaveResponseNoteResponse], error) {
	_, err := s.require(ctx, req.Msg.EventId, "manage_registrations")
	if err != nil {
		return nil, err
	}
	if len(req.Msg.Notes) > 20000 {
		return nil, invalid("note is too long")
	}
	if req.Msg.Registration == nil {
		return nil, invalid("registration reference is required")
	}
	column, err := registrationColumn(req.Msg.Registration.Source)
	if err != nil {
		return nil, err
	}
	var result *pb.EventResponse
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var eventID int32
		if err := tx.NewRaw(`SELECT id FROM events WHERE id=? FOR UPDATE`, req.Msg.EventId).Scan(ctx, &eventID); err != nil {
			return err
		}
		current, err := s.detail(ctx, tx, eventID, req.Msg.Registration, 0)
		if err != nil {
			return err
		}
		if current.ResponseId == "" {
			contact, err := protojson.Marshal(current.Contact)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO event_registration_responses(event_id,`+column+`,contacts,status,submitted_at) VALUES(?,?,?::jsonb,?,?)`, eventID, req.Msg.Registration.Id, string(contact), current.Status, current.SubmittedAt); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE event_registration_responses SET notes=? WHERE event_id=? AND `+column+`=? AND is_current=true`, req.Msg.Notes, eventID, req.Msg.Registration.Id); err != nil {
			return err
		}
		result, err = s.detail(ctx, tx, eventID, req.Msg.Registration, 0)
		return err
	})
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&pb.SaveResponseNoteResponse{Response: result}), nil
}
func (s *Service) MarkResponsesSeen(ctx context.Context, req *connect.Request[pb.MarkResponsesSeenRequest]) (*connect.Response[pb.MarkResponsesSeenResponse], error) {
	user, err := s.require(ctx, req.Msg.EventId, "manage_registrations")
	if err != nil {
		return nil, err
	}
	cutoff, err := strconv.ParseInt(req.Msg.LatestResponseId, 10, 64)
	if err != nil || cutoff < 0 {
		return nil, invalid("latest_response_id must be a nonnegative integer")
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO event_registration_response_reads(event_id,user_id,latest_response_id) VALUES(?,?,LEAST(?,(SELECT COALESCE(MAX(id),0) FROM event_registration_responses WHERE event_id=?))) ON CONFLICT(event_id,user_id) DO UPDATE SET latest_response_id=GREATEST(event_registration_response_reads.latest_response_id,EXCLUDED.latest_response_id)`, req.Msg.EventId, user, cutoff, req.Msg.EventId)
	if err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&pb.MarkResponsesSeenResponse{}), nil
}
func answerText(answer *pb.RegistrationAnswer, field *pb.FormField) string {
	switch v := answer.Value.(type) {
	case *pb.RegistrationAnswer_Text:
		return v.Text
	case *pb.RegistrationAnswer_Number:
		return strconv.FormatFloat(v.Number, 'f', -1, 64)
	case *pb.RegistrationAnswer_Checked:
		return strconv.FormatBool(v.Checked)
	case *pb.RegistrationAnswer_Selection:
		labels := map[string]string{}
		if field != nil {
			for _, o := range field.Options {
				labels[o.Id] = o.Label
			}
		}
		values := []string{}
		for _, id := range v.Selection.GetOptionIds() {
			if label, ok := labels[id]; ok {
				values = append(values, label)
			} else {
				values = append(values, id)
			}
		}
		return strings.Join(values, "; ")
	case *pb.RegistrationAnswer_Files:
		return strings.Join(v.Files.GetUploadIds(), "; ")
	}
	return ""
}
func (s *Service) ExportEventResponses(ctx context.Context, req *connect.Request[pb.ExportEventResponsesRequest]) (*connect.Response[pb.ExportEventResponsesResponse], error) {
	if _, err := s.require(ctx, req.Msg.EventId, "manage_registrations"); err != nil {
		return nil, err
	}
	q, err := filters(s.query(s.db, req.Msg.EventId), req.Msg.Search, req.Msg.Status, req.Msg.Source)
	if err != nil {
		return nil, err
	}
	var rows []responseRow
	if err := q.OrderExpr("r.submitted_at,r.source,r.id").Scan(ctx, &rows); err != nil {
		return nil, internal(err)
	}
	responses := []*pb.EventResponse{}
	ids := []string{}
	labels := map[string]string{}
	// Include historical definitions so renamed and removed questions retain a
	// stable export column rather than silently disappearing from the CSV.
	var versions []struct{ Definition json.RawMessage }
	if err := s.db.NewRaw(`SELECT definition FROM event_registration_form_versions WHERE event_id=? ORDER BY created_at,id`, req.Msg.EventId).Scan(ctx, &versions); err != nil {
		return nil, internal(err)
	}
	for _, v := range versions {
		f := new(pb.RegistrationForm)
		if err := protojson.Unmarshal(v.Definition, f); err != nil {
			return nil, internal(err)
		}
		for _, field := range f.Fields {
			if _, ok := labels[field.Id]; !ok {
				ids = append(ids, field.Id)
			}
			labels[field.Id] = field.Label
		}
	}
	for _, row := range rows {
		r, err := row.proto(0)
		if err != nil {
			return nil, internal(err)
		}
		responses = append(responses, r)
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	header := []string{"Registration ID", "Source", "Name", "Email", "Phone", "Status", "Submitted at", "Checked in at", "Private notes"}
	for _, id := range ids {
		header = append(header, labels[id]+" ["+id+"]")
	}
	for i := range header {
		header[i] = csvSafe(header[i])
	}
	if err := writer.Write(header); err != nil {
		return nil, internal(err)
	}
	for _, r := range responses {
		source := "map"
		if r.Registration.Source == pb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN {
			source = "sunday_open"
		}
		record := []string{fmt.Sprint(r.Registration.Id), source, r.Contact.FullName, r.Contact.Email, r.Contact.Phone, r.Status, r.SubmittedAt, r.CheckedInAt, r.Notes}
		answers := map[string]string{}
		fields := map[string]*pb.FormField{}
		if r.Form != nil {
			for _, field := range r.Form.Fields {
				fields[field.Id] = field
			}
		}
		for _, a := range r.Answers {
			answers[a.FieldId] = answerText(a, fields[a.FieldId])
		}
		for _, id := range ids {
			record = append(record, answers[id])
		}
		for i := range record {
			record[i] = csvSafe(record[i])
		}
		if err := writer.Write(record); err != nil {
			return nil, internal(err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, internal(err)
	}
	return connect.NewResponse(&pb.ExportEventResponsesResponse{Csv: output.String(), FileName: fmt.Sprintf("event-%d-responses.csv", req.Msg.EventId)}), nil
}
