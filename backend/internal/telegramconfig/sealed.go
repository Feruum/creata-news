// Package telegramconfig delivers a bot credential bundle to one deployment
// without putting plaintext credentials in its GitOps configuration.
package telegramconfig

import (
	"bytes"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"

	"golang.org/x/crypto/nacl/box"
)

const Algorithm = "nacl-box-sealed-v1"
const sealedPrefix = "sundayopen-telegram-v1:"
const keyPurpose = "studyverse/sundayopen/telegram-credentials/v1\x00"

var errInvalid = errors.New("Telegram credential configuration is invalid")
var tokenPattern = regexp.MustCompile(`^[0-9]{5,20}:[A-Za-z0-9_-]{20,128}$`)
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{2,29}[bB][oO][tT]$`)
var webhookPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,256}$`)

type Descriptor struct {
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"publicKey"`
	KeyID     string `json:"keyId"`
}

type Credentials struct {
	BotToken      string `json:"botToken"`
	BotUsername   string `json:"botUsername"`
	WebhookSecret string `json:"webhookSecret"`
}

type envelope struct {
	Version int `json:"version"`
	Credentials
}

func validCredentials(value Credentials) bool {
	return tokenPattern.MatchString(value.BotToken) && usernamePattern.MatchString(value.BotUsername) &&
		webhookPattern.MatchString(value.WebhookSecret)
}

func keyPair(secret, scope string) (public, private [32]byte, err error) {
	if len(secret) < 32 || scope == "" || len(scope) > 2048 {
		return public, private, errInvalid
	}
	derived, err := hkdf.Key(sha256.New, []byte(secret), nil, keyPurpose+scope, 32)
	if err != nil {
		return public, private, errInvalid
	}
	key, err := ecdh.X25519().NewPrivateKey(derived)
	if err != nil {
		return public, private, errInvalid
	}
	copy(private[:], derived)
	copy(public[:], key.PublicKey().Bytes())
	clear(derived)
	return public, private, nil
}

func descriptor(public [32]byte) Descriptor {
	fingerprint := sha256.Sum256(public[:])
	return Descriptor{
		Algorithm: Algorithm, PublicKey: base64.RawURLEncoding.EncodeToString(public[:]),
		KeyID: hex.EncodeToString(fingerprint[:]),
	}
}

// PublicKey derives a purpose- and deployment-scoped encryption key. The master
// auth secret must be the persistent, cryptographically random server key.
func PublicKey(secret, scope string) (Descriptor, error) {
	public, _, err := keyPair(secret, scope)
	if err != nil {
		return Descriptor{}, err
	}
	return descriptor(public), nil
}

// Seal keeps credentials confidential; permission to change the protected
// GitOps overlay, not possession of this public key, authorizes configuration.
func Seal(recipient Descriptor, value Credentials) (string, error) {
	if recipient.Algorithm != Algorithm || !validCredentials(value) {
		return "", errInvalid
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(recipient.PublicKey)
	if err != nil || len(raw) != 32 {
		return "", errInvalid
	}
	var public [32]byte
	copy(public[:], raw)
	if descriptor(public) != recipient {
		return "", errInvalid
	}
	plaintext, err := json.Marshal(envelope{Version: 1, Credentials: value})
	if err != nil {
		return "", errInvalid
	}
	// SealAnonymous generates the ephemeral key and nonce using the established
	// NaCl sealed-box construction: https://pkg.go.dev/golang.org/x/crypto/nacl/box.
	ciphertext, err := box.SealAnonymous(nil, plaintext, &public, rand.Reader)
	clear(plaintext)
	if err != nil {
		return "", errInvalid
	}
	return sealedPrefix + base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func Open(secret, scope, sealed string) (Credentials, error) {
	encoded, ok := strings.CutPrefix(sealed, sealedPrefix)
	if !ok || len(sealed) > 4096 || strings.ContainsAny(encoded, "\r\n\t ") {
		return Credentials{}, errInvalid
	}
	ciphertext, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil || len(ciphertext) < box.AnonymousOverhead || len(ciphertext) > 2048 {
		return Credentials{}, errInvalid
	}
	public, private, err := keyPair(secret, scope)
	if err != nil {
		return Credentials{}, errInvalid
	}
	plaintext, ok := box.OpenAnonymous(nil, ciphertext, &public, &private)
	clear(private[:])
	if !ok {
		return Credentials{}, errInvalid
	}
	defer clear(plaintext)
	var value envelope
	decoder := json.NewDecoder(bytes.NewReader(plaintext))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil || value.Version != 1 || !validCredentials(value.Credentials) {
		return Credentials{}, errInvalid
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Credentials{}, errInvalid
	}
	return value.Credentials, nil
}
