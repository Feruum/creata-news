package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSundayPublicRPCAliasAllowlist(t *testing.T) {
	called := 0
	handler := sundayPublicRPCHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		if r.URL.Path != "/sundayopen.v1.SundayOpenService/"+r.Header.Get("Expected-Method") {
			t.Fatal("prefix not stripped")
		}
		w.WriteHeader(204)
	}))
	for _, method := range []string{"ListPublicEvents", "GetUpcomingEvents", "GetEventBySlug", "RegisterExternally", "CancelExternalRegistration"} {
		r := httptest.NewRequest("POST", "/api/sundayopen/rpc/sundayopen.v1.SundayOpenService/"+method, nil)
		r.Header.Set("Expected-Method", method)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 204 {
			t.Fatalf("%s: %d", method, w.Code)
		}
	}
	for _, path := range []string{"/api/sundayopen/rpc/sundayopen.v1.SundayOpenService/GetExternalRegistrations", "/api/sundayopen/rpc/events.v1.EventsService/ListEvents", "/api/sundayopen/rpc/ListPublicEvents"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", path, nil))
		if w.Code != 404 {
			t.Fatalf("private RPC allowed %s", path)
		}
	}
	if called != 5 {
		t.Fatalf("unexpected forwarded calls %d", called)
	}
}

func TestSundayFormsAliasExposesOnlyPublishedFormReading(t *testing.T) {
	const base = "/api/sundayopen/rpc/registrationforms.v1.RegistrationFormsService/"
	called := 0
	handler := sundayPublicRPCHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		if r.URL.Path != "/registrationforms.v1.RegistrationFormsService/GetPublishedForm" {
			t.Fatalf("incorrect forms procedure: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, method := range []string{"GetPublishedForm", "GetFormDraft", "SaveFormDraft", "PublishRegistrationForm", "ListEventResponses", "GetEventResponse", "ReviewEventRegistration", "ExportEventResponses", "BeginFileUpload", "GetFileDownload", "GetPublishedForm/extra"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, base+method, nil))
		want := http.StatusNotFound
		if method == "GetPublishedForm" {
			want = http.StatusNoContent
		}
		if w.Code != want {
			t.Fatalf("%s: got %d, want %d", method, w.Code, want)
		}
	}
	if called != 1 {
		t.Fatalf("forwarded %d procedures, want only the public read", called)
	}
}

func TestFormsResponsesArePrivateAndUncacheable(t *testing.T) {
	handler := uncachedFormsHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/registrationforms.v1.RegistrationFormsService/ListEventResponses", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("changed downstream response: %d", w.Code)
	}
	if w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("CDN-Cache-Control") != "no-store" {
		t.Fatalf("private responses may be cached: %v", w.Header())
	}
}
