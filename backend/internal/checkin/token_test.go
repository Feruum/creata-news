package checkin

import (
	"strings"
	"testing"
	"time"
)

func TestRotatingCodeRetainsOnlyCurrentAndPreviousWindow(t *testing.T) {
	now := time.Unix(1800000045, 0)
	for _, offset := range []int64{0, -1} {
		if !ValidCode("event-secret", Code("event-secret", now.Unix()/30+offset), now) {
			t.Fatalf("rejected accepted window %d", offset)
		}
	}
	for _, code := range []string{"", "forged", Code("other-secret", now.Unix()/30), Code("event-secret", now.Unix()/30-2), Code("event-secret", now.Unix()/30+1)} {
		if ValidCode("event-secret", code, now) {
			t.Fatalf("accepted invalid code %q", code)
		}
	}
}

func TestTicketBindsPurposeExpiryAndEvent(t *testing.T) {
	secret := strings.Repeat("s", 32)
	now := time.Unix(1800000045, 0)
	claim := Claim{Purpose: "guest", EventID: 42, EventToken: "event-token", Origin: "https://sunday.example", ExpiresAt: now.Add(15 * time.Minute).Unix()}
	token, err := Sign(secret, claim)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Verify(secret, token, "guest", now)
	if err != nil || got != claim {
		t.Fatalf("claim mismatch: %+v %v", got, err)
	}
	for _, test := range []struct {
		key, value, purpose string
		at                  time.Time
	}{
		{secret, token, "organizer", now},
		{strings.Repeat("x", 32), token, "guest", now},
		{secret, token + "x", "guest", now},
		{secret, token, "guest", now.Add(15 * time.Minute)},
		{"short", token, "guest", now},
	} {
		if _, err := Verify(test.key, test.value, test.purpose, test.at); err == nil {
			t.Fatalf("accepted invalid ticket: %+v", test)
		}
	}
}

func TestAllowedReturnOriginIsExact(t *testing.T) {
	allowed := "https://sunday.example,http://localhost:3003"
	for _, origin := range []string{"https://sunday.example", "http://localhost:3003"} {
		if !AllowedOrigin(allowed, origin) {
			t.Fatalf("rejected %q", origin)
		}
	}
	for _, origin := range []string{"https://sunday.example.evil.test", "https://sunday.example/path", "https://user@sunday.example", "javascript:alert(1)", "https://sunday.example#fragment", "null", ""} {
		if AllowedOrigin(allowed, origin) {
			t.Fatalf("accepted %q", origin)
		}
	}
}

func TestGuestTicketRequiresReturnOrigin(t *testing.T) {
	secret := strings.Repeat("s", 32)
	now := time.Now()
	token, err := Sign(secret, Claim{Purpose: "guest", EventID: 1, EventToken: "event-token", ExpiresAt: now.Add(TicketLifetime).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(secret, token, "guest", now); err == nil {
		t.Fatal("accepted a guest continuation without its return origin")
	}
}
