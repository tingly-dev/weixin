package wechat

import (
	"context"
	"fmt"

	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/message/media"
	"github.com/tingly-dev/weixin/types"
)

// UploadAdapter handles media file uploads to WeChat CDN.
type UploadAdapter struct {
	bot *WechatBot
}

// NewUploadAdapter creates a new upload adapter.
func NewUploadAdapter(bot *WechatBot) *UploadAdapter {
	return &UploadAdapter{bot: bot}
}

// GetUploadURL retrieves a pre-signed URL for uploading media.
func (a *UploadAdapter) GetUploadURL(ctx context.Context, req *types.UploadURLRequest) (*types.UploadURLResult, error) {
	// Get account
	account, err := a.bot.Accounts().Get(req.FileKey) // FileKey is used as accountID here
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

	return &types.UploadURLResult{
		UploadParam: resp.UploadParam,
	}, nil
}

// UploadMedia uploads a media file and returns the reference.
func (a *UploadAdapter) UploadMedia(ctx context.Context, req *types.MediaUploadRequest) (*types.MediaUploadResult, error) {
	// Use the UploadMediaToCDN function which handles the full pipeline
	// First, get an account to use
	accounts, err := a.bot.Accounts().ListIDs()
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts available")
	}

	account, err := a.bot.Accounts().Get(accounts[0])
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

	return &types.MediaUploadResult{
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
		return types.UploadMediaTypeImage
	case "video":
		return types.UploadMediaTypeVideo
	case "audio", "voice":
		return types.UploadMediaTypeVoice
	default:
		return types.UploadMediaTypeFile
	}
}
