//go:build integration

package users

import (
	"connectrpc.com/connect"
	"context"
	"encoding/json"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/telegram"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManualNotificationsUseCredentialAndBoundedBatches(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "INSERT INTO users(kratos_id,username,email,password) VALUES('audit-notify-actor','audit-notify-actor','audit-notify-actor@test.local','test')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO users(username,email,password,telegram_id) SELECT 'audit-recipient-'||i,'audit-recipient-'||i||'@test.local','test',i FROM generate_series(1,201) i"); err != nil {
		t.Fatal(err)
	}
	var ids []int32
	if err := db.NewSelect().Table("users").Column("id").Where("telegram_id IS NOT NULL").Order("id").Scan(ctx, &ids); err != nil {
		t.Fatal(err)
	}
	sizes := []int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Secret") != "audit-notify-secret" {
			t.Error("notification credential missing")
		}
		var request telegram.BulkRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if request.ParseMode != "HTML" {
			t.Error("parse mode omitted")
		}
		sizes = append(sizes, len(request.Notifications))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(telegram.BulkResponse{Success: true, Sent: int32(len(request.Notifications)), Total: int32(len(request.Notifications))})
	}))
	defer server.Close()
	graph := testsupport.AllowedPermissions(t)
	s := NewService(db, graph, authz.NewEnforcer(db, graph), nil, nil, server.URL, "", "audit-notify-secret")
	ctx = context.WithValue(ctx, auth.UserIDKey, "audit-notify-actor")
	mode := "HTML"
	response, err := s.SendTelegramNotification(ctx, connect.NewRequest(&usersv1.SendTelegramNotificationRequest{UserIds: ids, Message: "Test", ParseMode: &mode}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Sent != 201 || response.Msg.Failed != 0 || len(sizes) != 3 || sizes[0] != 100 || sizes[1] != 100 || sizes[2] != 1 {
		t.Fatalf("incorrect notification batching: %v %+v", sizes, response.Msg)
	}
}
