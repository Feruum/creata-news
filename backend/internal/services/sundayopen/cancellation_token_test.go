package sundayopen

import (
	"strings"
	"testing"
	"time"
)

func TestCancellationTokenRejectsTamperingAndWrongPrincipal(t *testing.T) {
	secret := strings.Repeat("s", 32)
	token, err := signCancellationToken(secret, 9, 3, "+77001234567", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, token, secret, phone string
		event                      int32
		allowed                    bool
	}{
		{"valid", token, secret, "+77001234567", 3, true},
		{"tampered", token + "x", secret, "+77001234567", 3, false},
		{"other event", token, secret, "+77001234567", 4, false},
		{"other phone", token, secret, "+77009999999", 3, false},
		{"other key", token, strings.Repeat("t", 32), "+77001234567", 3, false},
		{"missing token", "", secret, "+77001234567", 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claim, err := verifyCancellationToken(tc.secret, tc.token, tc.event, tc.phone)
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v, want %v: %v", err == nil, tc.allowed, err)
			}
			if tc.allowed && claim.RegistrationID != 9 {
				t.Fatalf("registration=%d, want 9", claim.RegistrationID)
			}
		})
	}
}
