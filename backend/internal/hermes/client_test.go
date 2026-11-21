package hermes

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/studyverse/ems-backend/internal/config"
)

func clientTokenFile(t *testing.T) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "ops-token")
	if err := os.WriteFile(file, []byte(strings.Repeat("s", 48)), 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestClientNeverForwardsPrivateClaimsToRedirect(t *testing.T) {
	var destinationCalls atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { destinationCalls.Add(1); w.WriteHeader(204) }))
	defer destination.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 307) }))
	defer redirect.Close()
	client := NewClient(&config.Config{HermesOpsURL: redirect.URL, HermesOpsTokenFile: clientTokenFile(t)})
	_, err := client.Call(context.Background(), "settings", Request{Identity: Identity{KratosID: "owner"}, Action: "configure_telegram", BotToken: "private-bot-token"})
	if !errors.Is(err, ErrUnavailable) || destinationCalls.Load() != 0 {
		t.Fatalf("redirect was followed: %v, calls=%d", err, destinationCalls.Load())
	}
}

func TestClientReloadsRotatedSecretAndRequiresExactAccessStatus(t *testing.T) {
	var expected atomic.Value
	expected.Store(strings.Repeat("s", 48))
	var status atomic.Int32
	status.Store(204)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+expected.Load().(string) {
			t.Error("secret rotation ignored")
		}
		w.WriteHeader(int(status.Load()))
	}))
	defer server.Close()
	file := clientTokenFile(t)
	client := NewClient(&config.Config{HermesOpsURL: server.URL, HermesOpsTokenFile: file})
	if _, err := client.Call(context.Background(), "access", Request{}); err != nil {
		t.Fatal(err)
	}
	expected.Store(strings.Repeat("n", 48))
	if err := os.WriteFile(file, []byte(expected.Load().(string)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Call(context.Background(), "access", Request{}); err != nil {
		t.Fatal(err)
	}
	status.Store(200)
	if _, err := client.Call(context.Background(), "access", Request{}); !errors.Is(err, ErrUnavailable) {
		t.Fatal("access accepted a non-204 status")
	}
}

func TestClientCancellationAndDisabledRuntimeAreUnavailable(t *testing.T) {
	for _, cfg := range []*config.Config{nil, {}, {HermesOpsURL: "http://ops.invalid"}} {
		if _, err := NewClient(cfg).Call(context.Background(), "info", Request{}); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("missing config = %v", err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	defer server.Close()
	client := NewClient(&config.Config{HermesOpsURL: server.URL, HermesOpsTokenFile: clientTokenFile(t)})
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, err := client.Call(ctx, "info", Request{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("timeout = %v", err)
	}
}
