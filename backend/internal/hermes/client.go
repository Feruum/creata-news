package hermes

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/studyverse/ems-backend/internal/config"
)

type Client struct {
	origin    string
	tokenFile string
	http      *http.Client
}

// The counter accepts only a scoped derivative. Its caller never forwards the
// more privileged ops administration credential to a backend HTTP endpoint.
func (c *Client) ValidCountCredential(value string) bool {
	return c.validCredential(value, "studyverse/hermes/user-count/v1")
}

func (c *Client) ValidEventsCredential(value string) bool {
	return c.validCredential(value, "studyverse/hermes/events/v1")
}

func (c *Client) validCredential(value, purpose string) bool {
	if c == nil || len(value) != 64 {
		return false
	}
	key, err := readToken(c.tokenFile)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(purpose))
	return hmac.Equal([]byte(value), []byte(hex.EncodeToString(mac.Sum(nil))))
}

func NewClient(cfg *config.Config) *Client {
	origin, err := cfg.HermesOpsOrigin()
	if err != nil || cfg == nil {
		return &Client{}
	}
	// Deliberately do not use the telemetry transport or environment HTTP proxies.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &Client{origin: origin, tokenFile: cfg.HermesOpsTokenFile, http: &http.Client{
		Timeout: 45 * time.Second, Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (c *Client) Call(ctx context.Context, operation string, payload Request) (Info, error) {
	var info Info
	if c == nil || c.origin == "" || c.http == nil || (operation != "info" && operation != "access" && operation != "settings") {
		return info, ErrUnavailable
	}
	if operation != "settings" {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	token, err := readToken(c.tokenFile)
	if err != nil {
		return info, ErrUnavailable
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return info, ErrUnavailable
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.origin+"/internal/hermes/"+operation, bytes.NewReader(body))
	if err != nil {
		return info, ErrUnavailable
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	if payload.CoverBase64 != "" {
		r.Header.Set("X-Hermes-Event-Cover", "1")
	}
	response, err := c.http.Do(r)
	if err != nil {
		return info, ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden {
		return info, ErrDenied
	}
	if response.StatusCode == http.StatusBadRequest {
		return info, ErrInvalid
	}
	if operation == "access" {
		if response.StatusCode != http.StatusNoContent {
			return info, ErrUnavailable
		}
		return info, nil
	}
	if response.StatusCode != http.StatusOK {
		return info, ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 64*1024+1))
	if err != nil || len(data) > 64*1024 || !validInfoJSON(data) || json.Unmarshal(data, &info) != nil {
		return Info{}, ErrUnavailable
	}
	return info, nil
}

func validInfoJSON(data []byte) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return false
	}
	for _, name := range []string{"claimed", "telegram_linked", "telegram_configured", "gitlab_configured", "chatgpt_connected", "ready"} {
		value := string(bytes.TrimSpace(fields[name]))
		if value != "true" && value != "false" {
			return false
		}
	}
	return true
}

func readToken(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", ErrUnavailable
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil || len(data) > 4096 {
		return "", ErrUnavailable
	}
	token := strings.TrimSpace(string(data))
	if len(token) < 32 || strings.ContainsAny(token, " \t\r\n") {
		return "", ErrUnavailable
	}
	return token, nil
}
