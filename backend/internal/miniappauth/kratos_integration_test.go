package miniappauth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Opt-in against the pinned, official Kratos binary and a new disposable SQLite DB.
// No production account, Telegram API or shared development service is used.
func TestKratosBrowserLoginIntegration(t *testing.T) {
	binary := os.Getenv("KRATOS_TEST_BINARY")
	if binary == "" {
		t.Skip("set KRATOS_TEST_BINARY to a Kratos 1.3.1 SQLite binary")
	}
	publicURL := freeLocalURL(t)
	adminURL := freeLocalURL(t)
	h, _, now := fixture(t)
	*now = time.Now().UTC()
	schema, err := os.ReadFile("../../../../infra/vps/identity.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	mapper, err := os.ReadFile("../../../../infra/vps/telegram-miniapp-claims-mapper.jsonnet")
	if err != nil {
		t.Fatal(err)
	}
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test/schema" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(schema)
			return
		}
		if r.URL.Path == "/test/mapper" {
			_, _ = w.Write(mapper)
			return
		}
		if r.URL.Path == "/finish" {
			w.WriteHeader(200)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/auth/telegram/oidc/") {
			r.URL.Path = "/telegram-miniapp/" + strings.TrimPrefix(r.URL.Path, "/api/auth/telegram/oidc/")
		} else {
			r.URL.Path = "/telegram-miniapp/" + strings.TrimPrefix(r.URL.Path, "/api/auth/telegram/")
		}
		h.ServeHTTP(w, r)
	}))
	defer bridge.Close()
	h.config.Issuer = bridge.URL + "/api/auth/telegram/oidc"
	h.config.RedirectURI = publicURL + "/self-service/methods/oidc/callback/telegram-miniapp"
	h.origin = bridge.URL
	directory := t.TempDir()
	configPath := filepath.Join(directory, "kratos.json")
	public, _ := url.Parse(publicURL)
	admin, _ := url.Parse(adminURL)
	publicPort, _ := strconv.Atoi(public.Port())
	adminPort, _ := strconv.Atoi(admin.Port())
	config := map[string]any{
		"version": "v1.3.0",
		"dsn":     "sqlite://" + filepath.ToSlash(filepath.Join(directory, "kratos.sqlite")) + "?_fk=true&mode=rwc",
		"serve": map[string]any{
			"public": map[string]any{"host": "127.0.0.1", "port": publicPort, "base_url": publicURL + "/"},
			"admin":  map[string]any{"host": "127.0.0.1", "port": adminPort, "base_url": adminURL + "/"},
		},
		"secrets":  map[string]any{"cookie": []string{strings.Repeat("synthetic-cookie-", 3)}, "cipher": []string{strings.Repeat("x", 32)}},
		"log":      map[string]any{"level": "error", "leak_sensitive_values": false},
		"identity": map[string]any{"default_schema_id": "default", "schemas": []map[string]string{{"id": "default", "url": bridge.URL + "/test/schema"}}},
		"courier":  map[string]any{"smtp": map[string]string{"connection_uri": "smtp://127.0.0.1:9/?disable_starttls=true"}},
		"selfservice": map[string]any{
			"default_browser_return_url": bridge.URL + "/finish",
			"allowed_return_urls":        []string{bridge.URL},
			"methods": map[string]any{
				"password": map[string]any{"enabled": true},
				"oidc": map[string]any{"enabled": true, "config": map[string]any{"providers": []map[string]any{{
					"id": ProviderID, "provider": "generic", "client_id": ClientID, "client_secret": h.config.ClientSecret,
					"issuer_url": h.config.Issuer, "mapper_url": bridge.URL + "/test/mapper", "scope": []string{"openid"}, "claims_source": "id_token",
				}}}},
			},
			"flows": map[string]any{
				"login":        map[string]string{"ui_url": bridge.URL + "/login"},
				"registration": map[string]string{"ui_url": bridge.URL + "/register"},
				"error":        map[string]string{"ui_url": bridge.URL + "/error"},
			},
		},
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, configPath, encoded)
	migration := exec.Command(binary, "migrate", "sql", "-c", configPath, "-e", "--yes")
	migration.Env = cleanKratosEnvironment()
	if output, err := migration.CombinedOutput(); err != nil {
		t.Fatalf("Kratos migration: %v: %.2000s", err, output)
	}
	logs, err := os.Create(filepath.Join(directory, "kratos.log"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "serve", "-c", configPath, "--dev", "--sqa-opt-out")
	command.Env = cleanKratosEnvironment()
	command.Stdout, command.Stderr = logs, logs
	if err := command.Start(); err != nil {
		logs.Close()
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { _ = command.Wait(); close(exited) }()
	t.Cleanup(func() { _ = command.Process.Kill(); <-exited; _ = logs.Close() })
	ready := false
	for end := time.Now().Add(20 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		select {
		case <-exited:
			end = time.Now()
			continue
		default:
		}
		response, err := http.Get(adminURL + "/health/ready")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == 200 {
				ready = true
				break
			}
		}
	}
	if !ready {
		output, _ := os.ReadFile(logs.Name())
		t.Fatalf("Kratos not ready: %.2000s", output)
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	identityBody := `{"schema_id":"default","state":"active","traits":{"email":"synthetic@example.invalid","name":{"first":"Synthetic","last":"Student"}},"metadata_public":{"keep":"synthetic-public-metadata"},"metadata_admin":{"keep":"synthetic-admin-metadata"},"credentials":{"password":{"config":{"password":"synthetic-random-password-2026"}},"oidc":{"config":{"providers":[{"provider":"microsoft","subject":"synthetic-microsoft-id"}]}}}}`
	created := integrationJSON(t, client, "POST", adminURL+"/admin/identities", identityBody, "", 201)
	var identity struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(created, &identity) != nil || identity.ID == "" {
		t.Fatal("No fixture identity")
	}
	resolver := NewResolver(nil, adminURL, publicURL, h.config.ClientSecret)
	linked := true
	resolver.lookup = func(context.Context, int64) (*linkRecord, error) {
		if !linked {
			return nil, ErrNotLinked
		}
		return &linkRecord{IdentityID: identity.ID, TelegramID: 101, LinkedAt: now.Add(-time.Hour)}, nil
	}
	h.resolver = resolver
	payload, _ := json.Marshal(map[string]string{"initData": launch(*now)})
	integrationJSON(t, client, "POST", bridge.URL+"/api/auth/telegram/prepare", string(payload), bridge.URL, 200)
	confidential := integrationJSON(t, client, "GET", adminURL+"/admin/identities/"+identity.ID+"?include_credential=oidc&include_credential=password", "", "", 200)
	if !bytes.Contains(confidential, []byte("synthetic-microsoft-id")) || !bytes.Contains(confidential, []byte(ProviderID)) || !bytes.Contains(confidential, []byte("password")) {
		t.Fatal("Existing identity credentials were not preserved")
	}
	if !bytes.Contains(confidential, []byte("synthetic-admin-metadata")) {
		t.Fatal("Admin metadata lost")
	}
	flowBody := integrationJSON(t, client, "GET", publicURL+"/self-service/login/browser?return_to="+url.QueryEscape(bridge.URL+"/finish"), "", "", 200)
	var flow struct {
		UI struct {
			Action string `json:"action"`
			Nodes  []struct {
				Attributes struct {
					Name  string `json:"name"`
					Value any    `json:"value"`
				} `json:"attributes"`
			} `json:"nodes"`
		} `json:"ui"`
	}
	if json.Unmarshal(flowBody, &flow) != nil {
		t.Fatal("Invalid login flow")
	}
	csrf := ""
	for _, node := range flow.UI.Nodes {
		if node.Attributes.Name == "csrf_token" {
			csrf, _ = node.Attributes.Value.(string)
		}
	}
	if csrf == "" {
		t.Fatal("No CSRF token in browser flow")
	}
	form := url.Values{"csrf_token": {csrf}, "method": {"oidc"}, "provider": {ProviderID}}
	request, _ := http.NewRequest("POST", flow.UI.Action, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "text/html")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	for step := 0; step < 5 && (response.StatusCode == 303 || response.StatusCode == 302); step++ {
		next := response.Header.Get("Location")
		response.Body.Close()
		if !strings.HasPrefix(next, publicURL+"/") && !strings.HasPrefix(next, bridge.URL+"/") {
			t.Fatalf("Unexpected redirect: %s", next)
		}
		response, err = client.Get(next)
		if err != nil {
			t.Fatal(err)
		}
	}
	output, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("OIDC login failed: %d: %.2000s", response.StatusCode, output)
	}
	whoami := integrationJSON(t, client, "GET", publicURL+"/sessions/whoami", "", "", 200)
	var session struct {
		Active   bool `json:"active"`
		Identity struct {
			ID     string `json:"id"`
			Traits struct {
				Email string `json:"email"`
				Name  struct {
					First string `json:"first"`
					Last  string `json:"last"`
				} `json:"name"`
			} `json:"traits"`
			MetadataPublic map[string]string `json:"metadata_public"`
		} `json:"identity"`
	}
	if json.Unmarshal(whoami, &session) != nil || !session.Active || session.Identity.ID != identity.ID {
		t.Fatal("Mini App did not get the existing Kratos session")
	}
	if session.Identity.Traits.Email != "synthetic@example.invalid" || session.Identity.Traits.Name.First != "Synthetic" ||
		session.Identity.Traits.Name.Last != "Student" || session.Identity.MetadataPublic["keep"] != "synthetic-public-metadata" {
		t.Fatal("Existing account traits or public metadata changed during Telegram login")
	}
	linked = false
	integrationJSON(t, client, "POST", bridge.URL+"/api/auth/telegram/prepare", string(payload), bridge.URL, 404)
}

func cleanKratosEnvironment() []string {
	// Prevent local .env/production Kratos settings from entering the disposable test.
	var result []string
	for _, entry := range os.Environ() {
		name := strings.ToUpper(strings.SplitN(entry, "=", 2)[0])
		if name == "PATH" || name == "SYSTEMROOT" || name == "WINDIR" || name == "TEMP" || name == "TMP" || name == "HOME" || name == "USERPROFILE" {
			result = append(result, entry)
		}
	}
	return result
}
func freeLocalURL(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	return "http://" + address
}
func writeTestFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
}
func integrationJSON(t *testing.T, client *http.Client, method, target, body, origin string, status int) []byte {
	t.Helper()
	request, err := http.NewRequest(method, target, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Accept", "application/json")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	output, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != status {
		t.Fatalf("%s %s: %d, want %d: %.2000s", method, target, response.StatusCode, status, output)
	}
	return output
}
