package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxBulkNotifications = 100
const MaxMessageLength = 4096
const maxNotificationBodyBytes = 2 * 1024 * 1024

type Notification struct {
	TelegramID int64  `json:"telegramId"`
	Message    string `json:"message"`
}
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}
type ReplyMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}
type BulkRequest struct {
	Notifications []Notification `json:"notifications"`
	ParseMode     string         `json:"parseMode,omitempty"`
	ReplyMarkup   *ReplyMarkup   `json:"replyMarkup,omitempty"`
}
type BulkResponse struct {
	Success bool  `json:"success"`
	Sent    int32 `json:"sent"`
	Failed  int32 `json:"failed"`
	Total   int32 `json:"total"`
}

func ValidMessage(message string) bool {
	if message == "" || !utf8.ValidString(message) {
		return false
	}
	units := 0
	for _, r := range message {
		units++
		if r > 0xffff {
			units++
		}
		if units > MaxMessageLength {
			return false
		}
	}
	return true
}
func ValidParseMode(mode string) bool {
	return mode == "" || mode == "HTML" || mode == "Markdown" || mode == "MarkdownV2"
}

// SendBulk authenticates every bounded request and preserves delivery errors so
// durable callers can retry. Redirects cannot forward the internal credential.
func SendBulk(ctx context.Context, baseURL, secret string, request BulkRequest) (BulkResponse, error) {
	var total BulkResponse
	if secret == "" {
		return total, errors.New("internal notification credential is not configured")
	}
	if !ValidParseMode(request.ParseMode) {
		return total, errors.New("unsupported notification parse mode")
	}
	for _, notification := range request.Notifications {
		if notification.TelegramID == 0 || notification.TelegramID > 9007199254740991 || notification.TelegramID < -9007199254740991 || !ValidMessage(notification.Message) {
			return total, errors.New("invalid notification recipient or message")
		}
	}
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	total.Success = true
	for start := 0; start < len(request.Notifications); start += MaxBulkNotifications {
		chunk := request
		chunk.Notifications = request.Notifications[start:min(start+MaxBulkNotifications, len(request.Notifications))]
		data, err := json.Marshal(chunk)
		if err != nil {
			return total, fmt.Errorf("marshal notifications: %w", err)
		}
		if len(data) > maxNotificationBodyBytes {
			return total, errors.New("notification request exceeds body limit")
		}
		httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/notify/bulk", bytes.NewReader(data))
		if err != nil {
			return total, fmt.Errorf("create notification request: %w", err)
		}
		httpRequest.Header.Set("Content-Type", "application/json")
		httpRequest.Header.Set("X-Internal-Secret", secret)
		response, err := client.Do(httpRequest)
		if err != nil {
			return total, fmt.Errorf("post notifications: %w", err)
		}
		var result BulkResponse
		decodeErr := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&result)
		_ = response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return total, fmt.Errorf("notification service status %d", response.StatusCode)
		}
		if decodeErr != nil {
			return total, fmt.Errorf("decode notification response: %w", decodeErr)
		}
		if result.Sent < 0 || result.Failed < 0 || int(result.Sent+result.Failed) != len(chunk.Notifications) {
			return total, errors.New("invalid notification delivery counts")
		}
		total.Sent += result.Sent
		total.Failed += result.Failed
		total.Total += int32(len(chunk.Notifications))
		if !result.Success || result.Failed != 0 {
			total.Success = false
			return total, errors.New("notification delivery failed")
		}
	}
	return total, nil
}
