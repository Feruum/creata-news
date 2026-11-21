package sundayaccount

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"github.com/studyverse/ems-backend/internal/registration"
	"github.com/studyverse/ems-backend/internal/registrationfiles"
	"github.com/studyverse/ems-backend/internal/registrationforms"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/encoding/protojson"
)

type Registration struct {
	ID           int32      `json:"id"`
	EventID      int32      `json:"eventId"`
	Slug         string     `json:"slug"`
	Title        string     `json:"title"`
	StartTime    time.Time  `json:"startTime"`
	EndTime      time.Time  `json:"endTime"`
	Status       string     `json:"status"`
	EventStatus  string     `json:"eventStatus"`
	LocationText string     `json:"locationText"`
	RegisteredAt time.Time  `json:"registeredAt"`
	CheckedInAt  *time.Time `json:"checkedInAt,omitempty"`
}

// The snapshot contains participant-visible event metadata only. It does not
// include an event description, private organizer data or unpublished changes.
const locationSQL = `COALESCE(CASE e.structured_location->>'type' WHEN 'campus_room' THEN (e.structured_location->>'room_name') || CASE WHEN e.structured_location->>'floor' IS NOT NULL THEN ', Floor '||(e.structured_location->>'floor') ELSE '' END WHEN 'external' THEN e.structured_location->>'external_address' WHEN 'online' THEN COALESCE(e.structured_location->>'online_platform','Online') WHEN 'hybrid' THEN concat_ws(' + ',COALESCE(e.structured_location->>'room_name',e.structured_location->>'external_address'),COALESCE(e.structured_location->>'online_platform','Online')) END,e.structured_location->>'text',e.structured_location->>'address','')`
const snapshotSQL = `jsonb_build_object('slug',COALESCE((SELECT slug FROM event_slugs WHERE event_id=e.id AND is_current=true LIMIT 1),''),'title',e.title,'startTime',e.start_time,'endTime',e.end_time,'eventStatus',e.status,'locationText',` + locationSQL + `,'locationRevision',md5(e.structured_location::text),'eventRevision',e.updated_at)`

type eventSnapshot struct {
	Registration
	LocationRevision string `json:"locationRevision"`
	EventRevision    string `json:"eventRevision"`
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request, p *principal) {
	var rows []struct {
		ID, EventID  int32
		Status       string
		RegisteredAt time.Time
		CheckedInAt  *time.Time
		Snapshot     json.RawMessage
	}
	if err := h.db.NewRaw(`SELECT xr.id,xr.event_id,xr.status,xr.registered_at,xr.checked_in_at,CASE WHEN e.status='published' AND e.publish_on_sunday_open THEN `+snapshotSQL+` WHEN e.status='cancelled' THEN jsonb_set(xr.snapshot,'{eventStatus}','"cancelled"'::jsonb) ELSE jsonb_set(xr.snapshot,'{eventStatus}','"hidden"'::jsonb) END AS snapshot FROM external_registrations xr JOIN events e ON e.id=xr.event_id WHERE xr.sunday_open_account_id=? ORDER BY xr.registered_at DESC,xr.id DESC LIMIT 500`, p.ID).Scan(r.Context(), &rows); err != nil {
		fail(w, 503, "Could not load registrations")
		return
	}
	result := make([]Registration, 0, len(rows))
	for _, row := range rows {
		var reg Registration
		if err := json.Unmarshal(row.Snapshot, &reg); err != nil {
			fail(w, 503, "Could not load registration details")
			return
		}
		reg.ID = row.ID
		reg.EventID = row.EventID
		reg.Status = row.Status
		reg.RegisteredAt = row.RegisteredAt
		reg.CheckedInAt = row.CheckedInAt
		result = append(result, reg)
	}
	writeJSON(w, 200, map[string]any{"registrations": result})
}
func (h *Handler) register(w http.ResponseWriter, r *http.Request, p *principal) {
	var input struct {
		EventID        int32           `json:"eventId"`
		FormSubmission json.RawMessage `json:"formSubmission"`
	}
	if !decodeLimit(w, r, &input, 2*1024*1024) {
		return
	}
	var submission *pb.FormSubmission
	if len(input.FormSubmission) > 0 && string(input.FormSubmission) != "null" {
		submission = new(pb.FormSubmission)
		if err := protojson.Unmarshal(input.FormSubmission, submission); err != nil {
			fail(w, 400, "Invalid form submission")
			return
		}
	}
	if input.EventID <= 0 {
		fail(w, 400, "Invalid event")
		return
	}
	if _, err := normalizeEmail(p.Email); err != nil || !validName(p.FullName) {
		failCode(w, 409, "failed_precondition", "Complete your name and email in your profile before registering")
		return
	}
	phone, err := normalizePhone(p.Phone)
	if err != nil {
		failCode(w, 409, "failed_precondition", "Update the saved phone number in your profile before registering")
		return
	}
	var result Registration
	err = h.db.RunInTx(r.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		current, err := h.lockSessionAccount(ctx, tx, p)
		if err != nil {
			return err
		}
		p.Account = *current
		phone, err = normalizePhone(current.Phone)
		if _, emailErr := normalizeEmail(current.Email); err != nil || emailErr != nil || !validName(current.FullName) {
			return errProfileIncomplete
		}
		event, err := registration.LockOpenEvent(ctx, tx, input.EventID, time.Now)
		if err != nil {
			return err
		}
		var visible bool
		if err := tx.NewRaw(`SELECT publish_on_sunday_open FROM events WHERE id=?`, event.ID).Scan(ctx, &visible); err != nil {
			return err
		}
		if !visible {
			return errHiddenEvent
		}
		var existing struct {
			ID     int32
			Status string
		}
		err = tx.NewRaw(`SELECT id,status FROM external_registrations WHERE event_id=? AND sunday_open_account_id=?`, event.ID, p.ID).Scan(ctx, &existing)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil && existing.Status != "cancelled" {
			return errAlreadyRegistered
		}
		var phoneInUse bool
		if err := tx.NewRaw(`SELECT EXISTS(SELECT 1 FROM external_registrations WHERE event_id=? AND phone=? AND ?<>'' AND sunday_open_account_id IS DISTINCT FROM ?::uuid)`, event.ID, phone, phone, p.ID).Scan(ctx, &phoneInUse); err != nil {
			return err
		}
		if phoneInUse {
			return errAlreadyRegistered
		}
		status := "registered"
		var prepared *registrationforms.Prepared
		if h.forms != nil {
			prepared, err = h.forms.Prepare(ctx, tx, event.ID, submission)
			if err != nil {
				return err
			}
			status = prepared.Status
		}
		if status == "registered" {
			if err := registration.EnsureCapacity(ctx, tx, event); err != nil {
				return err
			}
		}
		var snapshot string
		if err := tx.NewRaw(`SELECT `+snapshotSQL+` FROM events e WHERE e.id=?`, event.ID).Scan(ctx, &snapshot); err != nil {
			return err
		}
		var id int32
		if err := tx.NewRaw(`INSERT INTO external_registrations(event_id,sunday_open_account_id,full_name,email,phone,status,registered_at,snapshot) SELECT ?,id,full_name,COALESCE(login_email,email,contact_email),NULLIF(?,''),?,NOW(),?::jsonb FROM sunday_open_accounts WHERE id=? ON CONFLICT(event_id,sunday_open_account_id) DO UPDATE SET full_name=EXCLUDED.full_name,email=EXCLUDED.email,phone=EXCLUDED.phone,status=EXCLUDED.status,registered_at=NOW(),cancelled_at=NULL,updated_at=NOW(),snapshot=EXCLUDED.snapshot RETURNING id`, event.ID, phone, status, string(snapshot), p.ID).Scan(ctx, &id); err != nil {
			return err
		}
		if h.forms != nil {
			if _, err := h.forms.Record(ctx, tx, prepared, registrationfiles.Owner{AccountID: p.ID}, pb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN, id, &pb.ResponseContact{FullName: p.FullName, Email: p.Email, Phone: phone}); err != nil {
				return err
			}
		}
		if err := json.Unmarshal([]byte(snapshot), &result); err != nil {
			return err
		}
		result.ID = id
		result.EventID = event.ID
		result.Status = status
		result.RegisteredAt = time.Now()
		if status != "registered" {
			return nil
		}
		// Durable intent is committed with the confirmed seat. Sending happens asynchronously.
		return enqueue(ctx, tx, fmt.Sprintf("registered:%d:%d", id, result.RegisteredAt.UnixNano()), p.ID, id, event.ID, "registered", []byte(snapshot), time.Now())
	})
	if err != nil {
		registrationError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"registration": result})
}

var errHiddenEvent = errors.New("event not found")
var errProfileIncomplete = errors.New("profile incomplete")
var errAlreadyRegistered = errors.New("already registered")

func registrationError(w http.ResponseWriter, err error) {
	var state interface{ SQLState() string }
	if errors.As(err, &state) && state.SQLState() == "23505" {
		err = errAlreadyRegistered
	}
	switch {
	case errors.Is(err, errInvalidCredentials):
		fail(w, 401, "Sign in to continue")
	case errors.Is(err, errProfileIncomplete):
		failCode(w, 409, "failed_precondition", "Complete your name and email in your profile before registering")
	case errors.Is(err, errHiddenEvent), errors.Is(err, sql.ErrNoRows):
		fail(w, 404, "Event or registration not found")
	case errors.Is(err, errAlreadyRegistered):
		failCode(w, 409, "already_exists", "You are already registered")
	case connect.CodeOf(err) == connect.CodeNotFound:
		failCode(w, 404, "not_found", connectMessage(err))
	case connect.CodeOf(err) == connect.CodeInvalidArgument:
		failCode(w, 400, "invalid_argument", connectMessage(err))
	case connect.CodeOf(err) == connect.CodePermissionDenied:
		failCode(w, 403, "permission_denied", connectMessage(err))
	case connect.CodeOf(err) == connect.CodeUnauthenticated:
		failCode(w, 401, "unauthenticated", connectMessage(err))
	case connect.CodeOf(err) == connect.CodeAborted:
		failCode(w, 409, "aborted", connectMessage(err))
	case connect.CodeOf(err) == connect.CodeUnavailable:
		failCode(w, 503, "unavailable", connectMessage(err))
	case connect.CodeOf(err) == connect.CodeResourceExhausted:
		failCode(w, 409, "resource_exhausted", connectMessage(err))
	case connect.CodeOf(err) == connect.CodeFailedPrecondition:
		failCode(w, 409, "failed_precondition", connectMessage(err))
	default:
		slog.Error("Sunday Open registration failed", "error", err)
		fail(w, 503, "Could not save registration")
	}
}
func connectMessage(err error) string {
	var coded *connect.Error
	if errors.As(err, &coded) {
		return coded.Message()
	}
	return err.Error()
}
func (h *Handler) cancel(w http.ResponseWriter, r *http.Request, p *principal, rawID string) {
	id, err := strconv.ParseInt(rawID, 10, 32)
	if err != nil || id <= 0 {
		fail(w, 400, "Invalid registration")
		return
	}
	err = h.db.RunInTx(r.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := h.lockSessionAccount(ctx, tx, p); err != nil {
			return err
		}
		var row struct {
			ID, EventID int32
			Status      string
			Snapshot    json.RawMessage
		}
		// Session ownership is mandatory; a body, query or Telegram ID cannot select the account.
		if err := tx.NewRaw(`SELECT event_id FROM external_registrations WHERE id=? AND sunday_open_account_id=?`, id, p.ID).Scan(ctx, &row.EventID); err != nil {
			return err
		}
		var locked int32
		if err := tx.NewRaw(`SELECT id FROM events WHERE id=? FOR UPDATE`, row.EventID).Scan(ctx, &locked); err != nil {
			return err
		}
		if err := tx.NewRaw(`SELECT id,event_id,status,snapshot FROM external_registrations WHERE id=? AND sunday_open_account_id=? AND status IN ('registered','pending_approval') FOR UPDATE`, id, p.ID).Scan(ctx, &row); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE external_registrations SET status='cancelled',cancelled_at=NOW(),updated_at=NOW() WHERE id=?`, row.ID); err != nil {
			return err
		}
		if h.forms != nil {
			if err := h.forms.Cancel(ctx, tx, pb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN, row.ID); err != nil {
				return err
			}
		}
		if row.Status != "registered" {
			return nil
		}
		return enqueue(ctx, tx, fmt.Sprintf("cancelled:%d:%d", id, time.Now().UnixNano()), p.ID, row.ID, row.EventID, "cancelled", row.Snapshot, time.Now())
	})
	if err != nil {
		registrationError(w, err)
		return
	}
	success(w)
}
func enqueue(ctx context.Context, tx bun.Tx, key, account string, registrationID, eventID int32, kind string, payload json.RawMessage, at time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO sunday_open_notification_deliveries(delivery_key,account_id,registration_id,event_id,kind,payload,available_at) VALUES(?,?,?,?,?,?::jsonb,?) ON CONFLICT(delivery_key) DO NOTHING`, key, account, registrationID, eventID, kind, string(payload), at)
	return err
}
