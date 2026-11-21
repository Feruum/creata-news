package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/config"
	"github.com/studyverse/ems-backend/internal/hermes"
)

func privateHermesEvents(cfg *config.Config, ops *hermes.Client, author hermes.EventActions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		credential, bearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if r.Method != http.MethodPost || !bearer || len(r.Header.Values("Authorization")) != 1 ||
			len(r.Header.Values("Origin")) != 0 || len(r.Header.Values("Cookie")) != 0 || !ops.ValidEventsCredential(credential) {
			writeHermesError(w, http.StatusForbidden)
			return
		}
		action := strings.TrimPrefix(r.URL.Path, "/internal/hermes/events/")
		switch action {
		case "rooms", "draft", "book", "publish":
		default:
			writeHermesError(w, http.StatusNotFound)
			return
		}
		contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || contentType != "application/json" {
			writeHermesError(w, http.StatusUnsupportedMediaType)
			return
		}
		limit := int64(48 << 10)
		if action == "draft" {
			limit = 3 << 20
		}
		body := http.MaxBytesReader(w, r.Body, limit)
		defer body.Close()
		decoder := json.NewDecoder(body)
		decoder.DisallowUnknownFields()
		var req hermes.EventRequest
		if err := decoder.Decode(&req); err != nil {
			writeHermesDecodeError(w, err)
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeHermesDecodeError(w, err)
			return
		}
		if cfg == nil || !hermes.MatchesEventOwner(req.OwnerID, cfg.HermesOwnerIDSHA256) {
			writeHermesError(w, http.StatusForbidden)
			return
		}
		if author == nil {
			writeHermesError(w, http.StatusServiceUnavailable)
			return
		}
		ctx := context.WithValue(r.Context(), auth.UserIDKey, req.OwnerID)
		result, err := author.Run(ctx, action, req)
		if err != nil {
			status := http.StatusServiceUnavailable
			switch connect.CodeOf(err) {
			case connect.CodeInvalidArgument:
				status = http.StatusBadRequest
			case connect.CodeUnauthenticated, connect.CodePermissionDenied:
				status = http.StatusForbidden
			case connect.CodeNotFound:
				status = http.StatusNotFound
			case connect.CodeFailedPrecondition, connect.CodeAlreadyExists, connect.CodeAborted:
				status = http.StatusConflict
			}
			writeJSON(w, status, map[string]string{"error": hermes.PublicEventMessage(err, http.StatusText(status))})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
}
