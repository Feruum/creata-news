package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/telegram"
)

const academicTestSecret = "academic-test-secret"

type academicUnreadBody struct {
	reads int
}

func (body *academicUnreadBody) Read([]byte) (int, error) {
	body.reads++
	return 0, errors.New("request body must not be read")
}

func TestAcademicNotificationsRejectCredentialsBeforeReadingOrLookingUp(t *testing.T) {
	for _, test := range []struct {
		name       string
		configured string
		provided   string
	}{
		{name: "missing", configured: academicTestSecret},
		{name: "incorrect", configured: academicTestSecret, provided: "incorrect"},
		{name: "short prefix", configured: academicTestSecret, provided: "academic"},
		{name: "long suffix", configured: academicTestSecret, provided: academicTestSecret + "extra"},
		{name: "unconfigured", provided: academicTestSecret},
		{name: "both empty"},
	} {
		t.Run(test.name, func(t *testing.T) {
			lookupCalls, sendCalls := 0, 0
			handler := newAcademicNotificationsHandler(test.configured,
				func(context.Context, string) (*models.User, error) {
					lookupCalls++
					return academicLinkedUser(101), nil
				},
				func(context.Context, telegram.BulkRequest) (telegram.BulkResponse, error) {
					sendCalls++
					return academicSentResponse(), nil
				},
			)
			body := &academicUnreadBody{}
			response := academicRequest(handler, test.provided, body)
			assertAcademicStatus(t, response, http.StatusUnauthorized)
			if body.reads != 0 || lookupCalls != 0 || sendCalls != 0 {
				t.Fatalf("unauthorized request read body %d times, looked up %d times, sent %d times", body.reads, lookupCalls, sendCalls)
			}
		})
	}
}

func TestAcademicNotificationsRejectInvalidPayloadBeforeLookingUp(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "empty body"},
		{name: "malformed", body: `{"identityId":"private-identity","message":`},
		{name: "array", body: `[]`},
		{name: "null", body: `null`},
		{name: "trailing object", body: `{"identityId":"private-identity","message":"Test"}{}`},
		{name: "trailing garbage", body: `{"identityId":"private-identity","message":"Test"}garbage`},
		{name: "missing identity", body: `{"message":"Test"}`},
		{name: "null identity", body: `{"identityId":null,"message":"Test"}`},
		{name: "empty identity", body: `{"identityId":"","message":"Test"}`},
		{name: "blank identity", body: `{"identityId":"   ","message":"Test"}`},
		{name: "long identity", body: academicPayload(t, strings.Repeat("i", 65), "Test")},
		{name: "numeric identity", body: `{"identityId":101,"message":"Test"}`},
		{name: "missing message", body: `{"identityId":"private-identity"}`},
		{name: "null message", body: `{"identityId":"private-identity","message":null}`},
		{name: "empty message", body: `{"identityId":"private-identity","message":""}`},
		{name: "long message", body: academicPayload(t, "private-identity", strings.Repeat("x", 4097))},
		{name: "long UTF16 message", body: academicPayload(t, "private-identity", strings.Repeat("😀", 2049))},
		{name: "numeric message", body: `{"identityId":"private-identity","message":101}`},
		{name: "selected recipient", body: `{"identityId":"private-identity","message":"Test","telegramId":202}`},
		{name: "selected parse mode", body: `{"identityId":"private-identity","message":"Test","parseMode":"HTML"}`},
		{name: "unknown field", body: `{"identityId":"private-identity","message":"Test","extra":true}`},
		{name: "incorrect field case", body: `{"IdentityId":"private-identity","message":"Test"}`},
		{name: "duplicate field", body: `{"identityId":"first-identity","identityId":"private-identity","message":"Test"}`},
		{name: "invalid UTF8", body: "{\"identityId\":\"private-identity\",\"message\":\"\xff\"}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			lookupCalls, sendCalls := 0, 0
			handler := newAcademicNotificationsHandler(academicTestSecret,
				func(context.Context, string) (*models.User, error) {
					lookupCalls++
					return academicLinkedUser(101), nil
				},
				func(context.Context, telegram.BulkRequest) (telegram.BulkResponse, error) {
					sendCalls++
					return academicSentResponse(), nil
				},
			)
			response := academicRequest(handler, academicTestSecret, strings.NewReader(test.body))
			assertAcademicStatus(t, response, http.StatusBadRequest)
			if lookupCalls != 0 || sendCalls != 0 {
				t.Fatalf("invalid payload looked up %d times, sent %d times", lookupCalls, sendCalls)
			}
		})
	}
}

func TestAcademicNotificationsBoundEntireBodyBeforeLookingUp(t *testing.T) {
	lookupCalls := 0
	handler := newAcademicNotificationsHandler(academicTestSecret,
		func(context.Context, string) (*models.User, error) {
			lookupCalls++
			return academicLinkedUser(101), nil
		},
		func(context.Context, telegram.BulkRequest) (telegram.BulkResponse, error) {
			t.Fatal("oversized body sent a notification")
			return telegram.BulkResponse{}, nil
		},
	)
	for _, body := range []string{
		strings.Repeat(" ", 24*1024) + academicPayload(t, "private-identity", "Test"),
		academicPayload(t, "private-identity", "Test") + strings.Repeat(" ", 24*1024),
	} {
		response := academicRequest(handler, academicTestSecret, strings.NewReader(body))
		assertAcademicStatus(t, response, http.StatusRequestEntityTooLarge)
	}
	if lookupCalls != 0 {
		t.Fatalf("oversized body looked up %d times", lookupCalls)
	}
}

func TestAcademicNotificationsSkipAbsentOrUnlinkedRecipients(t *testing.T) {
	zero, negative := int64(0), int64(-101)
	linkedAt := time.Now()
	for _, test := range []struct {
		name string
		user *models.User
	}{
		{name: "absent"},
		{name: "unlinked", user: &models.User{}},
		{name: "missing ID", user: &models.User{TelegramLinkedAt: &linkedAt}},
		{name: "zero ID", user: &models.User{TelegramID: &zero, TelegramLinkedAt: &linkedAt}},
		{name: "negative ID", user: &models.User{TelegramID: &negative, TelegramLinkedAt: &linkedAt}},
		{name: "unverified ID", user: &models.User{TelegramID: academicLinkedUser(101).TelegramID}},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := newAcademicNotificationsHandler(academicTestSecret,
				func(context.Context, string) (*models.User, error) { return test.user, nil },
				func(context.Context, telegram.BulkRequest) (telegram.BulkResponse, error) {
					t.Fatal("unlinked recipient sent a notification")
					return telegram.BulkResponse{}, nil
				},
			)
			response := academicRequest(handler, academicTestSecret, strings.NewReader(academicPayload(t, "private-identity", "Test")))
			assertAcademicResult(t, response, false, true)
		})
	}
}

func TestAcademicNotificationsResolveCurrentLinkAndSendOnePlainTextMessage(t *testing.T) {
	currentUser := academicLinkedUser(101)
	lookupCalls, sendCalls := 0, 0
	message := "<b>Assignment</b> & quiz due in 24 hours"
	handler := newAcademicNotificationsHandler(academicTestSecret,
		func(_ context.Context, identityID string) (*models.User, error) {
			lookupCalls++
			if identityID != "private-identity" {
				t.Fatalf("looked up unexpected identity %q", identityID)
			}
			return currentUser, nil
		},
		func(_ context.Context, request telegram.BulkRequest) (telegram.BulkResponse, error) {
			sendCalls++
			if len(request.Notifications) != 1 || request.Notifications[0].TelegramID != *currentUser.TelegramID || request.Notifications[0].Message != message {
				t.Fatalf("incorrect notification: %+v", request)
			}
			if request.ParseMode != "" || request.ReplyMarkup != nil {
				t.Fatalf("message was not plain text: %+v", request)
			}
			return academicSentResponse(), nil
		},
	)
	for _, recipient := range []int64{101, 202} {
		currentUser = academicLinkedUser(recipient)
		response := academicRequest(handler, academicTestSecret, strings.NewReader(academicPayload(t, "private-identity", message)))
		assertAcademicResult(t, response, true, false)
	}
	currentUser = &models.User{}
	response := academicRequest(handler, academicTestSecret, strings.NewReader(academicPayload(t, "private-identity", message)))
	assertAcademicResult(t, response, false, true)
	if lookupCalls != 3 || sendCalls != 2 {
		t.Fatalf("lookup calls = %d, send calls = %d", lookupCalls, sendCalls)
	}
}

func TestAcademicNotificationsAcceptUTF16MessageLimits(t *testing.T) {
	handler := newAcademicNotificationsHandler(academicTestSecret,
		func(context.Context, string) (*models.User, error) { return academicLinkedUser(101), nil },
		func(context.Context, telegram.BulkRequest) (telegram.BulkResponse, error) {
			return academicSentResponse(), nil
		},
	)
	for _, message := range []string{strings.Repeat("x", 4096), strings.Repeat("😀", 2048)} {
		response := academicRequest(handler, academicTestSecret, strings.NewReader(academicPayload(t, strings.Repeat("i", 64), message)))
		assertAcademicResult(t, response, true, false)
	}
}

func TestAcademicNotificationsLookupFailureDoesNotExposePrivateDetails(t *testing.T) {
	handler := newAcademicNotificationsHandler(academicTestSecret,
		func(context.Context, string) (*models.User, error) {
			return nil, errors.New("private-identity failed on telegram_id 101")
		},
		func(context.Context, telegram.BulkRequest) (telegram.BulkResponse, error) {
			t.Fatal("lookup failure sent a notification")
			return telegram.BulkResponse{}, nil
		},
	)
	response := academicRequest(handler, academicTestSecret, strings.NewReader(academicPayload(t, "private-identity", "Private message")))
	assertAcademicStatus(t, response, http.StatusServiceUnavailable)
	for _, private := range []string{"private-identity", "telegram_id", "101", "Private message"} {
		if strings.Contains(response.Body.String(), private) {
			t.Fatalf("response exposed private details: %s", response.Body.String())
		}
	}
}

func TestAcademicNotificationsRequireOneAcknowledgedDelivery(t *testing.T) {
	for _, test := range []struct {
		name   string
		result telegram.BulkResponse
		err    error
	}{
		{name: "upstream error", err: errors.New("private-identity transport failure")},
		{name: "empty acknowledgement"},
		{name: "no sent message", result: telegram.BulkResponse{Success: true, Total: 1}},
		{name: "failed", result: telegram.BulkResponse{Success: true, Sent: 1, Failed: 1, Total: 1}},
		{name: "unsuccessful", result: telegram.BulkResponse{Sent: 1, Total: 1}},
		{name: "too many sent", result: telegram.BulkResponse{Success: true, Sent: 2, Total: 2}},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := newAcademicNotificationsHandler(academicTestSecret,
				func(context.Context, string) (*models.User, error) { return academicLinkedUser(101), nil },
				func(context.Context, telegram.BulkRequest) (telegram.BulkResponse, error) {
					return test.result, test.err
				},
			)
			response := academicRequest(handler, academicTestSecret, strings.NewReader(academicPayload(t, "private-identity", "Test")))
			assertAcademicStatus(t, response, http.StatusServiceUnavailable)
			if strings.Contains(response.Body.String(), "private-identity") {
				t.Fatal("upstream error exposed private details")
			}
		})
	}
}

func TestAcademicNotificationsInternalCredentialNeedsNoUserSession(t *testing.T) {
	router := chi.NewRouter()
	router.Post("/internal/academic-notifications", newAcademicNotificationsHandler(academicTestSecret,
		func(context.Context, string) (*models.User, error) { return academicLinkedUser(101), nil },
		func(context.Context, telegram.BulkRequest) (telegram.BulkResponse, error) {
			return academicSentResponse(), nil
		},
	))
	// A nil Kratos client proves the session middleware does not contact Kratos.
	handler := auth.NewMiddleware(nil)(router)
	response := academicRequest(handler, academicTestSecret, strings.NewReader(academicPayload(t, "private-identity", "Test")))
	assertAcademicResult(t, response, true, false)
}

func academicLinkedUser(telegramID int64) *models.User {
	linkedAt := time.Now()
	return &models.User{TelegramID: &telegramID, TelegramLinkedAt: &linkedAt}
}

func academicSentResponse() telegram.BulkResponse {
	return telegram.BulkResponse{Success: true, Sent: 1, Total: 1}
}

func academicPayload(t *testing.T, identityID, message string) string {
	t.Helper()
	encoded, err := json.Marshal(map[string]string{"identityId": identityID, "message": message})
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func academicRequest(handler http.Handler, secret string, body io.Reader) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/internal/academic-notifications", body)
	request.Header.Set(auth.InternalSecretHeader, secret)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertAcademicStatus(t *testing.T, response *httptest.ResponseRecorder, status int) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q", response.Header().Get("Content-Type"))
	}
}

func assertAcademicResult(t *testing.T, response *httptest.ResponseRecorder, sent, skipped bool) {
	t.Helper()
	assertAcademicStatus(t, response, http.StatusOK)
	var result map[string]bool
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result["sent"] != sent || result["skipped"] != skipped {
		t.Fatalf("result = %s, want sent=%t skipped=%t", response.Body.String(), sent, skipped)
	}
}
