package hermes

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

type EventRequest struct {
	OwnerID                   string `json:"owner_id"`
	RequestID                 string `json:"request_id"`
	Title                     string `json:"title,omitempty"`
	Description               string `json:"description,omitempty"`
	Format                    string `json:"format,omitempty"`
	Paid                      bool   `json:"paid,omitempty"`
	OnlineURL                 string `json:"online_url,omitempty"`
	StartTime                 string `json:"start_time,omitempty"`
	EndTime                   string `json:"end_time,omitempty"`
	Capacity                  *int32 `json:"capacity,omitempty"`
	Cover                     []byte `json:"cover_base64,omitempty"`
	EventID                   int32  `json:"event_id,omitempty"`
	RoomID                    string `json:"room_id,omitempty"`
	OwnerVerifiedAvailability bool   `json:"owner_verified_availability,omitempty"`
}

type EventRoom struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Capacity *int32 `json:"capacity"`
	Status   string `json:"status"`
}

type EventResult struct {
	EventID       int32  `json:"event_id,omitempty"`
	Status        string `json:"status,omitempty"`
	RoomID        string `json:"room_id,omitempty"`
	BookingStatus string `json:"booking_status,omitempty"`
	StartTime     string `json:"start_time,omitempty"`
	EndTime       string `json:"end_time,omitempty"`
	Capacity      *int32 `json:"capacity,omitempty"`
	// Pointer preserves the empty array on a successful room search.
	Rooms *[]EventRoom `json:"rooms,omitempty"`
}

type EventActions interface {
	Run(context.Context, string, EventRequest) (EventResult, error)
}

func MatchesEventOwner(id, fingerprint string) bool {
	parsed, err := uuid.Parse(id)
	if err != nil || parsed == uuid.Nil || parsed.String() != id || len(fingerprint) != 64 {
		return false
	}
	hash := sha256.Sum256([]byte(id))
	return hmac.Equal([]byte(fingerprint), []byte(hex.EncodeToString(hash[:])))
}

type publicEventMessage string

func (m publicEventMessage) Error() string { return string(m) }

func eventFailure(code connect.Code, message string) error {
	return connect.NewError(code, publicEventMessage(message))
}

// PublicEventMessage only exposes strings specifically chosen for the private client.
func PublicEventMessage(err error, fallback string) string {
	var safe publicEventMessage
	if errors.As(err, &safe) {
		return string(safe)
	}
	return fallback
}

func (r EventRequest) validate(action string) error {
	invalid := eventFailure(connect.CodeInvalidArgument, "Проверьте поля события, дату и время.")
	id, err := uuid.Parse(r.RequestID)
	if err != nil || id == uuid.Nil || id.String() != r.RequestID {
		return invalid
	}
	if r.Capacity != nil && *r.Capacity < 1 || len(r.RoomID) > 150 || r.RoomID != "" && strings.TrimSpace(r.RoomID) == "" {
		return invalid
	}
	hasDraft := r.Title != "" || r.Description != "" || r.Format != "" || r.Paid || r.OnlineURL != "" || len(r.Cover) > 0
	hasTime := r.StartTime != "" || r.EndTime != "" || r.Capacity != nil
	switch action {
	case "rooms", "draft":
		if action == "rooms" && r.EventID > 0 && !hasDraft && !r.OwnerVerifiedAvailability {
			return nil // The server reads the interval and capacity from the scoped event.
		}
		start, startErr := time.Parse(time.RFC3339, r.StartTime)
		end, endErr := time.Parse(time.RFC3339, r.EndTime)
		if startErr != nil || endErr != nil || !end.After(start) || end.Sub(start) > 93*24*time.Hour ||
			r.EventID != 0 || r.OwnerVerifiedAvailability {
			return invalid
		}
		if action == "rooms" {
			if hasDraft {
				return invalid
			}
			return nil
		}
		if r.RoomID != "" || strings.TrimSpace(r.Title) == "" || utf8.RuneCountInString(r.Title) > 255 ||
			strings.TrimSpace(r.Description) == "" || len(r.Description) > 32*1024 || strings.ContainsRune(r.Title+r.Description, '\x00') {
			return invalid
		}
		switch r.Format {
		case "", "offline":
			if r.OnlineURL != "" {
				return invalid
			}
		case "online", "hybrid":
			if r.OnlineURL != "" {
				u, err := url.ParseRequestURI(r.OnlineURL)
				if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || len(r.OnlineURL) > 2048 || strings.ContainsAny(r.OnlineURL, "\\ \t\r\n") {
					return invalid
				}
			}
		default:
			return invalid
		}
		if len(r.Cover) > 0 {
			mime := http.DetectContentType(r.Cover)
			if len(r.Cover) > 2<<20 || (mime != "image/png" && mime != "image/jpeg") {
				return eventFailure(connect.CodeInvalidArgument, "Для обложки подойдёт JPEG или PNG размером до 2 МиБ.")
			}
		}
	case "book":
		start, startErr := time.Parse(time.RFC3339, r.StartTime)
		end, endErr := time.Parse(time.RFC3339, r.EndTime)
		if r.EventID <= 0 || strings.TrimSpace(r.RoomID) == "" || hasDraft || startErr != nil || endErr != nil || !end.After(start) {
			return invalid
		}
	case "publish":
		if r.EventID <= 0 || r.RoomID != "" || hasTime || hasDraft || r.OwnerVerifiedAvailability {
			return invalid
		}
	default:
		return invalid
	}
	return nil
}
