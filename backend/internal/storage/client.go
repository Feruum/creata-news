package storage

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/studyverse/ems-backend/internal/config"
)

func NewProvider(cfg *config.Config) (Provider, error) {
	if cfg.StorageAccountName == "" || cfg.StorageAccountKey == "" {
		return nil, fmt.Errorf("storage credentials not configured")
	}

	provider := strings.ToLower(strings.TrimSpace(cfg.StorageProvider))
	if provider == "rustfs" && strings.TrimSpace(cfg.StorageEndpoint) == "" {
		slog.Warn("STORAGE_PROVIDER is rustfs but STORAGE_ENDPOINT is empty; falling back to azure storage provider")
		provider = "azure"
	}

	switch provider {
	case "minio", "s3", "rustfs":
		return NewS3Provider(&S3Config{
			AccessKey: cfg.StorageAccountName,
			SecretKey: cfg.StorageAccountKey,
			Bucket:    cfg.StorageContainer,
			Endpoint:  cfg.StorageEndpoint,
			Region:    cfg.StorageRegion,
			PublicURL: cfg.StoragePublicURL,
		})
	case "azure", "":
		return NewAzureProvider(&AzureConfig{
			AccountName:   cfg.StorageAccountName,
			AccountKey:    cfg.StorageAccountKey,
			ContainerName: cfg.StorageContainer,
		})
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.StorageProvider)
	}
}

func GenerateEventImageKey(filename string) string {
	timestamp := time.Now().UnixMilli()
	return fmt.Sprintf("events/%d-%s", timestamp, sanitizeFilename(filename))
}

func GenerateOrganizationImageKey(filename string) string {
	timestamp := time.Now().UnixMilli()
	return fmt.Sprintf("organizations/%d-%s", timestamp, sanitizeFilename(filename))
}

func GenerateUserAvatarKey(filename string) string {
	timestamp := time.Now().UnixMilli()
	return fmt.Sprintf("avatars/%d-%s", timestamp, sanitizeFilename(filename))
}

func sanitizeFilename(filename string) string {
	result := make([]byte, 0, len(filename))
	for i := 0; i < len(filename); i++ {
		c := filename[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			result = append(result, c)
		} else if c == ' ' {
			result = append(result, '-')
		}
	}
	if len(result) == 0 {
		return "file"
	}
	return string(result)
}
