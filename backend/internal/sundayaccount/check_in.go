package sundayaccount

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/studyverse/ems-backend/internal/checkin"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

// startCheckIn exchanges a currently valid rotating QR for a bounded sign-in
// continuation. It never records attendance or accepts an identity from the URL.
func (h *Handler) startCheckIn(w http.ResponseWriter, r *http.Request) {
	var event models.Event
	err := h.db.NewSelect().Model(&event).Where("check_in_token = ?", r.URL.Query().Get("token")).Scan(r.Context())
	if err != nil {
		registrationError(w, err)
		return
	}
	if !event.PublishOnSundayOpen {
		fail(w, 404, "Event not found")
		return
	}
	if !checkin.Active(&event, time.Now()) {
		failCode(w, 409, "failed_precondition", "Check-in is closed")
		return
	}
	if event.CheckInSecret == nil || !checkin.ValidCode(*event.CheckInSecret, r.URL.Query().Get("code"), time.Now()) {
		fail(w, 400, "QR code has expired. Scan the current code.")
		return
	}
	origin := strings.TrimSpace(strings.Split(h.cfg.Origin, ",")[0])
	if !checkin.AllowedOrigin(h.cfg.Origin, origin) {
		fail(w, 503, "Guest check-in is not configured")
		return
	}
	ticket, err := checkin.Sign(h.cfg.Secret, checkin.Claim{
		Purpose: "guest", EventID: event.ID, EventToken: *event.CheckInToken, Origin: origin,
		ExpiresAt: time.Now().Add(checkin.TicketLifetime).Unix(),
	})
	if err != nil {
		fail(w, 503, "Could not continue check-in")
		return
	}
	target, _ := url.Parse(origin + "/check-in")
	target.Fragment = url.Values{"ticket": {ticket}}.Encode()
	writeJSON(w, 200, map[string]string{"url": target.String()})
}

func (h *Handler) continuation(r *http.Request, ticket string, lock bool, db bun.IDB) (*models.Event, error) {
	claim, err := checkin.Verify(h.cfg.Secret, ticket, "guest", time.Now())
	if err != nil {
		return nil, err
	}
	if !checkin.AllowedOrigin(h.cfg.Origin, claim.Origin) ||
		(r.Header.Get("Origin") != "" && r.Header.Get("Origin") != claim.Origin) {
		return nil, checkin.ErrInvalidTicket
	}
	var event models.Event
	query := db.NewSelect().Model(&event).Where("id = ?", claim.EventID).Where("check_in_token = ?", claim.EventToken)
	if lock {
		// Serialize seat allocation while allowing notification account/event FK locks.
		query = query.For("NO KEY UPDATE")
	}
	if err := query.Scan(r.Context()); err != nil {
		return nil, err
	}
	if !event.PublishOnSundayOpen {
		return nil, errHiddenEvent
	}
	if !checkin.Active(&event, time.Now()) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("check-in is closed"))
	}
	return &event, nil
}

func (h *Handler) checkInInfo(w http.ResponseWriter, r *http.Request) {
	event, err := h.continuation(r, r.Header.Get("X-Check-In-Ticket"), false, h.db)
	if err != nil {
		checkInError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{
		"eventId": event.ID, "title": event.Title,
		"universityUrl": strings.TrimRight(h.cfg.MapAppURL, "/") + "/check-in/" + url.PathEscape(*event.CheckInToken) + "?university=1",
	})
}

func checkInError(w http.ResponseWriter, err error) {
	if errors.Is(err, checkin.ErrInvalidTicket) {
		fail(w, 400, "Check-in link has expired. Scan the current QR code.")
		return
	}
	registrationError(w, err)
}

func (h *Handler) completeCheckIn(w http.ResponseWriter, r *http.Request, p *principal) {
	var input struct {
		Ticket string `json:"ticket"`
	}
	if !decode(w, r, &input) {
		return
	}
	var result Registration
	err := h.db.RunInTx(r.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		current, err := h.lockSessionAccount(ctx, tx, p)
		if err != nil {
			return err
		}
		p.Account = *current
		phone, err := normalizePhone(current.Phone)
		if _, emailErr := normalizeEmail(current.Email); err != nil || emailErr != nil || !validName(current.FullName) {
			return errProfileIncomplete
		}
		event, err := h.continuation(r, input.Ticket, true, tx)
		if err != nil {
			return err
		}
		var existing struct {
			ID     int32
			Status string
		}
		err = tx.NewRaw(`SELECT id,status FROM external_registrations WHERE event_id=? AND sunday_open_account_id=?`, event.ID, p.ID).Scan(ctx, &existing)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		newSeat := errors.Is(err, sql.ErrNoRows) || existing.Status != "registered"
		if newSeat {
			// The QR confirms admission without a form, approval, or capacity check.
			// Contact identity conflicts must still be rejected.
			var phoneInUse bool
			if err := tx.NewRaw(`SELECT EXISTS(SELECT 1 FROM external_registrations WHERE event_id=? AND phone=? AND ?<>'' AND sunday_open_account_id IS DISTINCT FROM ?::uuid)`, event.ID, phone, phone, p.ID).Scan(ctx, &phoneInUse); err != nil {
				return err
			}
			if phoneInUse {
				return errAlreadyRegistered
			}
		}
		var snapshot string
		if err := tx.NewRaw(`SELECT `+snapshotSQL+` FROM events e WHERE e.id=?`, event.ID).Scan(ctx, &snapshot); err != nil {
			return err
		}
		var row struct {
			ID                        int32
			RegisteredAt, CheckedInAt time.Time
		}
		if existing.ID == 0 {
			err = tx.NewRaw(`INSERT INTO external_registrations(event_id,sunday_open_account_id,full_name,email,phone,status,snapshot,checked_in_at) VALUES(?,?,?,?,NULLIF(?,''),'registered',?::jsonb,NOW()) RETURNING id,registered_at,checked_in_at`, event.ID, p.ID, p.FullName, p.Email, phone, snapshot).Scan(ctx, &row)
		} else {
			err = tx.NewRaw(`UPDATE external_registrations SET status='registered',registered_at=CASE WHEN status<>'registered' THEN NOW() ELSE registered_at END,cancelled_at=NULL,checked_in_at=COALESCE(checked_in_at,NOW()),updated_at=NOW(),snapshot=?::jsonb WHERE id=? AND sunday_open_account_id=? RETURNING id,registered_at,checked_in_at`, snapshot, existing.ID, p.ID).Scan(ctx, &row)
		}
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(snapshot), &result); err != nil {
			return err
		}
		result.ID, result.EventID, result.Status = row.ID, event.ID, "registered"
		result.RegisteredAt, result.CheckedInAt = row.RegisteredAt, &row.CheckedInAt
		if newSeat {
			return enqueue(ctx, tx, fmt.Sprintf("registered:%d:%d", row.ID, row.RegisteredAt.UnixNano()), p.ID, row.ID, event.ID, "registered", []byte(snapshot), time.Now())
		}
		return nil
	})
	if err != nil {
		checkInError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"registration": result})
}
