package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/studyverse/ems-backend/internal/buildinfo"
)

const readinessTimeout = 2 * time.Second

type Check struct {
	Name string
	Run  func(context.Context) error
}

type readinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func RegisterOperationalRoutes(router chi.Router, info buildinfo.Info, checks ...Check) {
	router.Get("/health", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("OK"))
	})

	router.Get("/health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})

	router.Get("/health/ready", func(writer http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithTimeout(request.Context(), readinessTimeout)
		defer cancel()

		statusCode := http.StatusOK
		response := readinessResponse{Status: "ready", Checks: make(map[string]string, len(checks))}
		for _, check := range checks {
			if err := check.Run(ctx); err != nil {
				response.Status = "not_ready"
				response.Checks[check.Name] = "error"
				statusCode = http.StatusServiceUnavailable
				continue
			}
			response.Checks[check.Name] = "ok"
		}

		writeJSON(writer, statusCode, response)
	})

	router.Get("/version", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, info)
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
