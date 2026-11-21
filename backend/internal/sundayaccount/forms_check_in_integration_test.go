//go:build integration

package sundayaccount

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/studyverse/ems-backend/internal/registrationforms"
)

func configureCheckInForm(t *testing.T, h *Handler, publish bool) {
	t.Helper()
	ctx := context.Background()
	for _, table := range []string{"event_registration_forms", "event_registration_form_versions"} {
		if _, err := h.db.ExecContext(ctx, "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	if publish {
		if _, err := h.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO event_registration_form_versions(id,event_id,definition) VALUES('%s',1,'{"eventId":1,"versionId":"%s","approvalMode":"APPROVAL_MODE_MANUAL","fields":[{"id":"required","label":"Required","type":"FORM_FIELD_TYPE_SHORT_TEXT","required":true}]}'); INSERT INTO event_registration_forms(event_id,draft,revision,published_version_id) VALUES(1,'{}',1,'%s')`, sundayFormVersion, sundayFormVersion, sundayFormVersion)); err != nil {
			t.Fatal(err)
		}
	}
	h.WithForms(registrationforms.NewService(h.db, nil, nil))
}

func TestGuestCheckInBypassesFormAndApproval(t *testing.T) {
	for _, tc := range []struct {
		name, status string
		forms        bool
	}{
		{"pending without injected forms", "pending_approval", false},
		{"rejected without injected forms", "rejected", false},
		{"pending with published form", "pending_approval", true},
		{"rejected with published form", "rejected", true},
		{"unregistered with published form", "", true},
		{"cancelled with published form", "cancelled", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := accountFixture(t)
			seedCheckIn(t, h)
			if tc.forms {
				configureCheckInForm(t, h, true)
			}
			cookie, csrf, account := startGuest(t, h, "form-scan@example.test", "")
			if tc.status != "" {
				if _, err := h.db.ExecContext(context.Background(), `INSERT INTO external_registrations(event_id,sunday_open_account_id,full_name,email,status) VALUES(1,?,'Applicant','form-scan@example.test',?)`, account.ID, tc.status); err != nil {
					t.Fatal(err)
				}
			}
			ticket := guestTicket(t, h)
			w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, cookie, csrf)
			if w.Code != 200 || !strings.Contains(w.Body.String(), "checkedInAt") {
				t.Fatalf("valid QR check-in blocked: %d %s", w.Code, w.Body)
			}
			var state struct {
				Registrations, CheckedIn, Notifications int
				Status                                  string
			}
			if err := h.db.NewRaw(`SELECT count(*) AS registrations,count(*) FILTER(WHERE checked_in_at IS NOT NULL) AS checked_in,COALESCE(max(status::text),'') AS status,(SELECT count(*) FROM sunday_open_notification_deliveries WHERE account_id=?) AS notifications FROM external_registrations WHERE event_id=1 AND sunday_open_account_id=?`, account.ID, account.ID).Scan(context.Background(), &state); err != nil {
				t.Fatal(err)
			}
			if state.Registrations != 1 || state.CheckedIn != 1 || state.Notifications != 1 || state.Status != "registered" {
				t.Fatalf("walk-in did not register and check in exactly once: %+v", state)
			}
		})
	}
}

func TestGuestCheckInAllowsConfirmedRegistrationWithPublishedForm(t *testing.T) {
	h, _ := accountFixture(t)
	seedCheckIn(t, h)
	configureCheckInForm(t, h, true)
	cookie, csrf, account := startGuest(t, h, "confirmed-form-scan@example.test", "")
	if _, err := h.db.ExecContext(context.Background(), `INSERT INTO external_registrations(event_id,sunday_open_account_id,full_name,email,status) VALUES(1,?,'Confirmed','confirmed-form-scan@example.test','registered')`, account.ID); err != nil {
		t.Fatal(err)
	}
	w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": guestTicket(t, h)}, cookie, csrf)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"checkedInAt"`) {
		t.Fatalf("confirmed participant refused: %d %s", w.Code, w.Body)
	}
}

func TestGuestCheckInAllowsWalkInWithoutPublishedForm(t *testing.T) {
	h, _ := accountFixture(t)
	seedCheckIn(t, h)
	configureCheckInForm(t, h, false)
	cookie, csrf, _ := startGuest(t, h, "no-form-scan@example.test", "")
	w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": guestTicket(t, h)}, cookie, csrf)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"checkedInAt"`) {
		t.Fatalf("no-form walk-in refused: %d %s", w.Code, w.Body)
	}
}
