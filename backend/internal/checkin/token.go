// Package checkin contains the event-scoped proofs shared by campus and guest check-in.
package checkin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/studyverse/ems-backend/internal/models"
)

const TicketLifetime = 15 * time.Minute
const RotationSeconds = 30

var ErrInvalidTicket = errors.New("invalid or expired check-in continuation")

type Claim struct {
	Purpose    string `json:"purpose"`
	EventID    int32  `json:"eventId"`
	EventToken string `json:"eventToken,omitempty"`
	Subject    string `json:"subject,omitempty"`
	Origin     string `json:"origin,omitempty"`
	ExpiresAt  int64  `json:"expiresAt"`
}

func Code(secret string, window int64) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(m, "%d", window)
	return hex.EncodeToString(m.Sum(nil))[:12]
}

func ValidCode(secret, code string, now time.Time) bool {
	if secret == "" || len(code) != 12 {
		return false
	}
	window := now.Unix() / RotationSeconds
	return hmac.Equal([]byte(Code(secret, window)), []byte(code)) ||
		hmac.Equal([]byte(Code(secret, window-1)), []byte(code))
}

func Active(event *models.Event, now time.Time) bool {
	if event.CompletedAt != nil {
		return false
	}
	grace := 120
	if event.CheckInGraceMinutes != nil {
		grace = int(*event.CheckInGraceMinutes)
	}
	phase := models.DeriveCheckInPhase(event.Status, now, event.StartTime, event.EndTime, grace)
	return phase == models.CheckInPhaseLive || phase == models.CheckInPhaseGrace
}

func Sign(secret string, claim Claim) (string, error) {
	if len(secret) < 32 || claim.EventID <= 0 || claim.ExpiresAt <= 0 ||
		(claim.Purpose != "guest" && claim.Purpose != "organizer") {
		return "", ErrInvalidTicket
	}
	payload, err := json.Marshal(claim)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + signature(secret, encoded), nil
}

func Verify(secret, token, purpose string, now time.Time) (Claim, error) {
	var claim Claim
	if len(secret) < 32 || len(token) > 4096 {
		return claim, ErrInvalidTicket
	}
	encoded, sig, found := strings.Cut(token, ".")
	if !found || !hmac.Equal([]byte(signature(secret, encoded)), []byte(sig)) {
		return claim, ErrInvalidTicket
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || json.Unmarshal(payload, &claim) != nil || claim.Purpose != purpose ||
		claim.EventID <= 0 || claim.ExpiresAt <= now.Unix() {
		return Claim{}, ErrInvalidTicket
	}
	if (purpose == "guest" && (claim.EventToken == "" || claim.Origin == "")) ||
		(purpose == "organizer" && (claim.Subject == "" || claim.Origin == "")) {
		return Claim{}, ErrInvalidTicket
	}
	return claim, nil
}

func signature(secret, encoded string) string {
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte("event-check-in-v1\x00" + encoded))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func AllowedOrigin(allowlist, origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" ||
		(u.Scheme != "https" && u.Scheme != "http") {
		return false
	}
	for _, allowed := range strings.Split(allowlist, ",") {
		if origin == strings.TrimSpace(allowed) {
			return true
		}
	}
	return false
}
