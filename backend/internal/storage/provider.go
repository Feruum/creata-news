package storage

import (
	"context"
	"time"
)

type UploadURLResult struct {
	UploadURL string
	PublicURL string
	ObjectKey string
	Headers   map[string]string
}

type Provider interface {
	UploadObject(ctx context.Context, key, contentType string, data []byte) error
	GeneratePresignedUploadURL(ctx context.Context, key, contentType string, expiry time.Duration) (*UploadURLResult, error)
	DeleteObject(ctx context.Context, key string) error
	GetPublicURL(key string) string
}
