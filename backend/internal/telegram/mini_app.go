package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type MiniAppIdentity struct {
	UserID      int64
	AuthDate    time.Time
	Fingerprint string
}

// VerifyInitData uses the Mini App HMAC protocol, which differs from Login Widget authentication.
// Only the verified numeric ID is an account identifier; display names are never used for lookup.
func VerifyInitData(raw, botToken string, now time.Time) (*MiniAppIdentity, error) {
	if botToken == "" {
		return nil, ErrMissingToken
	}
	if raw == "" || len(raw) > 16384 {
		return nil, ErrInvalidHash
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, ErrInvalidHash
	}
	keys := make([]string, 0, len(values))
	for key, entries := range values {
		if len(entries) != 1 || key == "" {
			return nil, ErrInvalidHash
		}
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	provided, err := hex.DecodeString(values.Get("hash"))
	if err != nil || len(provided) != sha256.Size {
		return nil, ErrInvalidHash
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	canonical := strings.Join(parts, "\n")
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	_, _ = mac.Write([]byte(canonical))
	if !hmac.Equal(mac.Sum(nil), provided) {
		return nil, ErrInvalidHash
	}
	timestamp, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || timestamp <= 0 {
		return nil, ErrInvalidHash
	}
	authDate := time.Unix(timestamp, 0)
	if authDate.Before(now.Add(-5*time.Minute)) || authDate.After(now.Add(30*time.Second)) {
		return nil, ErrExpiredAuth
	}
	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID <= 0 {
		return nil, ErrInvalidHash
	}
	fingerprint := sha256.Sum256([]byte(canonical))
	return &MiniAppIdentity{UserID: user.ID, AuthDate: authDate, Fingerprint: hex.EncodeToString(fingerprint[:])}, nil
}
