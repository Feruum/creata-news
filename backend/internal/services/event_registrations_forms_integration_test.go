//go:build integration

package services

import (
	"context"
	"os"
	"strconv"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/registrationforms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/encoding/protojson"
)

func formsRegistrationDB(t *testing.T) *bun.DB {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL must name a disposable test database")
	}
	db := testsupport.IsolatedDB(t)
	for _, table := range []string{"users", "organization_types", "organizations", "events", "event_registrations", "external_registrations", "sunday_open_accounts", "event_attendance", "event_registration_form_versions", "event_registration_forms", "event_registration_responses", "event_registration_response_reads", "event_registration_uploads"} {
		if _, err := db.ExecContext(context.Background(), "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestMapFormSubmissionPendingDoesNotAllocateOrConfirm(t *testing.T) {
	db := formsRegistrationDB(t)
	userID := seedAuthUser(t, db, "map-forms")
	eventID, _, _ := seedEvent(t, db, userID)
	ctx := authedContext("map-forms")
	version := uuid.NewString()
	f := &pb.RegistrationForm{EventId: eventID, VersionId: version, ApprovalMode: pb.ApprovalMode_APPROVAL_MODE_MANUAL, Fields: []*pb.FormField{{Id: "reason", Label: "Reason", Type: pb.FormFieldType_FORM_FIELD_TYPE_SHORT_TEXT, Required: true}}}
	raw, err := protojson.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE events SET capacity=1,publish_on_map=true WHERE id=?; INSERT INTO external_registrations(event_id,full_name,email,phone,status) VALUES(?,'Occupant','occupied@example.test','12345','registered'); INSERT INTO event_registration_form_versions(id,event_id,definition) VALUES(?::uuid,?,?::jsonb); INSERT INTO event_registration_forms(event_id,draft,published_version_id) VALUES(?,'{}',?::uuid)`, eventID, eventID, version, eventID, string(raw), eventID, version); err != nil {
		t.Fatal(err)
	}
	capture := &capturePublisher{}
	publisher := events.NewPublisher(capture)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	forms := registrationforms.NewService(db, enforcer, nil).WithPublisher(publisher)
	service := NewEventRegistrationsService(db, publisher, enforcer).WithForms(forms)
	req := &eventsv1.RegisterForEventRequest{EventId: eventID, UserId: userID}
	if _, err := service.RegisterForEvent(ctx, connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing answers bypassed form: %v", err)
	}
	req.FormSubmission = &pb.FormSubmission{FormVersionId: version, Answers: []*pb.RegistrationAnswer{{FieldId: "reason", Value: &pb.RegistrationAnswer_Text{Text: "Joining"}}}}
	if _, err := db.ExecContext(ctx, `UPDATE users SET email='' WHERE id=?`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RegisterForEvent(ctx, connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("incomplete identity contacts accepted: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE users SET email='map-forms@example.test' WHERE id=?`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE users SET username='',first_name=NULL,last_name=NULL WHERE id=?`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RegisterForEvent(ctx, connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("empty identity name accepted: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE users SET username='map-forms' WHERE id=?`, userID); err != nil {
		t.Fatal(err)
	}
	first, err := service.RegisterForEvent(ctx, connect.NewRequest(req))
	if err != nil {
		t.Fatal(err)
	}
	if first.Msg.Registration.Status != eventsv1.RegistrationStatus_REGISTRATION_STATUS_PENDING_APPROVAL {
		t.Fatal("manual submission registered immediately")
	}
	if capture.msg != nil {
		t.Fatal("pending submission published confirmed event")
	}
	if _, err := service.RegisterForEvent(ctx, connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("repeat pending bypass: %v", err)
	}
	ref := &pb.RegistrationRef{Source: pb.RegistrationSource_REGISTRATION_SOURCE_MAP, Id: first.Msg.Registration.Id}
	detail, err := forms.GetEventResponse(ctx, connect.NewRequest(&pb.GetEventResponseRequest{EventId: eventID, Registration: ref}))
	if err != nil {
		t.Fatal(err)
	}
	review := &pb.ReviewEventRegistrationRequest{EventId: eventID, Registration: ref, ResponseId: detail.Msg.Response.ResponseId, Action: pb.RegistrationReviewAction_REGISTRATION_REVIEW_ACTION_APPROVE}
	if _, err := forms.ReviewEventRegistration(ctx, connect.NewRequest(review)); connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("approval exceeded full capacity: %v", err)
	}
	cancelID := first.Msg.Registration.Id
	if _, err := service.CancelRegistration(ctx, connect.NewRequest(&eventsv1.CancelRegistrationRequest{RegistrationId: &cancelID})); err != nil {
		t.Fatal(err)
	}
	if capture.msg != nil {
		t.Fatal("pending cancellation published confirmed cancellation")
	}
	second, err := service.RegisterForEvent(ctx, connect.NewRequest(req))
	if err != nil {
		t.Fatal(err)
	}
	if second.Msg.Registration.Id != first.Msg.Registration.Id {
		t.Fatal("registration identity changed on reactivation")
	}
	detail, err = forms.GetEventResponse(ctx, connect.NewRequest(&pb.GetEventResponseRequest{EventId: eventID, Registration: ref}))
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Msg.Response.History) != 1 || detail.Msg.Response.History[0].Status != "cancelled" {
		t.Fatal("prior attempt history was lost")
	}
	if _, err := forms.ReviewEventRegistration(ctx, connect.NewRequest(review)); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("old response id reviewed: %v", err)
	}
	responseID, err := strconv.ParseInt(detail.Msg.Response.ResponseId, 10, 64)
	if err != nil || responseID <= 0 {
		t.Fatal("missing response id")
	}
	if _, err := db.ExecContext(context.Background(), `DELETE FROM external_registrations WHERE event_id=?`, eventID); err != nil {
		t.Fatal(err)
	}
	review.ResponseId = detail.Msg.Response.ResponseId
	if _, err := forms.ReviewEventRegistration(ctx, connect.NewRequest(review)); err != nil {
		t.Fatal(err)
	}
	if capture.topic != events.TopicRegistrationCreated {
		t.Fatal("approval did not publish confirmed registration")
	}
}
