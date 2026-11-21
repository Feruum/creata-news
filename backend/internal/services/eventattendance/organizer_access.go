package eventattendance

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/checkin"
	"github.com/studyverse/ems-backend/internal/models"
)

// OrganizerAccessHandler delegates only QR presentation for one event. The
// caller must be wrapped in fresh cookie-only campus authentication.
func (s *Service) OrganizerAccessHandler(secret, origins string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Type", "application/json")
		respondError := func(status int, code, message string) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
		}
		if r.Method != "POST" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			respondError(400, "invalid_argument", "Invalid request")
			return
		}
		var input struct {
			EventID int32  `json:"eventId"`
			Origin  string `json:"origin"`
			State   string `json:"state"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF || input.EventID <= 0 || len(input.State) < 16 || len(input.State) > 128 {
			respondError(400, "invalid_argument", "Invalid event request")
			return
		}
		if !checkin.AllowedOrigin(origins, input.Origin) {
			respondError(403, "permission_denied", "Return site is not allowed")
			return
		}
		if len(secret) < 32 || auth.IsSessionValidationUnavailable(r.Context()) {
			respondError(503, "unavailable", "Sign-in is temporarily unavailable")
			return
		}
		_, err := s.GetCheckInInfo(r.Context(), connect.NewRequest(&eventsv1.GetCheckInInfoRequest{EventId: input.EventID}))
		if err != nil {
			status := 503
			if connect.CodeOf(err) == connect.CodeUnauthenticated {
				status = 401
			}
			if connect.CodeOf(err) == connect.CodePermissionDenied {
				status = 403
			}
			if connect.CodeOf(err) == connect.CodeNotFound {
				status = 404
			}
			respondError(status, connect.CodeOf(err).String(), "Could not authorize QR display")
			return
		}
		var event models.Event
		if err := s.db.NewSelect().Model(&event).Where("id = ?", input.EventID).Where("publish_on_sunday_open = true").Scan(r.Context()); err != nil {
			respondError(404, "not_found", "Sunday Open event not found")
			return
		}
		ticket, err := checkin.Sign(secret, checkin.Claim{Purpose: "organizer", EventID: input.EventID, Subject: auth.GetUserID(r.Context()), Origin: input.Origin, ExpiresAt: time.Now().Add(checkin.TicketLifetime).Unix()})
		if err != nil || auth.GetUserID(r.Context()) == "" {
			respondError(503, "unavailable", errors.New("could not authorize QR display").Error())
			return
		}
		target, _ := url.Parse(fmt.Sprintf("%s/organizer/events/%d/qr", input.Origin, input.EventID))
		target.Fragment = url.Values{"ticket": {ticket}, "state": {input.State}}.Encode()
		_ = json.NewEncoder(w).Encode(map[string]string{"url": target.String()})
	})
}
