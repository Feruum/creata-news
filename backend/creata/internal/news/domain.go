package news

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Doc map[string]any

func str(v any) string { s, _ := v.(string); return s }
func integer(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}
func number(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}
func boolean(v any) bool { b, _ := v.(bool); return b }
func object(v any) Doc {
	switch d := v.(type) {
	case Doc:
		return d
	case map[string]any:
		return Doc(d)
	}
	return Doc{}
}
func array(v any) []any {
	switch a := v.(type) {
	case []any:
		return a
	case []string:
		r := make([]any, len(a))
		for i, s := range a {
			r[i] = s
		}
		return r
	}
	return []any{}
}
func has(v any, s string) bool {
	for _, x := range array(v) {
		if str(x) == s {
			return true
		}
	}
	return false
}
func id() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func hash(s string) string { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }
func now() string          { return time.Now().UTC().Format(time.RFC3339Nano) }
func localize(v any, locale string) string {
	d := object(v)
	if s := str(d[locale]); s != "" {
		return s
	}
	return str(d["ru"])
}
func localized(s string) Doc { return Doc{"ru": s, "kk": s, "en": s} }
func locale(s string) string {
	if s == "kk" || s == "en" {
		return s
	}
	return "ru"
}
func staff(u Doc) bool {
	return has(u["roles"], "author") || has(u["roles"], "editor") || has(u["roles"], "chief_editor") || has(u["roles"], "admin")
}
func canEdit(u, a Doc) bool {
	return has(u["roles"], "chief_editor") || (has(u["roles"], "editor") && has(u["verticals"], str(a["categoryId"]))) || (has(u["roles"], "author") && str(u["id"]) == str(a["authorId"]))
}
func canPublish(u, a Doc) bool {
	return has(u["roles"], "chief_editor") || (str(a["layer"]) != "eco" && has(u["roles"], "editor") && has(u["verticals"], str(a["categoryId"])))
}
func revise(a Doc) {
	delete(a, "activeAIJobId")
	a["revision"] = integer(a["revision"]) + 1
	delete(a, "approvedRevision")
	a["status"] = "review"
	delete(a, "scheduledAt")
	a["updatedAt"] = now()
}
func parsePublicURL(raw string) (*url.URL, error) {
	u, e := url.Parse(strings.TrimSpace(raw))
	if e != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return nil, errors.New("invalid public URL")
	}
	return u, nil
}
func canonicalURL(raw string) (string, error) {
	u, e := parsePublicURL(raw)
	if e != nil {
		return "", e
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	q := u.Query()
	for k := range q {
		if strings.HasPrefix(strings.ToLower(k), "utm_") || k == "fbclid" || k == "gclid" {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
func publishable(a Doc) error {
	if e := validateArticleExtras(a, true); e != nil {
		return e
	}
	if e := validateGraphicPublication(a); e != nil {
		return e
	}
	for _, f := range []string{"title", "body"} {
		for _, l := range []string{"ru", "kk", "en"} {
			if strings.TrimSpace(str(object(a[f])[l])) == "" {
				return fmt.Errorf("%s is required in %s", f, l)
			}
		}
	}
	for _, s := range array(a["sources"]) {
		if _, e := canonicalURL(str(object(s)["url"])); e != nil {
			return e
		}
	}
	m := object(a["media"])
	if str(m["url"]) != "" && (str(m["rights"]) == "" || str(m["credit"]) == "") {
		return errors.New("media credit and rights are required")
	}
	return nil
}
func slugify(s string) string {
	var b strings.Builder
	last := false
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r > 127 {
			b.WriteRune(r)
			last = false
		} else if !last {
			b.WriteByte('-')
			last = true
		}
	}
	v := strings.Trim(b.String(), "-")
	if len([]rune(v)) > 80 {
		v = string([]rune(v)[:80])
	}
	if v == "" {
		v = "article"
	}
	return v
}
func clone(d Doc) Doc { b, _ := json.Marshal(d); var c Doc; _ = json.Unmarshal(b, &c); return c }
func publicArticle(d Doc) Doc {
	p := clone(d)
	for _, k := range []string{"leaseToken", "publicationNote", "actorId", "note", "researchBrief", "activeAIJobId", "expectedAIJobId", "storySlug"} {
		delete(p, k)
	}
	p["reviewNotes"] = []any{}
	projectArticleExtras(p)
	return p
}
func parseTime(v any) time.Time                  { t, _ := time.Parse(time.RFC3339Nano, str(v)); return t }
func errStatus(status int, message string) error { return &apiError{status, message} }

type apiError struct {
	status  int
	message string
}

func (e *apiError) Error() string { return e.message }
func required(d Doc, key string) error {
	if strings.TrimSpace(str(d[key])) == "" {
		return fmt.Errorf("%s is required", key)
	}
	return nil
}
func parseInt(s string, fallback int) int {
	n, e := strconv.Atoi(s)
	if e != nil {
		return fallback
	}
	return n
}
