//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"github.com/studyverse/ems-backend/gen/registrationforms/v1/registrationformsv1connect"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/registrationforms"
	"github.com/studyverse/ems-backend/internal/sundayaccount"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/protobuf/encoding/protojson"
)

// Exercise the real Connect wire, Sunday session/CSRF HTTP API and shared DB.
// Only campus identity verification/permissions are replaced by the test fixture.
func TestPublishedFormSundaySubmissionOrganizerApprovalHTTPRoundTrip(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL must name a disposable migrated PostgreSQL")
	}
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	for _, table := range []string{"users", "events", "event_slugs", "event_registrations", "event_attendance", "external_registrations", "sunday_open_accounts", "sunday_open_sessions", "sunday_open_auth_challenges", "sunday_open_telegram_links", "sunday_open_telegram_updates", "sunday_open_notification_deliveries", "event_registration_forms", "event_registration_form_versions", "event_registration_responses", "event_registration_response_reads", "event_registration_uploads"} {
		if _, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users(id,kratos_id,username,email,password) VALUES(1,'http-organizer','Organizer','organizer@example.test','!disabled!');
 INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,capacity,check_in_secret,check_in_token,publish_on_map,publish_on_sunday_open)
 VALUES(1,'Shared HTTP event','{}',1,1,NOW(),NOW()+INTERVAL '1 day','{}','published',1,'http-secret','http-token',true,true)`); err != nil {
		t.Fatal(err)
	}
	forms := registrationforms.NewService(db, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
	path, handler := registrationformsv1connect.NewRegistrationFormsServiceHandler(forms)
	mux := http.NewServeMux()
	mux.Handle(path, uncachedFormsHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test-Organizer") == "yes" {
			r = r.WithContext(context.WithValue(r.Context(), auth.UserIDKey, "http-organizer"))
		}
		handler.ServeHTTP(w, r)
	})))
	mux.Handle("/api/sundayopen/rpc/", sundayPublicRPCHandler(handler))
	mux.Handle("/api/sundayopen/account/", sundayaccount.New(db, sundayaccount.Config{
		Origin: "https://sunday.example", Secret: strings.Repeat("s", 32),
	}, nil).WithForms(forms))
	server := httptest.NewServer(mux)
	defer server.Close()
	organizer := registrationformsv1connect.NewRegistrationFormsServiceClient(server.Client(), server.URL)
	save := connect.NewRequest(&pb.SaveFormDraftRequest{Form: &pb.RegistrationForm{
		EventId: 1, ApprovalMode: pb.ApprovalMode_APPROVAL_MODE_MANUAL,
		Fields: []*pb.FormField{{Id: "reason", Label: "Why attend?", Type: pb.FormFieldType_FORM_FIELD_TYPE_SHORT_TEXT, Required: true}},
	}})
	save.Header().Set("X-Test-Organizer", "yes")
	saved, err := organizer.SaveFormDraft(ctx, save)
	if err != nil {
		t.Fatal(err)
	}
	publish := connect.NewRequest(&pb.PublishRegistrationFormRequest{EventId: 1, ExpectedRevision: saved.Msg.Form.Revision})
	publish.Header().Set("X-Test-Organizer", "yes")
	published, err := organizer.PublishRegistrationForm(ctx, publish)
	if err != nil {
		t.Fatal(err)
	}
	public := registrationformsv1connect.NewRegistrationFormsServiceClient(server.Client(), server.URL+"/api/sundayopen/rpc")
	read, err := public.GetPublishedForm(ctx, connect.NewRequest(&pb.GetPublishedFormRequest{EventId: 1, Audience: pb.FormAudience_FORM_AUDIENCE_SUNDAY_OPEN}))
	if err != nil || read.Msg.Form.GetVersionId() != published.Msg.Form.VersionId {
		t.Fatalf("public form alias: %v %v", read, err)
	}
	if _, err := public.ListEventResponses(ctx, connect.NewRequest(&pb.ListEventResponsesRequest{EventId: 1})); err == nil {
		t.Fatal("organizer responses leaked through the public alias")
	}
	var cookie *http.Cookie
	var csrf string
	account := func(method, route string, body any, want int) []byte {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r, err := http.NewRequest(method, server.URL+"/api/sundayopen/account/"+route, bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Origin", "https://sunday.example")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-CSRF-Token", csrf)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		response, err := server.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != want {
			t.Fatalf("%s: HTTP %d, want %d: %s", route, response.StatusCode, want, data)
		}
		for _, next := range response.Cookies() {
			if next.Name == "sundayopen_session" {
				cookie = next
			}
		}
		return data
	}
	var session struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(account("POST", "start", map[string]string{"fullName": "New Participant", "email": "participant@example.test"}, 200), &session); err != nil {
		t.Fatal(err)
	}
	csrf = session.CSRFToken
	account("POST", "registrations", map[string]int{"eventId": 1}, 409)
	var before int
	if err := db.NewRaw("SELECT count(*) FROM external_registrations").Scan(ctx, &before); err != nil || before != 0 {
		t.Fatalf("profile/missing answers created a registration: %d %v", before, err)
	}
	submission, err := protojson.Marshal(&pb.FormSubmission{FormVersionId: published.Msg.Form.VersionId, Answers: []*pb.RegistrationAnswer{
		{FieldId: "reason", Value: &pb.RegistrationAnswer_Text{Text: "Learn and meet the community"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	account("POST", "registrations", map[string]any{"eventId": 1, "formSubmission": json.RawMessage(submission)}, 200)
	list := connect.NewRequest(&pb.ListEventResponsesRequest{EventId: 1})
	list.Header().Set("X-Test-Organizer", "yes")
	listed, err := organizer.ListEventResponses(ctx, list)
	if err != nil || len(listed.Msg.GetResponses()) != 1 {
		t.Fatalf("organizer list: %v %v", listed, err)
	}
	entry := listed.Msg.Responses[0]
	if entry.Status != "pending_approval" || entry.Contact.FullName != "New Participant" || entry.Answers[0].GetText() != "Learn and meet the community" || entry.Registration.Source != pb.RegistrationSource_REGISTRATION_SOURCE_SUNDAY_OPEN {
		t.Fatalf("shared response lost data: %v", entry)
	}
	review := connect.NewRequest(&pb.ReviewEventRegistrationRequest{EventId: 1, Registration: entry.Registration, ResponseId: entry.ResponseId, Action: pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_APPROVE})
	review.Header().Set("X-Test-Organizer", "yes")
	approved, err := organizer.ReviewEventRegistration(ctx, review)
	if err != nil || approved.Msg.Response.GetStatus() != "registered" {
		t.Fatalf("approval: %v %v", approved, err)
	}
	var history struct {
		Registrations []struct {
			Status string `json:"status"`
		} `json:"registrations"`
	}
	if err := json.Unmarshal(account("GET", "registrations", nil, 200), &history); err != nil || len(history.Registrations) != 1 || history.Registrations[0].Status != "registered" {
		t.Fatalf("participant did not receive confirmed status: %v %v", history.Registrations, err)
	}
}
