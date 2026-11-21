package eventattendance

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/studyverse/ems-backend/internal/authz"
)

func TestOrganizerAccessRejectsAnonymousAndUntrustedReturns(t *testing.T) {
	svc := NewService(nil, authz.NewEnforcer(nil, nil), "https://map.example")
	for _, test := range []struct {
		origin   string
		expected int
	}{
		{"https://sunday.example", 401},
		{"https://sunday.example.evil.test", 403},
		{"https://user@sunday.example", 403},
	} {
		r := httptest.NewRequest("POST", "/check-in/organizer-access", strings.NewReader(`{"eventId":1,"origin":"`+test.origin+`","state":"client-state-12345678"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		svc.OrganizerAccessHandler(strings.Repeat("s", 32), "https://sunday.example").ServeHTTP(w, r)
		if w.Code != test.expected || strings.Contains(w.Body.String(), "ticket=") {
			t.Fatalf("origin %s: %d %s", test.origin, w.Code, w.Body)
		}
	}
	// Simple cross-site form submissions cannot initiate a credentialed exchange.
	r := httptest.NewRequest(http.MethodPost, "/check-in/organizer-access", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	svc.OrganizerAccessHandler(strings.Repeat("s", 32), "https://sunday.example").ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("simple POST accepted: %d", w.Code)
	}
}
