package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ory "github.com/ory/kratos-client-go"
)

func TestSessionOutageIsDistinctFromSignedOut(t *testing.T) {
	for _, status := range []int{401, 403, 500, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			cfg := ory.NewConfiguration()
			cfg.Servers = ory.ServerConfigurations{{URL: server.URL}}
			handler := NewMiddleware(ory.NewAPIClient(cfg))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				want := status != 401 && status != 403
				if IsSessionValidationUnavailable(r.Context()) != want {
					t.Errorf("status %d: unavailable=%v", status, IsSessionValidationUnavailable(r.Context()))
				}
			}))
			r := httptest.NewRequest("POST", "/check-in", nil)
			r.Header.Set("Cookie", "ory_kratos_session=existing")
			handler.ServeHTTP(httptest.NewRecorder(), r)
		})
	}
}
