package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/config"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/telegram"
	"github.com/uptrace/bun"
)

const maxAcademicNotificationBodyBytes = 24 * 1024

type academicNotificationRequest struct {
	IdentityID string
	Message    string
}

type academicNotificationResponse struct {
	Sent    bool `json:"sent"`
	Skipped bool `json:"skipped"`
}

// NewAcademicNotificationsHandler resolves the current verified Telegram link
// for an internal identity and sends one plain-text academic notification.
func NewAcademicNotificationsHandler(db *bun.DB, cfg *config.Config) http.HandlerFunc {
	botURL, secret := cfg.TelegramBotURL, cfg.InternalReminderSecret
	return newAcademicNotificationsHandler(secret,
		func(ctx context.Context, identityID string) (*models.User, error) {
			user := new(models.User)
			err := db.NewSelect().Model(user).
				Column("telegram_id", "telegram_linked_at").
				Where("u.kratos_id = ?", identityID).
				Scan(ctx)
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			if err != nil {
				return nil, fmt.Errorf("resolve academic notification recipient: %w", err)
			}
			return user, nil
		},
		func(ctx context.Context, request telegram.BulkRequest) (telegram.BulkResponse, error) {
			return telegram.SendBulk(ctx, botURL, secret, request)
		},
	)
}

func newAcademicNotificationsHandler(
	secret string,
	lookup func(context.Context, string) (*models.User, error),
	send func(context.Context, telegram.BulkRequest) (telegram.BulkResponse, error),
) http.HandlerFunc {
	expectedHash := sha256.Sum256([]byte(secret))
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "private, no-store")
		providedHash := sha256.Sum256([]byte(request.Header.Get(auth.InternalSecretHeader)))
		if secret == "" || subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) != 1 {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "missing or invalid internal secret"})
			return
		}

		request.Body = http.MaxBytesReader(writer, request.Body, maxAcademicNotificationBodyBytes)
		data, err := io.ReadAll(request.Body)
		if err != nil {
			status := http.StatusBadRequest
			var limitError *http.MaxBytesError
			if errors.As(err, &limitError) {
				status = http.StatusRequestEntityTooLarge
			}
			writeJSON(writer, status, map[string]string{"error": "invalid notification payload"})
			return
		}
		payload, err := decodeAcademicNotification(data)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid notification payload"})
			return
		}

		user, err := lookup(request.Context(), payload.IdentityID)
		if err != nil {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"error": "notification delivery unavailable"})
			return
		}
		if user == nil || user.TelegramID == nil || *user.TelegramID <= 0 || user.TelegramLinkedAt == nil {
			writeJSON(writer, http.StatusOK, academicNotificationResponse{Skipped: true})
			return
		}

		result, err := send(request.Context(), telegram.BulkRequest{
			Notifications: []telegram.Notification{{TelegramID: *user.TelegramID, Message: payload.Message}},
		})
		if err != nil || !result.Success || result.Sent != 1 || result.Failed != 0 || result.Total != 1 {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"error": "notification delivery unavailable"})
			return
		}
		writeJSON(writer, http.StatusOK, academicNotificationResponse{Sent: true})
	}
}

func decodeAcademicNotification(data []byte) (academicNotificationRequest, error) {
	var payload academicNotificationRequest
	invalid := errors.New("invalid academic notification payload")
	if !utf8.Valid(data) {
		return payload, invalid
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return payload, invalid
	}
	fields := make(map[string]bool, 2)
	for decoder.More() {
		token, err = decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || fields[key] {
			return payload, invalid
		}
		fields[key] = true
		switch key {
		case "identityId":
			err = decoder.Decode(&payload.IdentityID)
		case "message":
			err = decoder.Decode(&payload.Message)
		default:
			return payload, invalid
		}
		if err != nil {
			return payload, invalid
		}
	}
	token, err = decoder.Token()
	if err != nil || token != json.Delim('}') {
		return payload, invalid
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return payload, invalid
	}
	if len(fields) != 2 || strings.TrimSpace(payload.IdentityID) == "" || len(payload.IdentityID) > 64 || !telegram.ValidMessage(payload.Message) {
		return payload, invalid
	}
	return payload, nil
}
