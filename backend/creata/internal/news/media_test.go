package news

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The fixture server verifies the wire-level AWS SigV4 contract independently:
// known body/hash, credential scope, signed header set, and server-reconstructed
// canonical request. It does not reuse the client's signing helpers.
func TestS3SignedPUTAndPublicCDNURL(t *testing.T) {
	const access = "test-creata-access"
	const secret = "test-only-secret-creata"
	const body = "creata-s3-upload-fixture"
	const payloadSHA = "207682ec5b2cc645119623f0c0465543baa7c6dfae08f7c93466964d9dc022a2"
	const headers = "content-type;host;x-amz-content-sha256;x-amz-date"
	received := make(chan bool, 1)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		valid := true
		check := func(ok bool, message string) {
			if !ok {
				t.Error(message)
				valid = false
			}
		}
		raw, e := io.ReadAll(r.Body)
		check(e == nil && string(raw) == body, "upload body was changed")
		check(r.Method == http.MethodPut, "upload did not use PUT")
		check(r.URL.EscapedPath() == "/creata-media/images/fixture.png" && r.URL.RawQuery == "", "incorrect S3 bucket/object path")
		check(r.Header.Get("Content-Type") == "image/png", "image content type was not preserved")
		check(r.Header.Get("X-Amz-Content-Sha256") == payloadSHA, "known payload hash mismatch")
		timestamp := r.Header.Get("X-Amz-Date")
		parsed, parseErr := time.Parse("20060102T150405Z", timestamp)
		check(parseErr == nil && time.Since(parsed) < time.Minute && time.Since(parsed) > -time.Minute, "S3 signing timestamp invalid")
		date := parsed.UTC().Format("20060102")
		scope := date + "/us-east-1/s3/aws4_request"
		authorization := r.Header.Get("Authorization")
		prefix := "AWS4-HMAC-SHA256 Credential=" + access + "/" + scope + ", SignedHeaders=" + headers + ", Signature="
		check(strings.HasPrefix(authorization, prefix), "credential scope or signed-header set invalid")
		canonical := strings.Join([]string{"PUT", "/creata-media/images/fixture.png", "", "content-type:image/png\nhost:" + r.Host + "\nx-amz-content-sha256:" + payloadSHA + "\nx-amz-date:" + timestamp + "\n", headers, payloadSHA}, "\n")
		digest := sha256.Sum256([]byte(canonical))
		text := strings.Join([]string{"AWS4-HMAC-SHA256", timestamp, scope, hex.EncodeToString(digest[:])}, "\n")
		signingKey := []byte("AWS4" + secret)
		for _, part := range []string{date, "us-east-1", "s3", "aws4_request"} {
			mac := hmac.New(sha256.New, signingKey)
			_, _ = mac.Write([]byte(part))
			signingKey = mac.Sum(nil)
		}
		mac := hmac.New(sha256.New, signingKey)
		_, _ = mac.Write([]byte(text))
		expected := hex.EncodeToString(mac.Sum(nil))
		check(strings.TrimPrefix(authorization, prefix) == expected, "wire signature failed server-side verification")
		received <- valid
		if !valid {
			http.Error(w, "SignatureDoesNotMatch", 403)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer endpoint.Close()
	t.Setenv("CREATA_MEDIA_PUBLIC_URL", "https://cdn.creata.test/news/")
	api := &Server{Config: Config{S3Endpoint: endpoint.URL, S3Bucket: "creata-media", S3Region: "us-east-1", S3AccessKey: access, S3SecretKey: secret}, client: endpoint.Client()}
	request := httptest.NewRequest(http.MethodPost, "http://localhost/upload", nil).WithContext(context.Background())
	location, e := api.s3Put(request, "images/fixture.png", "image/png", []byte(body))
	if e != nil {
		t.Fatal(e)
	}
	if location != "https://cdn.creata.test/news/images/fixture.png" {
		t.Fatalf("wrong public CDN URL: %s", location)
	}
	if !<-received {
		t.Fatal("S3 fixture rejected request")
	}
}

func TestS3UploadFailureIsPropagated(t *testing.T) {
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "AccessDenied", http.StatusForbidden) }))
	defer endpoint.Close()
	api := &Server{Config: Config{S3Endpoint: endpoint.URL, S3Bucket: "creata-media", S3Region: "us-east-1", S3AccessKey: "test-access", S3SecretKey: "test-secret"}, client: endpoint.Client()}
	request := httptest.NewRequest(http.MethodPost, "http://localhost/upload", nil)
	location, e := api.s3Put(request, "images/fixture.png", "image/png", []byte("fixture"))
	if e == nil || !strings.Contains(e.Error(), "403") || location != "" {
		t.Fatalf("S3 failure became success: url=%q error=%v", location, e)
	}
}
