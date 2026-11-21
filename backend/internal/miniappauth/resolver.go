package miniappauth

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

var identityPattern = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)

type linkRecord struct {
	IdentityID string
	TelegramID int64
	LinkedAt   time.Time
}
type SQLResolver struct {
	db                *bun.DB
	adminURL          string
	publicURL         string
	secret            string
	sessionCookieName string
	client            *http.Client
	lookup            func(context.Context, int64) (*linkRecord, error)
}

type ResolverOption func(*SQLResolver)

func WithSessionCookieName(name string) ResolverOption {
	return func(r *SQLResolver) {
		if name != "" {
			r.sessionCookieName = name
		}
	}
}

func NewResolver(db *bun.DB, adminURL, publicURL, secret string, options ...ResolverOption) *SQLResolver {
	r := &SQLResolver{db: db, adminURL: strings.TrimRight(adminURL, "/"), publicURL: strings.TrimRight(publicURL, "/"), secret: secret, sessionCookieName: "ory_kratos_session", client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
	for _, option := range options {
		option(r)
	}
	r.lookup = r.databaseLink
	return r
}
func (r *SQLResolver) databaseLink(ctx context.Context, id int64) (*linkRecord, error) {
	if r.db == nil {
		return nil, ErrUnavailable
	}
	var users []models.User
	err := r.db.NewSelect().Model(&users).Column("kratos_id", "telegram_id", "telegram_linked_at").Where("telegram_id = ?", id).Where("telegram_linked_at IS NOT NULL").Limit(2).Scan(ctx)
	if err != nil {
		return nil, ErrUnavailable
	}
	if len(users) != 1 || users[0].KratosID == nil || users[0].TelegramID == nil || users[0].TelegramLinkedAt == nil {
		return nil, ErrNotLinked
	}
	return &linkRecord{IdentityID: *users[0].KratosID, TelegramID: *users[0].TelegramID, LinkedAt: *users[0].TelegramLinkedAt}, nil
}
func (r *SQLResolver) link(record *linkRecord) Link {
	mac := hmac.New(sha256.New, []byte(r.secret))
	_, _ = fmt.Fprintf(mac, "telegram-miniapp\n%s\n%d\n%s", record.IdentityID, record.TelegramID, record.LinkedAt.UTC().Format(time.RFC3339Nano))
	return Link{IdentityID: record.IdentityID, TelegramID: record.TelegramID, Subject: hex.EncodeToString(mac.Sum(nil))}
}

type kratosIdentity struct {
	ID             string          `json:"id"`
	State          string          `json:"state"`
	Traits         json.RawMessage `json:"traits"`
	MetadataAdmin  json.RawMessage `json:"metadata_admin"`
	MetadataPublic json.RawMessage `json:"metadata_public"`
	Credentials    map[string]struct {
		Config struct {
			Providers []struct {
				Provider string `json:"provider"`
				Subject  string `json:"subject"`
			} `json:"providers"`
		} `json:"config"`
	} `json:"credentials"`
}

func (r *SQLResolver) getIdentity(ctx context.Context, id string, credentials bool) (*kratosIdentity, error) {
	if !identityPattern.MatchString(id) {
		return nil, ErrNotLinked
	}
	target := r.adminURL + "/admin/identities/" + id
	if credentials {
		target += "?include_credential=oidc"
	}
	body, status, err := r.call(ctx, http.MethodGet, target, nil, "")
	if status == 404 {
		return nil, ErrNotLinked
	}
	if err != nil || status != 200 {
		return nil, ErrUnavailable
	}
	var identity kratosIdentity
	if json.Unmarshal(body, &identity) != nil || identity.ID != id {
		return nil, ErrUnavailable
	}
	if identity.State != "active" {
		return nil, ErrNotLinked
	}
	return &identity, nil
}
func (r *SQLResolver) Resolve(ctx context.Context, id int64) (*Link, error) {
	record, err := r.lookup(ctx, id)
	if err != nil {
		return nil, err
	}
	if record == nil || record.TelegramID != id || id <= 0 || record.LinkedAt.IsZero() || !identityPattern.MatchString(record.IdentityID) {
		return nil, ErrNotLinked
	}
	if _, err := r.getIdentity(ctx, record.IdentityID, false); err != nil {
		return nil, err
	}
	link := r.link(record)
	return &link, nil
}
func (r *SQLResolver) Check(ctx context.Context, expected Link) error {
	actual, err := r.Resolve(ctx, expected.TelegramID)
	if err != nil {
		return err
	}
	if *actual != expected {
		return ErrNotLinked
	}
	return nil
}
func (r *SQLResolver) EnsureCredential(ctx context.Context, link Link) error {
	if err := r.Check(ctx, link); err != nil {
		return err
	}
	identity, err := r.getIdentity(ctx, link.IdentityID, true)
	if err != nil {
		return err
	}
	for _, provider := range identity.Credentials["oidc"].Config.Providers {
		if provider.Provider == ProviderID && provider.Subject == link.Subject {
			return nil
		}
	}
	// Kratos 1.3 importOIDCCredentials APPENDS these entries to its latest confidential identity.
	// Send only the new provider: importing Microsoft again would duplicate its identifiers.
	// Omitting state/schema preserves the values Kratos reads during the update; no password or MFA is imported.
	body := map[string]any{"traits": identity.Traits, "credentials": map[string]any{"oidc": map[string]any{"config": map[string]any{"providers": []map[string]string{{"provider": ProviderID, "subject": link.Subject}}}}}}
	if identity.MetadataAdmin != nil {
		body["metadata_admin"] = identity.MetadataAdmin
	}
	if identity.MetadataPublic != nil {
		body["metadata_public"] = identity.MetadataPublic
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return ErrUnavailable
	}
	_, status, err := r.call(ctx, http.MethodPut, r.adminURL+"/admin/identities/"+link.IdentityID, encoded, "")
	if err != nil || status != 200 {
		return ErrUnavailable
	}
	return r.Check(ctx, link)
}
func (r *SQLResolver) CurrentIdentity(ctx context.Context, cookie string) (string, error) {
	probe := &http.Request{Header: http.Header{"Cookie": []string{cookie}}}
	if _, err := probe.Cookie(r.sessionCookieName); err != nil {
		return "", nil
	}
	body, status, err := r.call(ctx, http.MethodGet, r.publicURL+"/sessions/whoami", nil, cookie)
	if status == 401 {
		return "", nil
	}
	if err != nil || status != 200 {
		return "", ErrUnavailable
	}
	var session struct {
		Active   bool `json:"active"`
		Identity struct {
			ID string `json:"id"`
		} `json:"identity"`
	}
	if json.Unmarshal(body, &session) != nil {
		return "", ErrUnavailable
	}
	if !session.Active {
		return "", nil
	}
	if !identityPattern.MatchString(session.Identity.ID) {
		return "", ErrUnavailable
	}
	return session.Identity.ID, nil
}
func (r *SQLResolver) call(ctx context.Context, method, target string, body []byte, cookie string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
	if err != nil || len(payload) > 2*1024*1024 {
		return nil, resp.StatusCode, ErrUnavailable
	}
	return payload, resp.StatusCode, nil
}
