package storage

import (
	"bytes"
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTrustedProvidersUploadTheValidatedBytesAndMIME(t *testing.T) {
	data := raster(t, "png")
	for _, kind := range []string{"s3", "azure"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "HEAD" {
					w.WriteHeader(http.StatusOK)
					return
				}
				calls++
				if r.Method != "PUT" || r.URL.Path != "/images/events/7/u2/safe.png" {
					t.Errorf("unsafe upload target: %s %s", r.Method, r.URL.Path)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || !bytes.Equal(body, data) {
					t.Errorf("upload changed bytes: %v", err)
				}
				mime := r.Header.Get("Content-Type")
				if kind == "azure" {
					mime = r.Header.Get("x-ms-blob-content-type")
				}
				if mime != "image/png" {
					t.Errorf("invalid MIME: %s", mime)
				}
				if !strings.Contains(r.Header.Get("Cache-Control")+r.Header.Get("x-ms-blob-cache-control"), "immutable") {
					t.Error("missing immutable cache policy")
				}
				w.WriteHeader(http.StatusCreated)
			}))
			defer server.Close()
			var provider Provider
			if kind == "s3" {
				p, err := NewS3Provider(&S3Config{AccessKey: "test", SecretKey: "test-only", Bucket: "images", Endpoint: server.URL})
				if err != nil {
					t.Fatal(err)
				}
				provider = p
			} else {
				client, err := azblob.NewClientWithNoCredential(server.URL, nil)
				if err != nil {
					t.Fatal(err)
				}
				provider = &AzureProvider{client: client, containerName: "images"}
			}
			if err := provider.UploadObject(context.Background(), "events/7/u2/safe.png", "image/png", data); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("got %d writes, want one", calls)
			}
		})
	}
}
