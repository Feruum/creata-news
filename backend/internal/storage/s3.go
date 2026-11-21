package storage

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Provider struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	endpoint      string
	region        string
	publicURL     string
}

type S3Config struct {
	AccessKey string
	SecretKey string
	Bucket    string
	Endpoint  string
	Region    string
	PublicURL string
}

func NewS3Provider(cfg *S3Config) (*S3Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("s3 config is nil")
	}

	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("s3 access key and secret key are required")
	}

	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket name is required")
	}

	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	clientOpts := func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	}

	client := s3.NewFromConfig(awsCfg, clientOpts)
	presignClient := s3.NewPresignClient(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(cfg.Bucket),
	})
	if err != nil {
		slog.Warn("Failed to verify S3 bucket", "bucket", cfg.Bucket, "error", err)
	} else {
		slog.Info("S3 bucket verified", "bucket", cfg.Bucket, "endpoint", cfg.Endpoint)
	}

	return &S3Provider{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.Bucket,
		endpoint:      cfg.Endpoint,
		region:        region,
		publicURL:     cfg.PublicURL,
	}, nil
}

func (p *S3Provider) GeneratePresignedUploadURL(ctx context.Context, key string, contentType string, expiry time.Duration) (*UploadURLResult, error) {
	presignedReq, err := p.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(p.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	publicURL := p.GetPublicURL(key)

	return &UploadURLResult{
		UploadURL: presignedReq.URL,
		PublicURL: publicURL,
		ObjectKey: key,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
	}, nil
}

func (p *S3Provider) DeleteObject(ctx context.Context, key string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (p *S3Provider) GetPublicURL(key string) string {
	if p.publicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(p.publicURL, "/"), key)
	}
	if p.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", p.endpoint, p.bucket, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", p.bucket, p.region, key)
}

func (p *S3Provider) UploadObject(ctx context.Context, key, contentType string, data []byte) error {
	_, err := p.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(p.bucket), Key: aws.String(key), Body: bytes.NewReader(data), ContentType: aws.String(contentType), ContentLength: aws.Int64(int64(len(data))), CacheControl: aws.String("public, max-age=31536000, immutable")})
	if err != nil {
		return fmt.Errorf("upload validated image: %w", err)
	}
	return nil
}
