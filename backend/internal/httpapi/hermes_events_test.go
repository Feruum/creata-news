package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/config"
	"github.com/studyverse/ems-backend/internal/hermes"
)

type eventRecorder struct{ actor string }

func (h *eventRecorder) Run(ctx context.Context, _ string, _ hermes.EventRequest) (hermes.EventResult, error) {
	h.actor = auth.GetUserID(ctx)
	return hermes.EventResult{}, nil
}

func TestPrivateHermesEventBoundary(t *testing.T) {
	const owner = "00000000-0000-4000-8000-000000000001"
	key := strings.Repeat("s", 48)
	file := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(file, []byte(key), 0600); err != nil {
		t.Fatal(err)
	}
	derive := func(purpose string) string {
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(purpose))
		return hex.EncodeToString(mac.Sum(nil))
	}
	scoped := derive("studyverse/hermes/events/v1")
	hash := sha256.Sum256([]byte(owner))
	cfg := &config.Config{HermesOpsURL: "http://ops-diagnostics:8090", HermesOpsTokenFile: file, HermesOwnerIDSHA256: hex.EncodeToString(hash[:])}
	actions := &eventRecorder{}
	h := WithHermesRoutes(http.NotFoundHandler(), cfg, nil, nil, actions)
	for _, tc := range []struct {
		name, key, header, body string
		want                    int
	}{
		{"owner", scoped, "", `{"owner_id":"` + owner + `"}`, 200},
		{"missing pin", scoped, "", "{}", 403},
		{"untrusted actor", scoped, "", `{"owner_id":"00000000-0000-4000-8000-000000000002"}`, 403},
		{"raw administration key", key, "", "{}", 403},
		{"counter key", derive("studyverse/hermes/user-count/v1"), "", "{}", 403},
		{"origin", scoped, "Origin", "{}", 403},
		{"cookie", scoped, "Cookie", "{}", 403},
		{"extra field", scoped, "", `{"owner_id":"` + owner + `","organization_id":1}`, 400},
		{"arbitrary image url", scoped, "", `{"owner_id":"` + owner + `","image_url":"http://127.0.0.1"}`, 400},
		{"trailing object", scoped, "", `{"owner_id":"` + owner + `"} {}`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/internal/hermes/events/draft", strings.NewReader(tc.body))
			r.Header.Set("Authorization", "Bearer "+tc.key)
			r.Header.Set("Content-Type", "application/json")
			if tc.header != "" {
				r.Header.Set(tc.header, "")
			}
			actions.actor = ""
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.want || w.Header().Get("Cache-Control") != "no-store" ||
				(tc.want == 200 && actions.actor != owner) || (tc.want != 200 && actions.actor != "") {
				t.Fatalf("status=%d actor=%q body=%s", w.Code, actions.actor, w.Body)
			}
		})
	}
}
