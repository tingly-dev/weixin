// Package adapters provides adapter implementations for the WeChat channel.
package adapters

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/channel"
	"github.com/tingly-dev/weixin/crypto"
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
	// Read file
	data, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Calculate MD5 of plaintext
	rawMD5 := fmt.Sprintf("%x", md5.Sum(data))
	rawSize := int64(len(data))

	// Generate AES key
	aesKey := make([]byte, 16)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("generate AES key: %w", err)
	}

	// Encrypt file
	encrypted, err := crypto.EncryptAesEcb(data, aesKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt file: %w", err)
	}
	fileSize := int64(len(encrypted))

	// Encode AES key as base64
	aesKeyB64 := base64.StdEncoding.EncodeToString(aesKey)

	// TODO: Handle thumbnail generation for images/videos
	// For now, we'll skip thumbnail upload

	// Get account (using first available account)
	accounts, err := a.plugin.Accounts().ListIDs()
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts available")
	}

	// Create API client
	account, err := a.plugin.Accounts().Get(accounts[0])
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	client := api.NewClient(account.BaseURL, account.BotToken)

	// Get upload URL
	uploadReq := &api.GetUploadURLRequest{
		MediaType:   getMediaType(req.MediaType),
		RawSize:     rawSize,
		RawMD5:      rawMD5,
		FileSize:    fileSize,
		AESKey:      aesKeyB64,
		NoNeedThumb: true, // Skip thumbnail for now
	}

	uploadResp, err := client.GetUploadURL(ctx, uploadReq)
	if err != nil {
		return nil, fmt.Errorf("get upload URL: %w", err)
	}

	// TODO: Upload encrypted file to CDN using uploadResp.UploadParam
	// This requires parsing the upload_param and doing a PUT request

	return &channel.MediaUploadResult{
		FileKey:      req.FilePath,
		FileSize:     rawSize,
		EncryptKey:   aesKey,
		EncryptQuery: uploadResp.UploadParam,
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
