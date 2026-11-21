//go:build integration

package sundayaccount

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

type onboardingResult struct {
	Role        string     `json:"role"`
	RoleOther   string     `json:"roleOther"`
	Interests   []string   `json:"interests"`
	Goals       []string   `json:"goals"`
	Formats     []string   `json:"formats"`
	Source      string     `json:"source"`
	SourceOther string     `json:"sourceOther"`
	PromptedAt  *time.Time `json:"promptedAt"`
	CompletedAt *time.Time `json:"completedAt"`
}

func readOnboarding(t *testing.T, w *httptest.ResponseRecorder) onboardingResult {
	t.Helper()
	if w.Code != 200 {
		t.Fatalf("onboarding status %d: %s", w.Code, w.Body)
	}
	var result struct {
		Onboarding onboardingResult `json:"onboarding"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Onboarding.Interests == nil || result.Onboarding.Goals == nil || result.Onboarding.Formats == nil {
		t.Fatalf("onboarding arrays must never be null: %s", w.Body)
	}
	return result.Onboarding
}

func onboardingAnswers() map[string]any {
	return map[string]any{
		"role": "student", "roleOther": "", "interests": []string{"it_ai", "design", "it_ai"},
		"goals": []string{"practical_skills", "meet_people"}, "formats": []string{"workshops", "talks"},
		"source": "friends", "sourceOther": "",
	}
}

func TestOnboardingEmptyReadDoesNotWriteAndPromptIsIdempotent(t *testing.T) {
	h, m := accountFixture(t)
	cookie, csrf, accountID := loginAccount(t, h, m, "onboarding-empty@example.test")
	got := readOnboarding(t, callAccount(t, h, "GET", "onboarding", nil, cookie, ""))
	if got.Role != "" || got.Source != "" || got.RoleOther != "" || got.SourceOther != "" || len(got.Interests)+len(got.Goals)+len(got.Formats) != 0 || got.PromptedAt != nil || got.CompletedAt != nil {
		t.Fatalf("absent onboarding is not empty: %+v", got)
	}
	var count int
	if err := h.db.NewRaw(`SELECT count(*) FROM sunday_open_onboarding WHERE account_id=?`, accountID).Scan(context.Background(), &count); err != nil || count != 0 {
		t.Fatalf("empty GET wrote onboarding: count=%d error=%v", count, err)
	}
	first := readOnboarding(t, callAccount(t, h, "POST", "onboarding/prompted", nil, cookie, csrf))
	second := readOnboarding(t, callAccount(t, h, "POST", "onboarding/prompted", nil, cookie, csrf))
	if first.PromptedAt == nil || !reflect.DeepEqual(first, second) || first.CompletedAt != nil || first.Role != "" {
		t.Fatalf("prompt must be idempotent and preserve empty answers: first=%+v second=%+v", first, second)
	}
}

func TestOnboardingSaveReadEditAndAccountOwnership(t *testing.T) {
	h, m := accountFixture(t)
	cookie, csrf, accountID := loginAccount(t, h, m, "onboarding-save@example.test")
	prompt := readOnboarding(t, callAccount(t, h, "POST", "onboarding/prompted", nil, cookie, csrf))
	saved := readOnboarding(t, callAccount(t, h, "PATCH", "onboarding", onboardingAnswers(), cookie, csrf))
	if saved.Role != "student" || saved.Source != "friends" || !reflect.DeepEqual(saved.Interests, []string{"it_ai", "design"}) || saved.CompletedAt == nil || !reflect.DeepEqual(saved.PromptedAt, prompt.PromptedAt) {
		t.Fatalf("saved answers/timestamps invalid: %+v", saved)
	}
	read := readOnboarding(t, callAccount(t, h, "GET", "onboarding", nil, cookie, ""))
	if !reflect.DeepEqual(saved, read) {
		t.Fatalf("read differs from saved: %+v %+v", saved, read)
	}
	var firstUpdated time.Time
	if err := h.db.NewRaw(`SELECT updated_at FROM sunday_open_onboarding WHERE account_id=?`, accountID).Scan(context.Background(), &firstUpdated); err != nil {
		t.Fatal(err)
	}
	answers := onboardingAnswers()
	answers["role"] = "other"
	answers["roleOther"] = "  Моя роль  "
	answers["source"] = "other"
	answers["sourceOther"] = "  Другое место  "
	answers["interests"] = []string{"sports_games"}
	edited := readOnboarding(t, callAccount(t, h, "PATCH", "onboarding", answers, cookie, csrf))
	if edited.RoleOther != "Моя роль" || edited.SourceOther != "Другое место" || !reflect.DeepEqual(edited.CompletedAt, saved.CompletedAt) || !reflect.DeepEqual(edited.PromptedAt, saved.PromptedAt) {
		t.Fatalf("editing lost first timestamps or text normalization: %+v", edited)
	}
	var updated time.Time
	if err := h.db.NewRaw(`SELECT updated_at FROM sunday_open_onboarding WHERE account_id=?`, accountID).Scan(context.Background(), &updated); err != nil || !updated.After(firstUpdated) {
		t.Fatalf("edit did not update updated_at: before=%s after=%s error=%v", firstUpdated, updated, err)
	}
	prompted := readOnboarding(t, callAccount(t, h, "POST", "onboarding/prompted", nil, cookie, csrf))
	if !reflect.DeepEqual(edited, prompted) {
		t.Fatal("prompt overwrote completed answers")
	}
	otherCookie, otherCSRF, otherID := loginAccount(t, h, m, "onboarding-other@example.test")
	other := readOnboarding(t, callAccount(t, h, "GET", "onboarding", nil, otherCookie, ""))
	if other.CompletedAt != nil || other.Role != "" {
		t.Fatal("account read another participant's answers")
	}
	malicious := onboardingAnswers()
	malicious["accountId"] = accountID
	if w := callAccount(t, h, "PATCH", "onboarding", malicious, otherCookie, otherCSRF); w.Code != 400 {
		t.Fatalf("client-specified account accepted: %d %s", w.Code, w.Body)
	}
	readOnboarding(t, callAccount(t, h, "PATCH", "onboarding", onboardingAnswers(), otherCookie, otherCSRF))
	var count int
	if err := h.db.NewRaw(`SELECT count(*) FROM sunday_open_onboarding WHERE account_id IN (?,?) AND completed_at IS NOT NULL`, accountID, otherID).Scan(context.Background(), &count); err != nil || count != 2 {
		t.Fatalf("answers are not account scoped: count=%d error=%v", count, err)
	}
	read = readOnboarding(t, callAccount(t, h, "GET", "onboarding", nil, cookie, ""))
	if !reflect.DeepEqual(edited, read) {
		t.Fatal("other account's save changed first account")
	}
}

func TestGuestCanCompleteOnboardingWithoutVerifiedEmail(t *testing.T) {
	h, _ := accountFixture(t)
	cookie, csrf, account := startGuest(t, h, "onboarding-guest@example.test", "+77009991234")
	got := readOnboarding(t, callAccount(t, h, "PATCH", "onboarding", onboardingAnswers(), cookie, csrf))
	if !account.IsGuest || got.CompletedAt == nil || got.PromptedAt != nil {
		t.Fatalf("guest onboarding was not saved independently: %+v", got)
	}
}

func TestOnboardingRejectsInvalidOrPartialAnswersAtomically(t *testing.T) {
	h, m := accountFixture(t)
	cookie, csrf, accountID := loginAccount(t, h, m, "onboarding-invalid@example.test")
	for name, change := range map[string]func(map[string]any){
		"missing role":         func(v map[string]any) { delete(v, "role") },
		"missing source":       func(v map[string]any) { delete(v, "source") },
		"missing interests":    func(v map[string]any) { delete(v, "interests") },
		"missing goals":        func(v map[string]any) { delete(v, "goals") },
		"missing formats":      func(v map[string]any) { delete(v, "formats") },
		"invalid role":         func(v map[string]any) { v["role"] = "admin" },
		"invalid source":       func(v map[string]any) { v["source"] = "advert" },
		"invalid interest":     func(v map[string]any) { v["interests"] = []string{"invalid"} },
		"invalid goal":         func(v map[string]any) { v["goals"] = []string{"invalid"} },
		"invalid format":       func(v map[string]any) { v["formats"] = []string{"invalid"} },
		"empty interests":      func(v map[string]any) { v["interests"] = []string{} },
		"null goals":           func(v map[string]any) { v["goals"] = nil },
		"wrong array type":     func(v map[string]any) { v["formats"] = "talks" },
		"null array element":   func(v map[string]any) { v["formats"] = []any{nil} },
		"too many interests":   func(v map[string]any) { v["interests"] = strings.Split(strings.Repeat("it_ai,", 8)+"it_ai", ",") },
		"missing other role":   func(v map[string]any) { v["role"] = "other" },
		"blank other role":     func(v map[string]any) { v["role"] = "other"; v["roleOther"] = " \t\u2003 " },
		"long other role":      func(v map[string]any) { v["role"] = "other"; v["roleOther"] = strings.Repeat("Я", 201) },
		"missing other source": func(v map[string]any) { v["source"] = "other" },
		"long other source":    func(v map[string]any) { v["source"] = "other"; v["sourceOther"] = strings.Repeat("界", 201) },
		"nul other source":     func(v map[string]any) { v["source"] = "other"; v["sourceOther"] = "test\x00" },
	} {
		t.Run(name, func(t *testing.T) {
			answers := onboardingAnswers()
			change(answers)
			w := callAccount(t, h, "PATCH", "onboarding", answers, cookie, csrf)
			if w.Code != 400 {
				t.Fatalf("invalid answers accepted: %d %s", w.Code, w.Body)
			}
		})
	}
	var count int
	if err := h.db.NewRaw(`SELECT count(*) FROM sunday_open_onboarding WHERE account_id=?`, accountID).Scan(context.Background(), &count); err != nil || count != 0 {
		t.Fatalf("invalid answers persisted: count=%d error=%v", count, err)
	}
	answers := onboardingAnswers()
	answers["role"] = "other"
	answers["roleOther"] = "  " + strings.Repeat("Я", 200) + "  "
	answers["source"] = "other"
	answers["sourceOther"] = "  " + strings.Repeat("界", 200) + "  "
	saved := readOnboarding(t, callAccount(t, h, "PATCH", "onboarding", answers, cookie, csrf))
	if saved.RoleOther != strings.Repeat("Я", 200) || saved.SourceOther != strings.Repeat("界", 200) {
		t.Fatal("200 Unicode characters must be accepted and trimmed")
	}
	answers["goals"] = []string{}
	if w := callAccount(t, h, "PATCH", "onboarding", answers, cookie, csrf); w.Code != 400 {
		t.Fatalf("partial edit accepted: %d", w.Code)
	}
	if got := readOnboarding(t, callAccount(t, h, "GET", "onboarding", nil, cookie, "")); !reflect.DeepEqual(got, saved) {
		t.Fatal("invalid edit changed saved answers")
	}
}

func TestOnboardingRequiresAuthenticationOriginAndCSRF(t *testing.T) {
	h, m := accountFixture(t)
	for _, route := range []struct{ method, path string }{{"GET", "onboarding"}, {"PATCH", "onboarding"}, {"POST", "onboarding/prompted"}} {
		if w := callAccount(t, h, route.method, route.path, onboardingAnswers(), nil, ""); w.Code != 401 {
			t.Fatalf("anonymous %s %s: %d", route.method, route.path, w.Code)
		}
	}
	cookie, csrf, _ := loginAccount(t, h, m, "onboarding-csrf@example.test")
	for _, route := range []struct{ method, path string }{{"PATCH", "onboarding"}, {"POST", "onboarding/prompted"}} {
		if w := callAccount(t, h, route.method, route.path, onboardingAnswers(), cookie, ""); w.Code != 403 {
			t.Fatalf("missing CSRF %s: %d", route.path, w.Code)
		}
		if w := callAccount(t, h, route.method, route.path, onboardingAnswers(), cookie, csrf+"bad"); w.Code != 403 {
			t.Fatalf("incorrect CSRF %s: %d", route.path, w.Code)
		}
	}
	r := httptest.NewRequest("PATCH", prefix+"onboarding", strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://attacker.example")
	r.Header.Set("X-CSRF-Token", csrf)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("foreign Origin accepted: %d", w.Code)
	}
}

func TestOnboardingStorageFailuresAreUnavailable(t *testing.T) {
	h, m := accountFixture(t)
	cookie, csrf, _ := loginAccount(t, h, m, "onboarding-failure@example.test")
	if _, err := h.db.ExecContext(context.Background(), `ALTER TABLE sunday_open_onboarding DROP COLUMN role`); err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct{ method, path string }{{"GET", "onboarding"}, {"PATCH", "onboarding"}, {"POST", "onboarding/prompted"}} {
		if w := callAccount(t, h, route.method, route.path, onboardingAnswers(), cookie, csrf); w.Code != 503 || !strings.Contains(w.Body.String(), `"code":"unavailable"`) {
			t.Fatalf("storage failure %s %s: %d %s", route.method, route.path, w.Code, w.Body)
		}
	}
}
