package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHermesPreflightPreservesPrivacyAndExplicitHeader(t *testing.T) {
	handler := corsMiddleware([]string{"https://map.webspace.cat"}, http.NotFoundHandler())
	for _, path := range []string{"/admin/hermes/info", "/admin/hermes/access", "/admin/hermes/settings"} {
		r := httptest.NewRequest(http.MethodOptions, path, nil)
		r.Header.Set("Origin", "https://map.webspace.cat")
		r.Header.Set("Access-Control-Request-Method", "POST")
		r.Header.Set("Access-Control-Request-Headers", "X-Studyverse-Hermes, Content-Type")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != "https://map.webspace.cat" {
			t.Fatalf("preflight = %d %v", w.Code, w.Header())
		}
		if !strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "X-Studyverse-Hermes") {
			t.Fatal("explicit Hermes header is not allowed")
		}
		for name, want := range map[string]string{"Cache-Control": "no-store", "Pragma": "no-cache", "Referrer-Policy": "no-referrer"} {
			if w.Header().Get(name) != want {
				t.Errorf("%s at %s = %q", name, path, w.Header().Get(name))
			}
		}
	}
}
