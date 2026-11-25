package news

import "testing"

func TestAdminThemePreference(t *testing.T) {
	user := Doc{"id": "legacy-user", "name": "Editor"}
	if safeUser(user)["adminTheme"] != "light" {
		t.Fatal("legacy users must default to light")
	}
	for _, theme := range []string{"cats", "dark", "light"} {
		if err := updatePreferences(user, Doc{"adminTheme": theme}); err != nil || user["adminTheme"] != theme {
			t.Fatalf("theme %q was not saved: %v", theme, err)
		}
	}
	for _, invalid := range []any{"pink", "", true, nil, 42} {
		if err := updatePreferences(user, Doc{"adminTheme": invalid}); err == nil {
			t.Fatalf("invalid theme accepted: %#v", invalid)
		}
	}
	if user["adminTheme"] != "light" {
		t.Fatal("invalid changes replaced a valid theme")
	}
	if err := updatePreferences(user, Doc{"name": "Another editor"}); err != nil || user["adminTheme"] != "light" {
		t.Fatal("unrelated preference changes reset the theme")
	}
}

func TestAdminThemeIsPersonalAndPersistsAcrossSessions(t *testing.T) {
	a := setupAPI(t)
	a.login("author")
	if theme := object(a.request("GET", "/auth/me", nil, 200, "")["user"])["adminTheme"]; theme != "light" {
		t.Fatalf("initial theme = %v", theme)
	}
	if out := a.request("PATCH", "/account", Doc{"adminTheme": "cats"}, 200, ""); out["adminTheme"] != "cats" {
		t.Fatal("theme was not saved", out)
	}
	a.request("PATCH", "/account", Doc{"adminTheme": "pink"}, 400, "")
	a.request("POST", "/auth/logout", nil, 200, "")
	a.login("editor")
	if theme := object(a.request("GET", "/auth/me", nil, 200, "")["user"])["adminTheme"]; theme != "light" {
		t.Fatalf("theme leaked to another account: %v", theme)
	}
	a.request("POST", "/auth/logout", nil, 200, "")
	a.login("author")
	if theme := object(a.request("GET", "/auth/me", nil, 200, "")["user"])["adminTheme"]; theme != "cats" {
		t.Fatalf("theme did not survive login: %v", theme)
	}
}
