package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSAllowsDistributedTraceContext(t *testing.T) {
	handler := corsMiddleware([]string{"http://localhost:5173"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("preflight reached application handler")
	}))
	request := httptest.NewRequest(http.MethodOptions, "/eventsv1.EventsService/ListEvents", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type,traceparent,tracestate")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("unexpected preflight: %d %v", response.Code, response.Header())
	}
	headers := strings.ToLower(response.Header().Get("Access-Control-Allow-Headers"))
	for _, header := range []string{"traceparent", "tracestate", "connect-protocol-version", "authorization"} {
		if !strings.Contains(headers, header) {
			t.Errorf("preflight does not permit %s", header)
		}
	}
}

func TestCORSDoesNotAllowUnknownOrigin(t *testing.T) {
	handler := corsMiddleware([]string{"http://localhost:5173"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "https://untrusted.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("untrusted origin allowed")
	}
}

func TestCORSAllowsScopedCheckInTicketHeaders(t *testing.T) {
	handler := corsMiddleware([]string{"https://sunday.example"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("preflight reached application handler")
	}))
	request := httptest.NewRequest(http.MethodOptions, "/api/sundayopen/account/check-in/info", nil)
	request.Header.Set("Origin", "https://sunday.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Access-Control-Request-Headers", "x-check-in-ticket,x-organizer-ticket")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	headers := strings.ToLower(response.Header().Get("Access-Control-Allow-Headers"))
	for _, header := range []string{"x-check-in-ticket", "x-organizer-ticket"} {
		if !strings.Contains(headers, header) {
			t.Errorf("preflight does not permit %s", header)
		}
	}
}
