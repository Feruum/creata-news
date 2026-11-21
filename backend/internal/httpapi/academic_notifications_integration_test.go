//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/config"
	"github.com/studyverse/ems-backend/internal/telegram"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestAcademicNotificationsProductionFollowsCurrentTelegramLink(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	identityID := fmt.Sprintf("academic-%d", time.Now().UnixNano())
	var userID int32
	if err := db.NewRaw(
		"INSERT INTO users(kratos_id,username,email,password,telegram_id,telegram_linked_at) VALUES(?,?,?,?,?,CURRENT_TIMESTAMP) RETURNING id",
		identityID, identityID, identityID+"@test.local", "test", int64(101),
	).Scan(ctx, &userID); err != nil {
		t.Fatal(err)
	}

	posted := make(chan telegram.BulkRequest, 4)
	bot := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/notify/bulk" || request.Header.Get(auth.InternalSecretHeader) != academicTestSecret {
			t.Error("notification transport did not use the authenticated bot route")
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body telegram.BulkRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode bot notification: %v", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		posted <- body
		writeJSON(writer, http.StatusOK, academicSentResponse())
	}))
	defer bot.Close()

	handler := NewAcademicNotificationsHandler(db, &config.Config{
		TelegramBotURL:         bot.URL,
		InternalReminderSecret: academicTestSecret,
	})
	message := "<b>Private assignment</b> & quiz due in 24 hours"
	for _, telegramID := range []int64{101, 202} {
		if _, err := db.ExecContext(ctx, "UPDATE users SET telegram_id = ?, telegram_linked_at = CURRENT_TIMESTAMP WHERE id = ?", telegramID, userID); err != nil {
			t.Fatal(err)
		}
		response := academicRequest(handler, academicTestSecret, strings.NewReader(academicPayload(t, identityID, message)))
		assertAcademicResult(t, response, true, false)
		select {
		case notification := <-posted:
			if len(notification.Notifications) != 1 || notification.Notifications[0].TelegramID != telegramID || notification.Notifications[0].Message != message {
				t.Fatalf("notification did not target current link: %+v", notification)
			}
			if notification.ParseMode != "" || notification.ReplyMarkup != nil {
				t.Fatal("notification was not plain text")
			}
		default:
			t.Fatal("sent response was not acknowledged by the bot")
		}
	}

	if _, err := db.ExecContext(ctx, "UPDATE users SET telegram_id = NULL, telegram_linked_at = NULL WHERE id = ?", userID); err != nil {
		t.Fatal(err)
	}
	for _, identity := range []string{identityID, "missing-academic-identity"} {
		response := academicRequest(handler, academicTestSecret, strings.NewReader(academicPayload(t, identity, message)))
		assertAcademicResult(t, response, false, true)
	}
	select {
	case notification := <-posted:
		t.Fatalf("absent or unlinked user received a notification: %+v", notification)
	default:
	}
}
