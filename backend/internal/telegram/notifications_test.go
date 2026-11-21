package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSendBulkRejectsMissingCredentialAndInvalidMessagesBeforeSending(t *testing.T) {
	var posted atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { posted.Add(1); w.WriteHeader(200) }))
	defer server.Close()
	valid := BulkRequest{Notifications: []Notification{{TelegramID: 101, Message: "Test"}}}
	if _, err := SendBulk(context.Background(), server.URL, "", valid); err == nil {
		t.Fatal("unconfigured credential accepted")
	}
	for _, message := range []string{"", strings.Repeat("x", 4097), strings.Repeat("😀", 2049)} {
		if _, err := SendBulk(context.Background(), server.URL, "test-secret", BulkRequest{Notifications: []Notification{{TelegramID: 101, Message: message}}}); err == nil {
			t.Fatal("unbounded message accepted")
		}
	}
	if _, err := SendBulk(context.Background(), server.URL, "test-secret", BulkRequest{Notifications: valid.Notifications, ParseMode: "unknown"}); err == nil {
		t.Fatal("invalid parse mode accepted")
	}
	if posted.Load() != 0 {
		t.Fatal("invalid notification sent")
	}
}
func TestSendBulkDoesNotForwardInternalSecretOnRedirect(t *testing.T) {
	var forwarded atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded.Add(1); w.WriteHeader(200) }))
	defer destination.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	if _, err := SendBulk(context.Background(), server.URL, "test-secret", BulkRequest{Notifications: []Notification{{TelegramID: 101, Message: "Test"}}}); err == nil {
		t.Fatal("redirect accepted")
	}
	if forwarded.Load() != 0 {
		t.Fatal("internal notification credential forwarded to redirect destination")
	}
}
