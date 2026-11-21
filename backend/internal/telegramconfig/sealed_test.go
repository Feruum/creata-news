package telegramconfig

import (
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/crypto/nacl/box"
)

func testCredentials() Credentials {
	return Credentials{
		BotToken: "12345678:" + strings.Repeat("a", 35), BotUsername: "sundayopen_bot",
		WebhookSecret: strings.Repeat("w", 43),
	}
}

func TestSealedCredentialsRoundtripAndRandomization(t *testing.T) {
	secret, scope := strings.Repeat("s", 64), "https://map.webspace.cat"
	descriptor, err := PublicKey(secret, scope)
	if err != nil {
		t.Fatal(err)
	}
	want := testCredentials()
	first, err := Seal(descriptor, want)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Seal(descriptor, want)
	if err != nil || second == first || strings.Contains(first, want.BotToken) || strings.Contains(first, want.WebhookSecret) {
		t.Fatal("sealing must use fresh randomness and hide credentials")
	}
	got, err := Open(secret, scope, first)
	if err != nil || got != want {
		t.Fatalf("could not decrypt own credentials: %v", err)
	}
	other, err := PublicKey(secret, scope)
	if err != nil || other != descriptor {
		t.Fatal("persistent private configuration did not yield stable public key")
	}
}

func TestSealedCredentialsRejectWrongServerAndTampering(t *testing.T) {
	secret, scope := strings.Repeat("s", 64), "https://map.webspace.cat"
	descriptor, _ := PublicKey(secret, scope)
	sealed, _ := Seal(descriptor, testCredentials())
	for _, tc := range []struct{ name, secret, scope, sealed string }{
		{"different private key", strings.Repeat("t", 64), scope, sealed},
		{"development scope", secret, "https://map.dev.webspace.cat", sealed},
		{"unknown format", secret, scope, "other-v1:" + sealed},
		{"truncated", secret, scope, sealed[:len(sealed)-9]},
		{"invalid base64", secret, scope, "sundayopen-telegram-v1:!"},
		{"oversized", secret, scope, strings.Repeat("a", 4097)},
		{"whitespace", secret, scope, sealed + "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Open(tc.secret, tc.scope, tc.sealed)
			if err == nil || got != (Credentials{}) {
				t.Fatal("invalid configuration was accepted")
			}
			if strings.Contains(err.Error(), tc.sealed) || strings.Contains(err.Error(), testCredentials().BotToken) {
				t.Fatal("error exposed configuration")
			}
		})
	}
	encoded := strings.TrimPrefix(sealed, "sundayopen-telegram-v1:")
	ciphertext, _ := base64.RawURLEncoding.DecodeString(encoded)
	ciphertext[len(ciphertext)-1] ^= 1
	if _, err := Open(secret, scope, "sundayopen-telegram-v1:"+base64.RawURLEncoding.EncodeToString(ciphertext)); err == nil {
		t.Fatal("tampered ciphertext was accepted")
	}
}

func TestSealedCredentialsValidateBeforeSealing(t *testing.T) {
	descriptor, _ := PublicKey(strings.Repeat("s", 64), "production")
	for _, tc := range []struct {
		name string
		edit func(*Credentials)
	}{
		{"missing token", func(v *Credentials) { v.BotToken = "" }},
		{"malformed token", func(v *Credentials) { v.BotToken = "not-a-token" }},
		{"missing username", func(v *Credentials) { v.BotUsername = "" }},
		{"other username", func(v *Credentials) { v.BotUsername = "human_user" }},
		{"invalid username", func(v *Credentials) { v.BotUsername = "bad/name_bot" }},
		{"short webhook key", func(v *Credentials) { v.WebhookSecret = "short" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := testCredentials()
			tc.edit(&value)
			if sealed, err := Seal(descriptor, value); err == nil || sealed != "" {
				t.Fatal("invalid credentials were sealed")
			}
		})
	}
	for _, secret := range []string{"", strings.Repeat("s", 31)} {
		if _, err := PublicKey(secret, "production"); err == nil {
			t.Fatal("missing/short private master secret was accepted")
		}
	}
	if _, err := PublicKey(strings.Repeat("s", 64), ""); err == nil {
		t.Fatal("empty deployment scope was accepted")
	}
	bad := descriptor
	bad.KeyID = strings.Repeat("0", 64)
	if _, err := Seal(bad, testCredentials()); err == nil {
		t.Fatal("mismatched public key fingerprint accepted")
	}
}

func TestSetupKeyUsesDedicatedHKDFPurpose(t *testing.T) {
	secret, scope := strings.Repeat("s", 64), "production"
	descriptor, _ := PublicKey(secret, scope)
	derived, err := hkdf.Key(sha256.New, []byte(secret), nil, "studyverse/sundayopen/telegram-credentials/v1\x00"+scope, 32)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ecdh.X25519().NewPrivateKey(derived)
	if err != nil || descriptor.PublicKey != base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()) {
		t.Fatal("encryption key did not use its dedicated HKDF context")
	}
}

func TestOpenValidatesEncryptedEnvelopeFromOtherEncryptors(t *testing.T) {
	secret, scope := strings.Repeat("s", 64), "production"
	descriptor, _ := PublicKey(secret, scope)
	publicBytes, _ := base64.RawURLEncoding.DecodeString(descriptor.PublicKey)
	var public [32]byte
	copy(public[:], publicBytes)
	credentialsJSON, _ := json.Marshal(testCredentials())
	for _, plaintext := range []string{
		`{"version":2,"botToken":"x"}`,
		`{"version":1,"botToken":"x","botUsername":"sundayopen_bot","webhookSecret":"x"}`,
		`{"version":1,"extra":true,` + string(credentialsJSON[1:]),
		`{"version":1,` + string(credentialsJSON[1:]) + `{}`,
	} {
		ciphertext, err := box.SealAnonymous(nil, []byte(plaintext), &public, rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		sealed := "sundayopen-telegram-v1:" + base64.RawURLEncoding.EncodeToString(ciphertext)
		if got, err := Open(secret, scope, sealed); err == nil || got != (Credentials{}) {
			t.Fatal("malformed encrypted payload accepted")
		}
	}
}
