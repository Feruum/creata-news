//go:build integration

package sundayaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/studyverse/ems-backend/internal/registrationforms"
)

const sundayFormVersion = "27c85842-e46a-4579-99e6-d5193a9a2584"

func accountFormsFixture(t *testing.T, mode string) (*Handler, *recordingMailer) {
	t.Helper()
	h, mailer := accountFixture(t)
	for _, table := range []string{"event_registration_forms", "event_registration_form_versions", "event_registration_responses"} {
		if _, err := h.db.ExecContext(context.Background(), "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	statements := []string{
		`INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,capacity,check_in_secret,check_in_token) VALUES(1,'Application event','{}',1,1,NOW()+INTERVAL '2 days',NOW()+INTERVAL '3 days','{}','published',true,1,'a','b')`,
		`INSERT INTO external_registrations(event_id,full_name,email,phone,status) VALUES(1,'Existing seat','seat@example.test','+77010000000','registered')`,
		fmt.Sprintf(`INSERT INTO event_registration_form_versions(id,event_id,definition) VALUES('%s',1,'{"eventId":1,"versionId":"%s","approvalMode":"%s","fields":[{"id":"required","label":"Required","type":"FORM_FIELD_TYPE_SHORT_TEXT","required":true}]}')`, sundayFormVersion, sundayFormVersion, mode),
		fmt.Sprintf(`INSERT INTO event_registration_forms(event_id,draft,revision,published_version_id) VALUES(1,'{}',1,'%s')`, sundayFormVersion),
	}
	for _, statement := range statements {
		if _, err := h.db.ExecContext(context.Background(), statement); err != nil {
			t.Fatal(err)
		}
	}
	h.WithForms(registrationforms.NewService(h.db, nil, nil))
	return h, mailer
}

func validSundayFormSubmission() map[string]any {
	return map[string]any{"eventId": 1, "formSubmission": map[string]any{"formVersionId": sundayFormVersion, "answers": []any{map[string]any{"fieldId": "required", "text": "My answer"}}}}
}

func TestManualSundayApplicationValidatesBeforeCapacityAndCancelsAtomically(t *testing.T) {
	h, mailer := accountFormsFixture(t, "APPROVAL_MODE_MANUAL")
	mailer.fail = true
	cookie, csrf, account := startGuest(t, h, "application@example.test", "")
	accountID := account.ID
	for _, tc := range []struct {
		body   any
		status int
	}{
		{map[string]any{"eventId": 1}, 409},
		{map[string]any{"eventId": 1, "formSubmission": map[string]any{"formVersionId": "stale"}}, 409},
		{map[string]any{"eventId": 1, "formSubmission": map[string]any{"formVersionId": sundayFormVersion}}, 400},
	} {
		w := callAccount(t, h, "POST", "registrations", tc.body, cookie, csrf)
		if w.Code != tc.status {
			t.Fatalf("submission validation: %d %s", w.Code, w.Body)
		}
	}
	w := callAccount(t, h, "POST", "registrations", validSundayFormSubmission(), cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("manual application blocked by full capacity: %d %s", w.Code, w.Body)
	}
	var saved struct {
		Registration Registration `json:"registration"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Registration.Status != "pending_approval" {
		t.Fatalf("application consumed a seat: %s", w.Body)
	}
	var deliveries int
	if err := h.db.NewRaw(`SELECT count(*) FROM sunday_open_notification_deliveries WHERE account_id=?`, accountID).Scan(context.Background(), &deliveries); err != nil {
		t.Fatal(err)
	}
	if deliveries != 0 {
		t.Fatal("pending application sent a confirmation")
	}
	w = callAccount(t, h, "DELETE", fmt.Sprintf("registrations/%d", saved.Registration.ID), nil, cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("cancel pending application: %d %s", w.Code, w.Body)
	}
	var responseStatus string
	if err := h.db.NewRaw(`SELECT status FROM event_registration_responses WHERE external_registration_id=? AND is_current`, saved.Registration.ID).Scan(context.Background(), &responseStatus); err != nil {
		t.Fatal(err)
	}
	if responseStatus != "cancelled" {
		t.Fatalf("response cancellation not atomic: %s", responseStatus)
	}
	w = callAccount(t, h, "POST", "registrations", validSundayFormSubmission(), cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("resubmit cancelled application: %d %s", w.Code, w.Body)
	}
	var attempts int
	if err := h.db.NewRaw(`SELECT count(*) FROM event_registration_responses WHERE external_registration_id=?`, saved.Registration.ID).Scan(context.Background(), &attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("previous response attempt lost: %d", attempts)
	}
	if _, err := h.db.ExecContext(context.Background(), `UPDATE external_registrations SET status='rejected' WHERE id=?`, saved.Registration.ID); err != nil {
		t.Fatal(err)
	}
	w = callAccount(t, h, "POST", "registrations", validSundayFormSubmission(), cookie, csrf)
	if w.Code != 409 {
		t.Fatalf("rejected application reapplied: %d %s", w.Code, w.Body)
	}
	w = callAccount(t, h, "GET", "registrations", nil, cookie, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"rejected"`) {
		t.Fatalf("rejected application missing from history: %d %s", w.Code, w.Body)
	}
}

func TestAutomaticSundayFormStillChecksCapacity(t *testing.T) {
	h, mailer := accountFormsFixture(t, "APPROVAL_MODE_AUTOMATIC")
	cookie, csrf, _ := loginAccount(t, h, mailer, "automatic@example.test")
	w := callAccount(t, h, "POST", "registrations", validSundayFormSubmission(), cookie, csrf)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "resource_exhausted") {
		t.Fatalf("automatic application bypassed capacity: %d %s", w.Code, w.Body)
	}
	var attempts int
	if err := h.db.NewRaw(`SELECT count(*) FROM event_registration_responses`).Scan(context.Background(), &attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 0 {
		t.Fatal("failed registration saved a response")
	}
}

func TestSundayUploadsRequireCSRFAndRejectForgedOwner(t *testing.T) {
	h, mailer := accountFixture(t)
	cookie, csrf, _ := loginAccount(t, h, mailer, "upload@example.test")
	for _, path := range []string{"forms/uploads", "forms/uploads/test/complete"} {
		w := callAccount(t, h, "POST", path, map[string]any{}, cookie, "")
		if w.Code != 403 {
			t.Fatalf("upload omitted CSRF: %d %s", w.Code, w.Body)
		}
	}
	w := callAccount(t, h, "POST", "forms/uploads", map[string]any{"eventId": 1, "accountId": "victim"}, cookie, csrf)
	if w.Code != 400 {
		t.Fatalf("upload owner accepted from body: %d %s", w.Code, w.Body)
	}
}
