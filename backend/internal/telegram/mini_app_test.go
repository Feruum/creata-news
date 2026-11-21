package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

const testMiniAppToken = "123456:synthetic-bot-token"

func signedMiniApp(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(testMiniAppToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(parts, "\n")))
	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return values.Encode()
}

func testMiniAppValues(now time.Time) url.Values {
	return url.Values{"user": {`{"id":123456789,"first_name":"Synthetic"}`}, "auth_date": {fmt.Sprint(now.Unix())}, "query_id": {"synthetic-query"}}
}

func TestMiniAppVerifiesNumericIdentityWithoutUsername(t *testing.T) {
	now := time.Unix(1791190800, 0)
	data, err := VerifyInitData(signedMiniApp(testMiniAppValues(now)), testMiniAppToken, now)
	if err != nil {
		t.Fatal(err)
	}
	if data.UserID != 123456789 || len(data.Fingerprint) != 64 {
		t.Fatalf("invalid identity: %+v", data)
	}
}

func TestMiniAppRejectsTamperingAndWidgetSignature(t *testing.T) {
	now := time.Unix(1791190800, 0)
	valid := signedMiniApp(testMiniAppValues(now))
	values, _ := url.ParseQuery(valid)
	values.Set("user", `{"id":999999}`)
	if _, err := VerifyInitData(values.Encode(), testMiniAppToken, now); err == nil {
		t.Fatal("tampered identity accepted")
	}
	if _, err := VerifyInitData(valid, "another-bot-token", now); err == nil {
		t.Fatal("another bot accepted")
	}
	if _, err := VerifyInitData(valid, "", now); err == nil {
		t.Fatal("missing token accepted")
	}
	values.Set("hash", strings.Repeat("0", 64))
	if _, err := VerifyInitData(values.Encode(), testMiniAppToken, now); err == nil {
		t.Fatal("invalid hash accepted")
	}
}

func TestMiniAppRejectsExpiredFutureAndAmbiguousLaunches(t *testing.T) {
	now := time.Unix(1791190800, 0)
	for _, tc := range []struct{ name, field, value string }{
		{"expired", "auth_date", fmt.Sprint(now.Add(-301 * time.Second).Unix())},
		{"future", "auth_date", fmt.Sprint(now.Add(31 * time.Second).Unix())},
		{"missing date", "auth_date", ""},
		{"missing user", "user", ""},
		{"negative id", "user", `{"id":-1}`},
		{"fractional id", "user", `{"id":1.5}`},
		{"zero id", "user", `{"id":0}`},
		{"string id", "user", `{"id":"123456789"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := testMiniAppValues(now)
			values.Set(tc.field, tc.value)
			if _, err := VerifyInitData(signedMiniApp(values), testMiniAppToken, now); err == nil {
				t.Fatal("invalid launch accepted")
			}
		})
	}
	values := testMiniAppValues(now)
	values.Add("auth_date", fmt.Sprint(now.Unix()))
	if _, err := VerifyInitData(signedMiniApp(values), testMiniAppToken, now); err == nil {
		t.Fatal("duplicate field accepted")
	}
	if _, err := VerifyInitData(strings.Repeat("x", 16385), testMiniAppToken, now); err == nil {
		t.Fatal("oversized launch accepted")
	}
}

func TestMiniAppFingerprintIgnoresParameterOrderAndEncoding(t *testing.T) {
	now := time.Unix(1791190800, 0)
	valid := signedMiniApp(testMiniAppValues(now))
	a, err := VerifyInitData(valid, testMiniAppToken, now)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(valid, "&")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	variant := strings.ReplaceAll(strings.Join(parts, "&"), "%7B", "%7b")
	b, err := VerifyInitData(variant, testMiniAppToken, now)
	if err != nil {
		t.Fatal(err)
	}
	if a.Fingerprint != b.Fingerprint {
		t.Fatal("encoding can bypass replay detection")
	}
}
