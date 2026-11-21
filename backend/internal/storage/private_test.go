package storage

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/studyverse/ems-backend/internal/config"
)

func TestPrivateSigningMatchesStorageProxyCanonicalPath(t *testing.T) {
	client := s3.New(s3.Options{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("testkey", "testsecret", ""), BaseEndpoint: aws.String("https://map.example.test"), UsePathStyle: true})
	base := &S3Provider{client: client, presignClient: s3.NewPresignClient(client), bucket: "ems-registration-files"}
	p := &privateS3{S3Provider: base, proxyPrefix: "/storage"}
	got, err := p.SignUpload(context.Background(), "staging/test-id", "application/pdf", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := base.GeneratePresignedUploadURL(context.Background(), "staging/test-id", "application/pdf", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	external, _ := url.Parse(got.UploadURL)
	canonical, _ := url.Parse(signed.UploadURL)
	if external.Host != canonical.Host || strings.TrimPrefix(external.Path, "/storage") != canonical.Path || external.Query().Get("X-Amz-Signature") != canonical.Query().Get("X-Amz-Signature") {
		t.Fatal("proxy would change the signed host, path, or signature")
	}
	if got.PublicURL != "" {
		t.Fatal("private adapter leaked public URL")
	}
	if _, err := p.SignUpload(context.Background(), "final/test-id", "application/pdf", time.Minute); err == nil {
		t.Fatal("final key signed for PUT")
	}
}
func TestBoundedPrivateReadAndDownloadDisposition(t *testing.T) {
	if _, err := boundedRead(io.NopCloser(strings.NewReader("12345")), 4); err == nil {
		t.Fatal("oversized body accepted")
	}
	if data, err := boundedRead(io.NopCloser(strings.NewReader("1234")), 4); err != nil || string(data) != "1234" {
		t.Fatalf("bounded read: %q %v", data, err)
	}
	if !strings.HasPrefix(disposition("résumé.pdf"), "attachment;") {
		t.Fatal("unsafe disposition")
	}
}

func TestPrivateAzureDeleteIsIdempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-ms-error-code", "BlobNotFound")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><Error><Code>BlobNotFound</Code><Message>Missing</Message></Error>`))
	}))
	defer server.Close()
	client, err := azblob.NewClientWithNoCredential(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := &privateAzure{&AzureProvider{client: client, containerName: "private"}}
	if err = p.Delete(context.Background(), "staging/missing"); err != nil {
		t.Fatal("cleanup must tolerate already absent staging objects", err)
	}
}

func TestPrivateS3PolicyCorsAndImmutableWrite(t *testing.T) {
	for _, publicPolicy := range []bool{false, true} {
		t.Run(map[bool]string{false: "private", true: "public policy rejected"}[publicPolicy], func(t *testing.T) {
			var saved []byte
			var cors string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodHead:
					w.WriteHeader(200)
				case r.URL.Query().Has("policy"):
					if publicPolicy {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"Statement":[{"Effect":"Allow","Principal":"*"}]}`))
						return
					}
					w.Header().Set("Content-Type", "application/xml")
					w.WriteHeader(404)
					_, _ = w.Write([]byte(`<Error><Code>NoSuchBucketPolicy</Code><Message>Missing</Message></Error>`))
				case r.URL.Query().Has("cors"):
					body, _ := io.ReadAll(r.Body)
					cors = string(body)
					w.WriteHeader(200)
				case r.Method == http.MethodPut:
					if r.Header.Get("If-None-Match") != "*" {
						t.Error("final write lacks exclusive create")
					}
					if r.Header.Get("Cache-Control") != "private, no-store" || !strings.HasPrefix(r.Header.Get("Content-Disposition"), "attachment;") {
						t.Error("missing private download headers")
					}
					if saved != nil {
						w.Header().Set("Content-Type", "application/xml")
						w.WriteHeader(412)
						_, _ = w.Write([]byte(`<Error><Code>PreconditionFailed</Code></Error>`))
						return
					}
					saved, _ = io.ReadAll(r.Body)
					w.WriteHeader(200)
				default:
					w.WriteHeader(400)
				}
			}))
			defer server.Close()
			cfg := &config.Config{StorageProvider: "s3", StorageEndpoint: server.URL, StorageRegion: "us-east-1", StorageAccountName: "testkey", StorageAccountKey: "testsecret", StorageContainer: "images", StoragePrivateContainer: "private-files", CORSOrigins: []string{"https://map.example.test"}, SundayOpenOrigins: "https://sunday.example.test"}
			p, err := NewPrivateProvider(cfg)
			if publicPolicy {
				if err == nil {
					t.Fatal("anonymous bucket accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(cors, "https://map.example.test") || !strings.Contains(cors, "https://sunday.example.test") || strings.Contains(cors, "<AllowedOrigin>*</AllowedOrigin>") {
				t.Fatal("CORS does not restrict configured client origins")
			}
			if err = p.Write(context.Background(), "final/immutable", "application/pdf", "test.pdf", []byte("first")); err != nil {
				t.Fatal(err)
			}
			if err = p.Write(context.Background(), "final/immutable", "application/pdf", "test.pdf", []byte("second")); err == nil || string(saved) != "first" {
				t.Fatal("accepted bytes were replaceable")
			}
		})
	}
}

func TestPrivateEndpointDoesNotContactPublicStorage(t *testing.T) {
	for _, scenario := range []string{"shared default", "separate private endpoint", "private endpoint only"} {
		t.Run(scenario, func(t *testing.T) {
			publicCalls, privateCalls := 0, 0
			handler := func(count *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*count++
					if r.URL.Query().Has("policy") {
						w.Header().Set("Content-Type", "application/xml")
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`<Error><Code>NoSuchBucketPolicy</Code></Error>`))
					}
				}
			}
			public := httptest.NewServer(handler(&publicCalls))
			defer public.Close()
			private := httptest.NewServer(handler(&privateCalls))
			defer private.Close()
			cfg := &config.Config{
				StorageProvider: "rustfs", StorageEndpoint: public.URL,
				StorageRegion: "us-east-1", StorageAccountName: "testkey", StorageAccountKey: "testsecret",
				StorageContainer: "images", StoragePrivateContainer: "private-files",
				CORSOrigins: []string{"https://map.example.test"},
			}
			if scenario != "shared default" {
				cfg.StoragePrivateEndpoint = private.URL
			}
			if scenario == "private endpoint only" {
				cfg.StorageEndpoint = ""
			}
			if _, err := NewPrivateProvider(cfg); err != nil {
				t.Fatal(err)
			}
			if scenario == "shared default" {
				if publicCalls == 0 || privateCalls != 0 {
					t.Fatal("default endpoint compatibility changed")
				}
			} else if publicCalls != 0 || privateCalls == 0 {
				t.Fatal("private requests reached the public storage provider")
			}
		})
	}
}
