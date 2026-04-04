// Package processor provides message processing pipeline for WeChat.
package message

import (
	"context"
	"fmt"

	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/message/cdn"
	"github.com/tingly-dev/weixin/storage"
)

// MessageProcessor handles the complete message processing pipeline.
type MessageProcessor struct {
	accountID    string
	baseURL      string
	cdnBaseURL   string
	token        string
	typingTicket string

	// Handlers
	onAuthCheck func(accountID, userID string) (authorized bool)
	onRoute     func(accountID, userID string) (agentID string)
	onDispatch  func(ctx context.Context, msg *ProcessMessage) error
	onDownload  func(mediaType, encryptedParam, aesKey, cdnBaseURL string) ([]byte, error)
}

// MediaDownloadOpts contains paths to downloaded media files.
type MediaDownloadOpts struct {
	ImagePath string
	VideoPath string
	VoicePath string
	VoiceMime string // MIME type for voice (audio/wav or audio/silk)
	FilePath  string
	FileMime  string
}

// ProcessMessage contains all data needed for message processing.
type ProcessMessage struct {
	WeixinMessage *api.WeixinMessage
	AccountID     string
	ToUserID      string
	ContextToken  string
	FromUserID    string
	SessionID     string
	TextBody      string
	Media         *MediaDownloadOpts
}

// NewMessageProcessor creates a new message processor.
func NewMessageProcessor(accountID, baseURL, cdnBaseURL, token string) *MessageProcessor {
	return &MessageProcessor{
		accountID:  accountID,
		baseURL:    baseURL,
		cdnBaseURL: cdnBaseURL,
		token:      token,
	}
}

// SetTypingTicket sets the typing ticket for sending typing indicators.
func (p *MessageProcessor) SetTypingTicket(ticket string) {
	p.typingTicket = ticket
}

// SetOnAuthCheck sets the authorization check handler.
func (p *MessageProcessor) SetOnAuthCheck(handler func(accountID, userID string) (authorized bool)) {
	p.onAuthCheck = handler
}

// SetOnRoute sets the routing handler.
func (p *MessageProcessor) SetOnRoute(handler func(accountID, userID string) (agentID string)) {
	p.onRoute = handler
}

// SetOnDispatch sets the dispatch handler.
func (p *MessageProcessor) SetOnDispatch(handler func(ctx context.Context, msg *ProcessMessage) error) {
	p.onDispatch = handler
}

// SetOnDownload sets the media download handler.
func (p *MessageProcessor) SetOnDownload(handler func(mediaType, encryptedParam, aesKey, cdnBaseURL string) ([]byte, error)) {
	p.onDownload = handler
}

// Process handles a single inbound message through the complete pipeline.
func (p *MessageProcessor) Process(ctx context.Context, msg *api.WeixinMessage) error {
	// 1. Check session guard
	if IsSessionPaused(p.accountID) {
		remaining := GetRemainingPauseMs(p.accountID)
		return fmt.Errorf("session paused for %v", remaining)
	}

	// 2. Filter only USER messages
	if msg.MessageType != api.MessageTypeUser {
		return nil
	}

	// 3. Check authorization
	if !p.checkAuthorization(msg.FromUserID) {
		return fmt.Errorf("user %s not authorized", msg.FromUserID)
	}

	// 4. Download media if present
	mediaOpts, err := p.downloadMedia(ctx, msg)
	if err != nil {
		return fmt.Errorf("download media: %w", err)
	}

	// 5. Build process message
	procMsg := &ProcessMessage{
		WeixinMessage: msg,
		AccountID:     p.accountID,
		ToUserID:      msg.FromUserID,
		ContextToken:  msg.ContextToken,
		FromUserID:    msg.FromUserID,
		SessionID:     msg.SessionID,
		TextBody:      p.extractText(msg),
		Media:         mediaOpts,
	}

	// 6. Route and dispatch
	agentID := p.routeMessage(msg.FromUserID)
	if agentID == "" {
		return fmt.Errorf("no agent found for user %s", msg.FromUserID)
	}

	// 7. Dispatch message
	if p.onDispatch != nil {
		return p.onDispatch(ctx, procMsg)
	}

	return nil
}

// checkAuthorization checks if a user is authorized to send commands.
func (p *MessageProcessor) checkAuthorization(userID string) bool {
	if p.onAuthCheck == nil {
		// Default: allow all
		return true
	}
	return p.onAuthCheck(p.accountID, userID)
}

// routeMessage routes a message to the appropriate agent.
func (p *MessageProcessor) routeMessage(userID string) string {
	if p.onRoute == nil {
		return "default"
	}
	return p.onRoute(p.accountID, userID)
}

// downloadMedia downloads and decrypts media from the message.
func (p *MessageProcessor) downloadMedia(ctx context.Context, msg *api.WeixinMessage) (*MediaDownloadOpts, error) {
	opts := &MediaDownloadOpts{}

	if msg.ItemList == nil {
		return opts, nil
	}

	for _, item := range msg.ItemList {
		switch item.Type {
		case api.MessageItemTypeImage:
			if item.ImageItem != nil && item.ImageItem.Media != nil {
				data, err := p.downloadImage(ctx, item.ImageItem)
				if err == nil && len(data) > 0 {
					// Save to temp file (simplified)
					opts.ImagePath = p.saveToTemp(data, "image")
				}
			}

		case api.MessageItemTypeVideo:
			if item.VideoItem != nil && item.VideoItem.Media != nil {
				data, err := p.downloadVideo(ctx, item.VideoItem)
				if err == nil && len(data) > 0 {
					opts.VideoPath = p.saveToTemp(data, "video")
				}
			}

		case api.MessageItemTypeVoice:
			if item.VoiceItem != nil && item.VoiceItem.Media != nil {
				data, err := p.downloadVoice(ctx, item.VoiceItem)
				if err == nil && len(data) > 0 {
					opts.VoicePath = p.saveToTemp(data, "voice")
					opts.VoiceMime = "audio/silk" // WeChat uses SILK format
				}
			}

		case api.MessageItemTypeFile:
			if item.FileItem != nil && item.FileItem.Media != nil {
				data, err := p.downloadFile(ctx, item.FileItem)
				if err == nil && len(data) > 0 {
					opts.FilePath = p.saveToTemp(data, item.FileItem.FileName)
					opts.FileMime = "application/octet-stream"
				}
			}
		}

		// Only download first media item
		if opts.ImagePath != "" || opts.VideoPath != "" || opts.VoicePath != "" || opts.FilePath != "" {
			break
		}
	}

	return opts, nil
}

// downloadImage downloads and decrypts an image.
func (p *MessageProcessor) downloadImage(ctx context.Context, img *api.ImageItem) ([]byte, error) {
	aesKey := p.getAESKey(img.AESKey, img.Media.AESKey)
	if p.onDownload != nil {
		return p.onDownload("image", img.Media.EncryptQueryParam, aesKey, p.cdnBaseURL)
	}
	return cdn.DownloadAndDecryptBuffer(ctx, img.Media.EncryptQueryParam, aesKey, p.cdnBaseURL)
}

// downloadVideo downloads and decrypts a video.
func (p *MessageProcessor) downloadVideo(ctx context.Context, vid *api.VideoItem) ([]byte, error) {
	aesKey := p.getAESKey("", vid.Media.AESKey)
	if p.onDownload != nil {
		return p.onDownload("video", vid.Media.EncryptQueryParam, aesKey, p.cdnBaseURL)
	}
	return cdn.DownloadAndDecryptBuffer(ctx, vid.Media.EncryptQueryParam, aesKey, p.cdnBaseURL)
}

// downloadVoice downloads and decrypts a voice message.
func (p *MessageProcessor) downloadVoice(ctx context.Context, voice *api.VoiceItem) ([]byte, error) {
	aesKey := voice.Media.AESKey
	if p.onDownload != nil {
		return p.onDownload("voice", voice.Media.EncryptQueryParam, aesKey, p.cdnBaseURL)
	}
	return cdn.DownloadAndDecryptBuffer(ctx, voice.Media.EncryptQueryParam, aesKey, p.cdnBaseURL)
}

// downloadFile downloads and decrypts a file.
func (p *MessageProcessor) downloadFile(ctx context.Context, file *api.FileItem) ([]byte, error) {
	aesKey := file.Media.AESKey
	if p.onDownload != nil {
		return p.onDownload("file", file.Media.EncryptQueryParam, aesKey, p.cdnBaseURL)
	}
	return cdn.DownloadAndDecryptBuffer(ctx, file.Media.EncryptQueryParam, aesKey, p.cdnBaseURL)
}

// getAESKey returns the AES key from either hex or base64 format.
func (p *MessageProcessor) getAESKey(hexKey, base64Key string) string {
	// Prefer hex key (16 bytes = 32 hex chars)
	if hexKey != "" && len(hexKey) == 32 {
		return hexKey
	}
	// Fall back to base64 key
	return base64Key
}

// extractText extracts the text body from a message.
func (p *MessageProcessor) extractText(msg *api.WeixinMessage) string {
	if msg.ItemList == nil {
		return ""
	}

	for _, item := range msg.ItemList {
		if item.Type == api.MessageItemTypeText && item.TextItem != nil {
			return item.TextItem.Text
		}
	}

	return ""
}

// saveToTemp saves data to a temp file and returns the path.
// This is a simplified implementation - in production, use proper temp file handling.
func (p *MessageProcessor) saveToTemp(data []byte, prefix string) string {
	stateDir, err := storage.GetWeixinStateDir()
	if err != nil {
		return ""
	}

	mediaDir := stateDir + "/media"
	if err := storage.EnsureDir(mediaDir); err != nil {
		return ""
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s-%d.tmp", prefix, len(data))
	path := mediaDir + "/" + filename

	// Write file
	if err := storage.WriteFileAtomic(path, data); err != nil {
		return ""
	}

	return path
}
