package sundayaccount

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/studyverse/ems-backend/gen/registrationforms/v1"
)

func TestRegistrationAcceptsProtoJSONAnswersBeforeProfileValidation(t *testing.T) {
	for _, answer := range []string{
		`{"fieldId":"name","text":"Visitor"}`,
		`{"fieldId":"age","number":21}`,
		`{"fieldId":"consent","checked":false}`,
		`{"fieldId":"track","selection":{"optionIds":["a"]}}`,
		`{"fieldId":"proof","files":{"uploadIds":["upload-a"]}}`,
	} {
		t.Run(answer, func(t *testing.T) {
			r := httptest.NewRequest("POST", prefix+"registrations", strings.NewReader(`{"eventId":1,"formSubmission":{"formVersionId":"v1","answers":[`+answer+`]}}`))
			w := httptest.NewRecorder()
			New(nil, Config{}, nil).register(w, r, &principal{})
			if w.Code != 409 || !strings.Contains(w.Body.String(), "Complete your name") {
				t.Fatalf("valid protobuf answer rejected before profile check: %d %s", w.Code, w.Body)
			}
		})
	}
}

func TestFormUploadResponsesArePrivateAndUseProtoJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeProto(w, &pb.CompleteFileUploadResponse{Attachment: &pb.RegistrationAttachment{SizeBytes: 42}})
	if w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("upload response can be cached: %q", w.Header().Get("Cache-Control"))
	}
	if !strings.Contains(w.Body.String(), `"sizeBytes":"42"`) {
		t.Fatalf("response lost protobuf int64 representation: %s", w.Body)
	}
}

func TestRegistrationRejectsMalformedProtoJSON(t *testing.T) {
	for _, submission := range []string{
		`{"unknown":true}`,
		`{"answers":[{"fieldId":"f","text":"a","checked":true}]}`,
		`{"answers":[{"fieldId":"f","number":"not-a-number"}]}`,
	} {
		r := httptest.NewRequest("POST", prefix+"registrations", strings.NewReader(`{"eventId":1,"formSubmission":`+submission+`}`))
		w := httptest.NewRecorder()
		New(nil, Config{}, nil).register(w, r, &principal{})
		if w.Code != 400 {
			t.Fatalf("malformed protobuf submission accepted: %d %s", w.Code, w.Body)
		}
	}
}

func TestRegistrationFormErrorsKeepRecoverableCodesAndMessages(t *testing.T) {
	for _, tc := range []struct {
		code    connect.Code
		status  int
		message string
	}{
		{connect.CodeInvalidArgument, 400, "A required answer is missing"},
		{connect.CodeFailedPrecondition, 409, "The registration form changed; reload the form"},
		{connect.CodeUnavailable, 503, "File uploads are not configured"},
		{connect.CodeResourceExhausted, 409, "Too many outstanding or recent uploads"},
		{connect.CodeNotFound, 404, "File not found"},
	} {
		w := httptest.NewRecorder()
		registrationError(w, connect.NewError(tc.code, errors.New(tc.message)))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.message) || !strings.Contains(w.Body.String(), tc.code.String()) {
			t.Fatalf("unrecoverable form error %s: %d %s", tc.code, w.Code, w.Body)
		}
	}
}

func TestFormUploadRoutesRequireSession(t *testing.T) {
	h := New(nil, Config{Origin: "https://sunday.example", Secret: strings.Repeat("s", 32)}, nil)
	for _, route := range []struct{ method, path string }{
		{"POST", "forms/uploads"},
		{"POST", "forms/uploads/test/complete"},
		{"GET", "forms/uploads/test/download"},
	} {
		r := httptest.NewRequest(route.method, prefix+route.path, strings.NewReader(`{}`))
		r.Header.Set("Origin", "https://sunday.example")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("anonymous upload access: %d %s", w.Code, w.Body)
		}
	}
}
