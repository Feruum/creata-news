package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

type AzureProvider struct {
	client        *azblob.Client
	containerName string
	accountName   string
	credential    *azblob.SharedKeyCredential
}

type AzureConfig struct {
	AccountName   string
	AccountKey    string
	ContainerName string
}

func NewAzureProvider(cfg *AzureConfig) (*AzureProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("azure config is nil")
	}

	if cfg.AccountName == "" || cfg.AccountKey == "" {
		return nil, fmt.Errorf("azure account name and key are required")
	}

	credential, err := azblob.NewSharedKeyCredential(cfg.AccountName, cfg.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", cfg.AccountName)

	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Blob client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containerClient := client.ServiceClient().NewContainerClient(cfg.ContainerName)
	_, err = containerClient.GetProperties(ctx, nil)
	if err != nil {
		slog.Warn("Failed to verify Azure container", "container", cfg.ContainerName, "error", err)
	} else {
		slog.Info("Azure storage container verified", "container", cfg.ContainerName, "account", cfg.AccountName)
	}

	return &AzureProvider{
		client:        client,
		containerName: cfg.ContainerName,
		accountName:   cfg.AccountName,
		credential:    credential,
	}, nil
}

func (p *AzureProvider) GeneratePresignedUploadURL(ctx context.Context, blobName string, contentType string, expiry time.Duration) (*UploadURLResult, error) {
	permissions := sas.BlobPermissions{
		Write:  true,
		Create: true,
	}

	// Azure SAS timestamps must be encoded in UTC. The SDK formats the values
	// into query params and using local time (e.g. TZ=Etc/GMT-5 in prod) can
	// produce future windows that Azure rejects as AuthenticationFailed.
	nowUTC := time.Now().UTC()
	expiryTime := nowUTC.Add(expiry)

	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		StartTime:     nowUTC.Add(-5 * time.Minute),
		ExpiryTime:    expiryTime,
		Permissions:   permissions.String(),
		ContainerName: p.containerName,
		BlobName:      blobName,
		ContentType:   contentType,
	}.SignWithSharedKey(p.credential)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SAS token: %w", err)
	}

	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		p.accountName, p.containerName, blobName)
	sasURL := fmt.Sprintf("%s?%s", blobURL, sasQueryParams.Encode())

	return &UploadURLResult{
		UploadURL: sasURL,
		PublicURL: blobURL,
		ObjectKey: blobName,
		Headers: map[string]string{
			"Content-Type":   contentType,
			"x-ms-blob-type": "BlockBlob",
		},
	}, nil
}

func (p *AzureProvider) DeleteObject(ctx context.Context, blobName string) error {
	_, err := p.client.DeleteBlob(ctx, p.containerName, blobName, nil)
	if err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
	}
	return nil
}

func (p *AzureProvider) GetPublicURL(blobName string) string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		p.accountName, p.containerName, blobName)
}

func (p *AzureProvider) ServiceClient() *service.Client {
	return p.client.ServiceClient()
}

func (p *AzureProvider) UploadObject(ctx context.Context, key, contentType string, data []byte) error {
	cache := "public, max-age=31536000, immutable"
	_, err := p.client.UploadBuffer(ctx, p.containerName, key, data, &azblob.UploadBufferOptions{HTTPHeaders: &blob.HTTPHeaders{BlobContentType: &contentType, BlobCacheControl: &cache}})
	if err != nil {
		return fmt.Errorf("upload validated image: %w", err)
	}
	return nil
}
