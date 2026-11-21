package news

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type postizClient struct {
	key, baseURL, mode string
	http               *http.Client
	mu                 sync.Mutex
	catalog            []postizChannel
	expires            time.Time
}

type postizChannel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Supported bool   `json:"supported"`
	Reason    string `json:"reason,omitempty"`
}

func newPostizClient(c Config) (*postizClient, error) {
	p := &postizClient{key: strings.TrimSpace(c.PostizAPIKey), baseURL: strings.TrimRight(c.PostizAPIURL, "/"), mode: c.PostizMode}
	if p.mode == "" {
		p.mode = "draft"
	}
	if p.mode != "draft" && p.mode != "now" {
		return nil, errors.New("CREATA_POSTIZ_MODE must be draft or now")
	}
	if p.baseURL == "" {
		p.baseURL = "https://api.postiz.com/public/v1"
	}
	u, err := url.Parse(p.baseURL)
	local := c.Environment == "test" || c.Environment == "development"
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && local && isLoopback(u.Hostname()))) {
		return nil, errors.New("CREATA_POSTIZ_API_URL must be HTTPS (loopback HTTP is available in development and test)")
	}
	p.http = &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return p, nil
}

type postizError struct {
	status int
	retry  time.Duration
}

var errPostizNotConnected = errors.New("Could not connect to Postiz before sending the request")

func (e *postizError) Error() string { return fmt.Sprintf("Postiz returned HTTP %d", e.status) }

func (p *postizClient) request(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			return e
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", p.key)
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	var connected atomic.Bool
	trace := &httptrace.ClientTrace{GotConn: func(httptrace.GotConnInfo) { connected.Store(true) }}
	res, err := p.http.Do(req.WithContext(httptrace.WithClientTrace(req.Context(), trace)))
	if err != nil {
		if !connected.Load() {
			return errPostizNotConnected
		}
		return errors.New("Postiz connection interrupted")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		retry := time.Hour
		after := res.Header.Get("Retry-After")
		if n, e := strconv.Atoi(after); e == nil && n >= 0 && n <= 86400 {
			retry = max(time.Duration(n)*time.Second, time.Second)
		} else if t, e := http.ParseTime(after); e == nil {
			retry = max(time.Until(t), time.Second)
		}
		return &postizError{res.StatusCode, retry}
	}
	if err = json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(out); err != nil {
		return errors.New("Postiz returned an unreadable response")
	}
	return nil
}

func postizSupport(provider string) string {
	switch provider {
	case "x", "linkedin", "linkedin-page", "facebook", "threads", "bluesky", "mastodon", "vk", "nostr", "instagram", "instagram-standalone", "youtube":
		return ""
	case "telegram":
		return "Telegram delivery is managed by the Creata bot"
	default:
		return "This provider is not supported by Creata"
	}
}

func (s *Server) loadPostizChannels(ctx context.Context, allowLastKnown bool) ([]postizChannel, error) {
	p := s.postiz
	if p.key == "" {
		return []postizChannel{}, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Now().Before(p.expires) {
		return p.catalog, nil
	}
	var accounts []struct {
		ID, Name, Identifier string
		Disabled             bool
	}
	catalogID := hash(p.baseURL + "\x00" + p.key)
	if err := p.request(ctx, http.MethodGet, "/integrations", nil, &accounts); err != nil {
		// Editors can still publish to the site with their previously selected
		// destinations during a Postiz outage. Sending needs a fresh catalog.
		if allowLastKnown {
			var saved []postizChannel
			if e := s.Store.Pool.QueryRow(ctx, "SELECT data->'items' FROM creata_records WHERE kind='postiz_catalog' AND id=$1", catalogID).Scan(&saved); e == nil {
				return saved, nil
			}
		}
		return nil, err
	}
	catalog := make([]postizChannel, 0, len(accounts))
	for _, a := range accounts {
		if a.ID == "" || a.Identifier == "" {
			continue
		}
		reason := postizSupport(a.Identifier)
		if a.Disabled {
			reason = "This Postiz channel is disabled"
		}
		catalog = append(catalog, postizChannel{a.ID, a.Name, a.Identifier, reason == "", reason})
	}
	_, err := s.Store.Pool.Exec(ctx, `INSERT INTO creata_records(kind,id,data) VALUES('postiz_catalog',$1,$2)
		ON CONFLICT(kind,id) DO UPDATE SET data=excluded.data,updated_at=now()`, catalogID, Doc{"id": catalogID, "items": catalog})
	if err != nil {
		return nil, err
	}
	p.catalog, p.expires = catalog, time.Now().Add(5*time.Minute)
	return catalog, nil
}

func (s *Server) postizChannels(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	if !has(u["roles"], "editor") && !has(u["roles"], "chief_editor") && !has(u["roles"], "admin") {
		return errStatus(403, "Postiz channel access denied")
	}
	channels, e := s.loadPostizChannels(r.Context(), false)
	if e != nil {
		return errStatus(502, "Cannot read Postiz channels")
	}
	respond(w, 200, Doc{"configured": s.postiz.key != "", "mode": s.postiz.mode, "items": channels})
	return nil
}

type postizTarget struct {
	channel                   postizChannel
	lang, visibility, mediaID string
}

func postizLocale(lang string) bool { return lang == "ru" || lang == "en" || lang == "kk" }
func (s *Server) postizTargets(ctx context.Context, u Doc, articleID string, d Doc) ([]postizTarget, error) {
	raw, exists := d["postizTargets"]
	if !exists {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok || len(items) > 60 {
		return nil, errStatus(400, "postizTargets must be a list of at most 60 account and language pairs")
	}
	if len(items) == 0 {
		return nil, nil
	}
	if d["action"] != "publish" && d["action"] != "schedule" {
		return nil, errStatus(400, "Postiz targets require publication or scheduling")
	}
	a, e := s.Store.Get(ctx, "article", articleID)
	if e != nil {
		return nil, e
	}
	if !canPublish(u, a) {
		return nil, errStatus(403, "Publication denied")
	}
	if integer(a["approvedRevision"]) != integer(a["revision"]) || integer(d["expectedRevision"]) != integer(a["revision"]) {
		return nil, errStatus(409, "Current revision must be approved")
	}
	if s.postiz.key == "" {
		return nil, errStatus(409, "Postiz is not configured")
	}
	channels, e := s.loadPostizChannels(ctx, true)
	if e != nil {
		return nil, errStatus(502, "Cannot verify the selected Postiz channels")
	}
	byID := map[string]postizChannel{}
	for _, c := range channels {
		byID[c.ID] = c
	}
	targets := make([]postizTarget, 0, len(items))
	seen := map[[2]string]bool{}
	for _, raw := range items {
		item := object(raw)
		c, ok := byID[str(item["channelId"])]
		lang := str(item["locale"])
		if !ok || !c.Supported || !postizLocale(lang) {
			return nil, errStatus(400, "Select a supported Postiz account and language")
		}
		key := [2]string{c.ID, lang}
		if seen[key] {
			return nil, errStatus(400, "Duplicate Postiz account and language")
		}
		seen[key] = true
		visibility := ""
		if v, exists := item["visibility"]; exists {
			visibility = str(v)
			if c.Provider != "youtube" || (visibility != "private" && visibility != "public" && visibility != "unlisted") {
				return nil, errStatus(400, "Select a valid YouTube visibility")
			}
		} else if c.Provider == "youtube" {
			visibility = "private"
		}
		mediaID := ""
		if v, exists := item["mediaId"]; exists {
			mediaID = str(v)
			if mediaID == "" || len(mediaID) > 256 || (c.Provider != "instagram" && c.Provider != "instagram-standalone" && c.Provider != "youtube") {
				return nil, errStatus(400, "Select a library asset for a supported media channel")
			}
		}
		targets = append(targets, postizTarget{c, lang, visibility, mediaID})
	}
	return targets, nil
}

func (s *Server) postizPreview(w http.ResponseWriter, r *http.Request) error {
	u, e := s.require(r, true)
	if e != nil {
		return e
	}
	a, e := s.Store.Get(r.Context(), "article", r.PathValue("id"))
	if e != nil {
		return e
	}
	if !canPublish(u, a) {
		return errStatus(403, "Publication denied")
	}
	provider, lang := r.URL.Query().Get("provider"), r.URL.Query().Get("locale")
	if !postizLocale(lang) || postizSupport(provider) != "" {
		return errStatus(400, "Unsupported Postiz provider or locale")
	}
	content, link, e := s.postizText(a, provider, lang)
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"content": content, "articleUrl": link})
	return nil
}

func (s *Server) postizText(a Doc, provider, lang string) (string, string, error) {
	section := "articles"
	if materialFormat(a) == "graphic" {
		section = "charts"
	}
	link := strings.TrimRight(s.Config.PublicURL, "/") + "/" + lang + "/" + section + "/" + url.PathEscape(str(a["slug"])) + "?utm_content=creata-r" + strconv.Itoa(integer(a["revision"]))
	title := strings.Join(strings.Fields(str(object(a["title"])[lang])), " ")
	lead := strings.Join(strings.Fields(str(object(a["lead"])[lang])), " ")
	if lead == "" {
		lead = strings.Join(strings.Fields(str(object(a["summary"])[lang])), " ")
	}
	body := strings.TrimSpace(title + "\n\n" + lead)
	limit, linkCost := 2000, len([]rune(link))
	switch provider {
	case "x":
		limit, linkCost = 280, 23
	case "bluesky":
		limit = 300
	case "threads", "mastodon":
		limit = 500
	}
	budget := limit - linkCost - 2
	if budget < 2 {
		return "", "", errStatus(400, "The article link is too long for this Postiz channel")
	}
	var used int
	for offset, r := range body {
		weight := 1
		if provider == "x" && !(r <= 0x10ff || r >= 0x2000 && r <= 0x200d || r >= 0x2010 && r <= 0x201f || r >= 0x2032 && r <= 0x2037) {
			weight = 2
		}
		// Reserve space for a two-unit ellipsis when truncating an X post.
		if used+weight > budget-2 {
			body = strings.TrimSpace(body[:offset]) + "…"
			break
		}
		used += weight
	}
	return body + "\n\n" + link, link, nil
}
