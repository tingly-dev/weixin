// Package wechat provides WeChat ilink bot implementation.
package wechat

import (
	"context"
	"fmt"

	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/message"
	"github.com/tingly-dev/weixin/message/media"
	"github.com/tingly-dev/weixin/types"
)

// Send sends a text or mixed message.
func (b *WechatBot) Send(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	if b.account == nil {
		return nil, fmt.Errorf("bot has no account configured")
	}

	// Convert message
	items := message.ConvertOutboundMessageToList(msg)

	// Send directly via API client
	if err := b.account.Client().SendMessage(ctx, msg.To, msg.ContextToken, items); err != nil {
		return &types.OutboundResult{OK: false, Error: err.Error()}, err
	}

	return &types.OutboundResult{OK: true}, nil
}

// SendMedia uploads media to CDN and sends the message.
func (b *WechatBot) SendMedia(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	if b.account == nil {
		return nil, fmt.Errorf("bot has no account configured")
	}

	if msg.FilePath == "" {
		return nil, fmt.Errorf("FilePath is required for media upload")
	}

	client := b.account.Client()
	account := b.account.WeChatAccount()

	// Detect media type
	mediaType := types.UploadMediaTypeFile
	switch msg.ContentType {
	case "image":
		mediaType = types.UploadMediaTypeImage
	case "video":
		mediaType = types.UploadMediaTypeVideo
	case "audio", "voice":
		mediaType = types.UploadMediaTypeVoice
	}

	// Upload media to CDN (preserving existing pipeline)
	uploaded, err := media.UploadMediaToCDN(ctx, msg.FilePath, b.account.ID(), account.BaseURL, account.CDNBaseURL, account.BotToken, mediaType)
	if err != nil {
		return nil, fmt.Errorf("upload media: %w", err)
	}

	// Build message item with CDN metadata
	var item api.MessageItem
	switch msg.ContentType {
	case "image":
		item = message.BuildImageItemFromUpload(uploaded, uploaded.FileSize)
	case "video":
		item = message.BuildVideoItemFromUpload(uploaded, uploaded.FileSize)
	case "audio", "voice":
		// Build voice item from upload
		item = api.MessageItem{
			Type: api.MessageItemTypeVoice,
			VoiceItem: &api.VoiceItem{
				Media: &api.CDNMedia{
					EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
					AESKey:            string(uploaded.AESKey),
					EncryptType:       1,
				},
			},
		}
	default:
		item = message.BuildFileItemFromUpload(uploaded, msg.FileName, uploaded.FileSize)
	}

	// Send directly via API client
	if err := client.SendMessage(ctx, msg.To, msg.ContextToken, []api.MessageItem{item}); err != nil {
		return &types.OutboundResult{OK: false, Error: err.Error()}, err
	}

	return &types.OutboundResult{OK: true}, nil
}

// SendStream sends a streaming text chunk.
func (b *WechatBot) SendStream(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	if b.account == nil {
		return nil, fmt.Errorf("bot has no account configured")
	}

	// Convert message
	items := message.ConvertOutboundMessageToList(msg)

	// Send directly via API client
	if err := b.account.Client().SendMessage(ctx, msg.To, msg.ContextToken, items); err != nil {
		return &types.OutboundResult{OK: false, Error: err.Error()}, err
	}

	return &types.OutboundResult{OK: true}, nil
}

// GetUploadURL retrieves a pre-signed URL for uploading media to WeChat CDN.
// This is useful for custom upload workflows where you want to upload directly to CDN.
func (b *WechatBot) GetUploadURL(ctx context.Context, req *types.UploadURLRequest) (*types.UploadURLResult, error) {
	if b.account == nil {
		return nil, fmt.Errorf("bot has no account configured")
	}

	// Call API client
	apiReq := &api.GetUploadURLRequest{
		FileKey:   req.FileKey,
		MediaType: req.MediaType,
		RawSize:   req.RawSize,
		RawMD5:    req.RawMD5,
		FileSize:  req.FileSize,
		AESKey:    req.AESKey,
	}

	resp, err := b.account.Client().GetUploadURL(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("get upload URL: %w", err)
	}

	return &types.UploadURLResult{
		UploadParam: resp.UploadParam,
		FileKey:     req.FileKey,
		AuthToken:   "", // WeChat doesn't require separate auth token
	}, nil
}
