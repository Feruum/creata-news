package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/studyverse/ems-backend/internal/config"
)

// PrivateProvider deliberately exposes no public object URL. Only staging keys
// can be signed for writes; accepted bytes are written to a different final key.
type PrivateProvider interface {
	SignUpload(context.Context, string, string, time.Duration) (*UploadURLResult, error)
	Read(context.Context, string, int64) ([]byte, error)
	Write(context.Context, string, string, string, []byte) error
	Delete(context.Context, string) error
	SignDownload(context.Context, string, string, time.Duration) (string, error)
}

type privateS3 struct {
	*S3Provider
	proxyPrefix string
}
type privateAzure struct{ *AzureProvider }

func NewPrivateProvider(cfg *config.Config) (PrivateProvider, error) {
	if cfg == nil || cfg.StorageAccountName == "" || cfg.StorageAccountKey == "" {
		return nil, fmt.Errorf("private storage credentials not configured")
	}
	bucket := strings.TrimSpace(cfg.StoragePrivateContainer)
	if bucket == "" {
		bucket = "ems-registration-files"
	}
	if bucket == cfg.StorageContainer {
		return nil, fmt.Errorf("private storage must use a separate container")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	origins := []string{}
	seen := map[string]bool{}
	for _, origin := range append(append([]string{}, cfg.CORSOrigins...), strings.Split(cfg.SundayOpenOrigins, ",")...) {
		origin = strings.TrimSpace(origin)
		if origin != "" && origin != "*" && !seen[origin] {
			seen[origin] = true
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return nil, fmt.Errorf("private storage requires configured client origins")
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.StorageProvider))
	endpoint := cfg.StoragePrivateEndpoint
	if endpoint == "" {
		endpoint = cfg.StorageEndpoint
	}
	if provider == "rustfs" && endpoint == "" {
		provider = "azure"
	}
	switch provider {
	case "s3", "minio", "rustfs":
		p, err := NewS3Provider(&S3Config{AccessKey: cfg.StorageAccountName, SecretKey: cfg.StorageAccountKey, Bucket: bucket, Endpoint: endpoint, Region: cfg.StorageRegion})
		if err != nil {
			return nil, err
		}
		if _, err = p.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
			return nil, fmt.Errorf("private bucket unavailable: %w", err)
		}
		// Refuse accidental anonymous policies. Provisioning owns policy removal.
		policy, e := p.client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{Bucket: aws.String(bucket)})
		if e != nil {
			var api smithy.APIError
			if !errors.As(e, &api) || api.ErrorCode() != "NoSuchBucketPolicy" {
				return nil, fmt.Errorf("verify private bucket policy: %w", e)
			}
		}
		if e == nil && policy.Policy != nil && strings.Contains(*policy.Policy, "Allow") {
			return nil, fmt.Errorf("private bucket must have no allow policy")
		}
		_, err = p.client.PutBucketCors(ctx, &s3.PutBucketCorsInput{Bucket: aws.String(bucket), CORSConfiguration: &types.CORSConfiguration{CORSRules: []types.CORSRule{{AllowedOrigins: origins, AllowedMethods: []string{"GET", "PUT", "HEAD"}, AllowedHeaders: []string{"Content-Type", "Cache-Control", "If-None-Match", "x-amz-*"}, ExposeHeaders: []string{"ETag"}, MaxAgeSeconds: aws.Int32(300)}}}})
		if err != nil {
			return nil, fmt.Errorf("configure private bucket CORS: %w", err)
		}
		// Sign the path RustFS receives after Caddy strips its external prefix.
		// Preserve the external Host because Caddy preserves incoming Host too.
		prefix := ""
		if cfg.StoragePublicURL != "" {
			endpoint := strings.TrimSuffix(strings.TrimRight(cfg.StoragePublicURL, "/"), "/"+cfg.StorageContainer)
			external, err := url.Parse(endpoint)
			if err != nil || external.Host == "" || external.User != nil || (external.Scheme != "https" && external.Scheme != "http") {
				return nil, fmt.Errorf("invalid public storage endpoint")
			}
			prefix = strings.TrimRight(external.Path, "/")
			endpoint = external.Scheme + "://" + external.Host
			p.presignClient = s3.NewPresignClient(p.client, func(o *s3.PresignOptions) {
				o.ClientOptions = append(o.ClientOptions, func(c *s3.Options) { c.BaseEndpoint = aws.String(endpoint); c.UsePathStyle = true })
			})
		}
		return &privateS3{S3Provider: p, proxyPrefix: prefix}, nil
	case "azure", "":
		p, err := NewAzureProvider(&AzureConfig{AccountName: cfg.StorageAccountName, AccountKey: cfg.StorageAccountKey, ContainerName: bucket})
		if err != nil {
			return nil, err
		}
		props, err := p.client.ServiceClient().NewContainerClient(bucket).GetProperties(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("private Azure container unavailable: %w", err)
		}
		if props.BlobPublicAccess != nil && *props.BlobPublicAccess != "" {
			return nil, fmt.Errorf("private Azure container allows public access")
		}
		// Azure CORS is account-wide. Merge the private rule to preserve image rules.
		svc := p.ServiceClient()
		properties, err := svc.GetProperties(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("read Azure CORS: %w", err)
		}
		rules := properties.CORS
		privateOrigins := strings.Join(origins, ",")
		hasPrivateRule := false
		for _, rule := range rules {
			if rule == nil || rule.AllowedOrigins == nil {
				continue
			}
			for _, origin := range strings.Split(*rule.AllowedOrigins, ",") {
				if !seen[strings.TrimSpace(origin)] {
					return nil, fmt.Errorf("Azure account CORS contains origins outside configured client origins")
				}
			}
			if *rule.AllowedOrigins == privateOrigins && rule.AllowedMethods != nil && *rule.AllowedMethods == "GET,PUT,HEAD" {
				hasPrivateRule = true
			}
		}
		if !hasPrivateRule {
			rules = append(rules, &service.CORSRule{AllowedOrigins: aws.String(privateOrigins), AllowedMethods: aws.String("GET,PUT,HEAD"), AllowedHeaders: aws.String("content-type,x-ms-blob-type,x-ms-version"), ExposedHeaders: aws.String("ETag"), MaxAgeInSeconds: aws.Int32(300)})
		}
		if len(rules) > 5 {
			return nil, fmt.Errorf("Azure CORS rule limit reached")
		}
		if _, err = svc.SetProperties(ctx, &service.SetPropertiesOptions{CORS: rules}); err != nil {
			return nil, fmt.Errorf("configure private Azure CORS: %w", err)
		}
		return &privateAzure{p}, nil
	default:
		return nil, fmt.Errorf("unsupported private storage provider")
	}
}

func boundedRead(r io.ReadCloser, max int64) ([]byte, error) {
	defer r.Close()
	if max < 1 || max > 10*1024*1024 {
		return nil, fmt.Errorf("invalid private read limit")
	}
	data := make([]byte, max+1)
	n, err := io.ReadFull(r, data)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	if int64(n) > max {
		return nil, fmt.Errorf("object exceeds upload size limit")
	}
	return data[:n], nil
}
func disposition(filename string) string {
	return mime.FormatMediaType("attachment", map[string]string{"filename": filename})
}

func (p *privateS3) SignUpload(ctx context.Context, key, mime string, expiry time.Duration) (*UploadURLResult, error) {
	if !strings.HasPrefix(key, "staging/") {
		return nil, fmt.Errorf("only staging uploads may be signed")
	}
	r, e := p.GeneratePresignedUploadURL(ctx, key, mime, expiry)
	if r != nil {
		r.PublicURL = ""
		r.UploadURL = p.externalURL(r.UploadURL)
	}
	return r, e
}
func (p *privateS3) Read(ctx context.Context, key string, max int64) ([]byte, error) {
	r, e := p.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(p.bucket), Key: aws.String(key)})
	if e != nil {
		return nil, e
	}
	return boundedRead(r.Body, max)
}
func (p *privateS3) Write(ctx context.Context, key, mime, name string, data []byte) error {
	if !strings.HasPrefix(key, "final/") {
		return fmt.Errorf("accepted files require final keys")
	}
	_, e := p.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(p.bucket), Key: aws.String(key), Body: bytes.NewReader(data), ContentType: aws.String(mime), ContentLength: aws.Int64(int64(len(data))), CacheControl: aws.String("private, no-store"), ContentDisposition: aws.String(disposition(name)), IfNoneMatch: aws.String("*")})
	return e
}
func (p *privateS3) Delete(ctx context.Context, key string) error { return p.DeleteObject(ctx, key) }
func (p *privateS3) SignDownload(ctx context.Context, key, name string, expiry time.Duration) (string, error) {
	r, e := p.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(p.bucket), Key: aws.String(key), ResponseCacheControl: aws.String("private, no-store"), ResponseContentDisposition: aws.String(disposition(name))}, s3.WithPresignExpires(expiry))
	if e != nil {
		return "", e
	}
	return p.externalURL(r.URL), nil
}

func (p *privateS3) externalURL(signed string) string {
	if p.proxyPrefix == "" {
		return signed
	}
	u, err := url.Parse(signed)
	if err != nil {
		return signed
	}
	u.Path = p.proxyPrefix + u.Path
	u.RawPath = ""
	return u.String()
}

func (p *privateAzure) SignUpload(ctx context.Context, key, mime string, expiry time.Duration) (*UploadURLResult, error) {
	if !strings.HasPrefix(key, "staging/") {
		return nil, fmt.Errorf("only staging uploads may be signed")
	}
	r, e := p.GeneratePresignedUploadURL(ctx, key, mime, expiry)
	if r != nil {
		r.PublicURL = ""
	}
	return r, e
}
func (p *privateAzure) Read(ctx context.Context, key string, max int64) ([]byte, error) {
	r, e := p.client.DownloadStream(ctx, p.containerName, key, nil)
	if e != nil {
		return nil, e
	}
	return boundedRead(r.Body, max)
}
func (p *privateAzure) Write(ctx context.Context, key, mime, name string, data []byte) error {
	if !strings.HasPrefix(key, "final/") {
		return fmt.Errorf("accepted files require final keys")
	}
	cache, disp := "private, no-store", disposition(name)
	none := azcore.ETag("*")
	_, e := p.client.UploadBuffer(ctx, p.containerName, key, data, &azblob.UploadBufferOptions{HTTPHeaders: &blob.HTTPHeaders{BlobContentType: &mime, BlobCacheControl: &cache, BlobContentDisposition: &disp}, AccessConditions: &blob.AccessConditions{ModifiedAccessConditions: &blob.ModifiedAccessConditions{IfNoneMatch: &none}}})
	return e
}
func (p *privateAzure) Delete(ctx context.Context, key string) error {
	err := p.DeleteObject(ctx, key)
	var response *azcore.ResponseError
	if errors.As(err, &response) && response.ErrorCode == "BlobNotFound" {
		return nil
	}
	return err
}
func (p *privateAzure) SignDownload(ctx context.Context, key, name string, expiry time.Duration) (string, error) {
	now := time.Now().UTC()
	q, e := (sas.BlobSignatureValues{Protocol: sas.ProtocolHTTPS, StartTime: now.Add(-time.Minute), ExpiryTime: now.Add(expiry), Permissions: (&sas.BlobPermissions{Read: true}).String(), ContainerName: p.containerName, BlobName: key, ContentDisposition: disposition(name), CacheControl: "private, no-store"}).SignWithSharedKey(p.credential)
	if e != nil {
		return "", e
	}
	return p.client.ServiceClient().NewContainerClient(p.containerName).NewBlobClient(key).URL() + "?" + q.Encode(), nil
}
