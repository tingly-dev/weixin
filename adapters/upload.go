// Package adapters provides adapter implementations for the WeChat channel.
package adapters

import (
	"context"
	"fmt"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/channel"
	"github.com/tingly-dev/weixin/media"
)

// UploadAdapter handles media file uploads to WeChat CDN.
type UploadAdapter struct {
	plugin weixin.PluginInterface
}

// NewUploadAdapter creates a new upload adapter.
func NewUploadAdapter(plugin weixin.PluginInterface) *UploadAdapter {
	return &UploadAdapter{plugin: plugin}
}

// GetUploadURL retrieves a pre-signed URL for uploading media.
func (a *UploadAdapter) GetUploadURL(ctx context.Context, req *channel.UploadURLRequest) (*channel.UploadURLResult, error) {
	// Get account
	account, err := a.plugin.Accounts().Get(req.FileKey) // FileKey is used as accountID here
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	// Create API client
	client := api.NewClient(account.BaseURL, account.BotToken)

	// Build request
	apiReq := &api.GetUploadURLRequest{
		FileKey:   req.FileKey,
		MediaType: req.MediaType,
		RawSize:   req.RawSize,
		RawMD5:    req.RawMD5,
		FileSize:  req.FileSize,
		AESKey:    req.AESKey,
	}

	// Call API
	resp, err := client.GetUploadURL(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("get upload URL: %w", err)
	}

	return &channel.UploadURLResult{
		UploadParam: resp.UploadParam,
	}, nil
}

// UploadMedia uploads a media file and returns the reference.
func (a *UploadAdapter) UploadMedia(ctx context.Context, req *channel.MediaUploadRequest) (*channel.MediaUploadResult, error) {
	// Use the UploadMediaToCDN function which handles the full pipeline
	// First, get an account to use
	accounts, err := a.plugin.Accounts().ListIDs()
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts available")
	}

	account, err := a.plugin.Accounts().Get(accounts[0])
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	// Use media.UploadMediaToCDN for the complete upload pipeline
	cdnBaseURL := account.CDNBaseURL
	if cdnBaseURL == "" {
		cdnBaseURL = "https://novac2c.cdn.weixin.qq.com/c2c" // Default CDN URL
	}

	uploaded, err := media.UploadMediaToCDN(
		ctx,
		req.FilePath,
		account.ID, // Use account ID as toUserID for now
		account.BaseURL,
		cdnBaseURL,
		account.BotToken,
		getMediaType(req.MediaType),
	)
	if err != nil {
		return nil, fmt.Errorf("upload media: %w", err)
	}

	return &channel.MediaUploadResult{
		FileKey:      uploaded.FileKey,
		FileSize:     uploaded.FileSize,
		EncryptKey:   uploaded.AESKey,
		EncryptQuery: uploaded.DownloadEncryptedQueryParam,
	}, nil
}

// getMediaType converts media type string to WeChat constant.
func getMediaType(mediaType string) int {
	switch mediaType {
	case "image":
		return weixin.UploadMediaTypeImage
	case "video":
		return weixin.UploadMediaTypeVideo
	case "audio", "voice":
		return weixin.UploadMediaTypeVoice
	default:
		return weixin.UploadMediaTypeFile
	}
}
