//go:build integration

package sundayaccount

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/checkin"
)

type organizerPolicy struct {
	denied bool
	calls  int
}

func (p *organizerPolicy) GetCheckInInfo(ctx context.Context, req *connect.Request[eventsv1.GetCheckInInfoRequest]) (*connect.Response[eventsv1.GetCheckInInfoResponse], error) {
	p.calls++
	if p.denied || auth.GetUserID(ctx) != "campus-organizer" || req.Msg.EventId != 1 {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("event permission required"))
	}
	base, code, until := "https://map.example/check-in/token-one", "rotating", time.Now().Add(30*time.Second).Format(time.RFC3339)
	return connect.NewResponse(&eventsv1.GetCheckInInfoResponse{CheckInPhase: eventsv1.CheckInPhase_CHECK_IN_PHASE_LIVE, CheckInUrl: &base, CurrentRotatingToken: &code, TokenValidUntil: &until}), nil
}

func TestOrganizerTicketIsEventScopedAndRechecksPermissions(t *testing.T) {
	h, _ := accountFixture(t)
	seedCheckIn(t, h)
	policy := &organizerPolicy{}
	h.WithOrganizer(policy)
	ticket, err := checkin.Sign(h.cfg.Secret, checkin.Claim{Purpose: "organizer", EventID: 1, Subject: "campus-organizer", Origin: "https://sunday.example", ExpiresAt: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	call := func(value string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", prefix+"organizer/check-in", nil)
		r.Header.Set("X-Organizer-Ticket", value)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, invalid := range []string{"", ticket + "forged", guestTicket(t, h)} {
		if w := call(invalid); w.Code != 403 || strings.Contains(w.Body.String(), "rotating") {
			t.Fatalf("invalid QR access %d %s", w.Code, w.Body)
		}
	}
	if policy.calls != 0 {
		t.Fatal("invalid ticket reached permission service")
	}
	if w := call(ticket); w.Code != 200 || !strings.Contains(w.Body.String(), "https://map.example/check-in/token-one?code=rotating") {
		t.Fatalf("valid QR %d %s", w.Code, w.Body)
	}
	policy.denied = true
	if w := call(ticket); w.Code != 403 || strings.Contains(w.Body.String(), "rotating") {
		t.Fatalf("revoked permission %d %s", w.Code, w.Body)
	}
	if policy.calls != 2 {
		t.Fatalf("permission checks %d", policy.calls)
	}
}
