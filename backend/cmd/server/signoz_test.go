package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSigNozPreflightPreservesPrivacyHeaders(t *testing.T) {
	handler := corsMiddleware([]string{"https://admin.example.test"}, http.NotFoundHandler())
	for _, path := range []string{"/admin/signoz/info", "/admin/signoz/access"} {
		request := httptest.NewRequest(http.MethodOptions, path, nil)
		request.Header.Set("Origin", "https://admin.example.test")
		request.Header.Set("Access-Control-Request-Method", "GET")
		request.Header.Set("Access-Control-Request-Headers", "X-Studyverse-SigNoz")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("preflight status = %d", response.Code)
		}
		for name, expected := range map[string]string{
			"Cache-Control": "no-store", "Pragma": "no-cache", "Referrer-Policy": "no-referrer",
		} {
			if response.Header().Get(name) != expected {
				t.Errorf("%s at %s = %q, want %q", name, path, response.Header().Get(name), expected)
			}
		}
	}
}
