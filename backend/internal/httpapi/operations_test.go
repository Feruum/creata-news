package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/studyverse/ems-backend/internal/buildinfo"
)

func TestReadinessReturnsUnavailableWhenDependencyFails(t *testing.T) {
	router := chi.NewRouter()
	RegisterOperationalRoutes(router, buildinfo.Current("test", "test"),
		Check{Name: "postgres", Run: func(context.Context) error { return nil }},
		Check{Name: "redis", Run: func(context.Context) error { return errors.New("unavailable") }},
	)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestVersionReturnsBuildMetadata(t *testing.T) {
	router := chi.NewRouter()
	info := buildinfo.Info{Service: "ems-backend", Version: "1.2.3", Commit: "abc", BuiltAt: "now", Environment: "test"}
	RegisterOperationalRoutes(router, info)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/version", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Body.String(); got != "{\"service\":\"ems-backend\",\"version\":\"1.2.3\",\"commit\":\"abc\",\"built_at\":\"now\",\"environment\":\"test\"}\n" {
		t.Fatalf("response = %s", got)
	}
}
