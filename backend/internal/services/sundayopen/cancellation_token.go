package sundayopen

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
)

type cancellationClaim struct {
	RegistrationID int32  `json:"registration"`
	EventID        int32  `json:"event"`
	Phone          string `json:"phone"`
	RegisteredAt   int64  `json:"registered_at"`
}

func cancellationConfigured(secret string) bool { return len(secret) >= 32 }

func signCancellationToken(secret string, registrationID, eventID int32, phone string, registeredAt time.Time) (string, error) {
	if !cancellationConfigured(secret) {
		return "", connect.NewError(connect.CodeUnavailable, errors.New("secure guest cancellation is not configured; contact the event organizer"))
	}
	claim, err := json.Marshal(cancellationClaim{registrationID, eventID, phone, registeredAt.UnixMicro()})
	if err != nil {
		return "", err
	}
	payload := "v1." + base64.RawURLEncoding.EncodeToString(claim)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func verifyCancellationToken(secret, token string, eventID int32, phone string) (cancellationClaim, error) {
	if !cancellationConfigured(secret) {
		return cancellationClaim{}, connect.NewError(connect.CodeUnavailable, errors.New("secure guest cancellation is not configured; contact the event organizer"))
	}
	denied := connect.NewError(connect.CodePermissionDenied, errors.New("a valid registration cancellation token is required"))
	parts := strings.Split(token, ".")
	if len(token) > 4096 || len(parts) != 3 || parts[0] != "v1" {
		return cancellationClaim{}, denied
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return cancellationClaim{}, denied
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return cancellationClaim{}, denied
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return cancellationClaim{}, denied
	}
	var claim cancellationClaim
	if err := json.Unmarshal(payload, &claim); err != nil || claim.RegistrationID <= 0 || claim.EventID != eventID || claim.Phone != phone {
		return cancellationClaim{}, denied
	}
	return claim, nil
}
