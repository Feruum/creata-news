package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidHash  = errors.New("invalid telegram authentication hash")
	ErrExpiredAuth  = errors.New("telegram authentication has expired")
	ErrMissingToken = errors.New("telegram bot token not configured")
)

type LoginData struct {
	ID        int64
	FirstName string
	LastName  string
	Username  string
	PhotoURL  string
	AuthDate  int64
	Hash      string
}

func VerifyLogin(data *LoginData, botToken string, maxAge time.Duration) error {
	if botToken == "" {
		return ErrMissingToken
	}

	if maxAge > 0 {
		age := time.Now().Unix() - data.AuthDate
		if age > int64(maxAge.Seconds()) {
			return ErrExpiredAuth
		}
	}

	params := make(map[string]string)
	params["id"] = fmt.Sprintf("%d", data.ID)
	params["first_name"] = data.FirstName
	params["auth_date"] = fmt.Sprintf("%d", data.AuthDate)

	if data.LastName != "" {
		params["last_name"] = data.LastName
	}
	if data.Username != "" {
		params["username"] = data.Username
	}
	if data.PhotoURL != "" {
		params["photo_url"] = data.PhotoURL
	}

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	dataCheckString := strings.Join(parts, "\n")

	secretKey := sha256.Sum256([]byte(botToken))

	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(h.Sum(nil))

	if expectedHash != data.Hash {
		return ErrInvalidHash
	}

	return nil
}
