// package weixin provides a WeChat channel plugin for AgentChannel.
//
// This plugin implements the WeChat messaging protocol, supporting:
// - QR code login for account authorization
// - Long-polling for message synchronization
// - Text and media message sending
// - AES-128-ECB encrypted CDN media uploads
package weixin

import "time"

// WeChatAccount represents a configured WeChat account.
type WeChatAccount struct {
	ID          string    `json:"id"`
	Name        string    `json:"name,omitempty"`
	BotToken    string    `json:"botToken"`
	BotID       string    `json:"botId"`
	UserID      string    `json:"userId"`
	BaseURL     string    `json:"baseUrl"`
	CDNBaseURL  string    `json:"cdnBaseUrl,omitempty"` // CDN base URL for media uploads/downloads
	Enabled     bool      `json:"enabled"`
	Configured  bool      `json:"configured"`
	CreatedAt   time.Time `json:"createdAt"`
	LastLoginAt time.Time `json:"lastLoginAt"`
}

// WeChatConfig holds plugin configuration.
type WeChatConfig struct {
	BaseURL string // Default API base URL
	BotType string // Bot type for QR login (default: "3")
}

// Message type constants from WeChat API.
const (
	MessageTypeNone = iota
	MessageTypeUser
	MessageTypeBot
)

// Message item type constants.
const (
	MessageItemTypeNone = iota
	MessageItemTypeText
	MessageItemTypeImage
	MessageItemTypeVoice
	MessageItemTypeFile
	MessageItemTypeVideo
)

// Message state constants.
const (
	MessageStateNew = iota
	MessageStateGenerating
	MessageStateFinish
)

// Upload media type constants.
const (
	UploadMediaTypeImage = iota + 1
	UploadMediaTypeVideo
	UploadMediaTypeFile
	UploadMediaTypeVoice
)

// WeixinMessage represents a message from WeChat API.
type WeixinMessage struct {
	Seq          int64         `json:"seq,omitempty"`
	MessageID    int64         `json:"message_id,omitempty"`
	FromUserID   string        `json:"from_user_id,omitempty"`
	ToUserID     string        `json:"to_user_id,omitempty"`
	CreateTimeMs int64         `json:"create_time_ms,omitempty"`
	SessionID    string        `json:"session_id,omitempty"`
	MessageType  int           `json:"message_type,omitempty"`
	MessageState int           `json:"message_state,omitempty"`
	ItemList     []MessageItem `json:"item_list,omitempty"`
	ContextToken string        `json:"context_token,omitempty"`
}

// MessageItem represents content within a message.
type MessageItem struct {
	Type      int        `json:"type,omitempty"`
	TextItem  *TextItem  `json:"text_item,omitempty"`
	ImageItem *ImageItem `json:"image_item,omitempty"`
	VoiceItem *VoiceItem `json:"voice_item,omitempty"`
	FileItem  *FileItem  `json:"file_item,omitempty"`
	VideoItem *VideoItem `json:"video_item,omitempty"`
}

// TextItem represents text content.
type TextItem struct {
	Text string `json:"text,omitempty"`
}

// ImageItem represents an image with CDN reference.
type ImageItem struct {
	Media       *CDNMedia `json:"media,omitempty"`
	ThumbMedia  *CDNMedia `json:"thumb_media,omitempty"`
	AESKey      string    `json:"aeskey,omitempty"`
	URL         string    `json:"url,omitempty"`
	MidSize     int64     `json:"mid_size,omitempty"`
	ThumbSize   int64     `json:"thumb_size,omitempty"`
	ThumbHeight int       `json:"thumb_height,omitempty"`
	ThumbWidth  int       `json:"thumb_width,omitempty"`
	HDSize      int64     `json:"hd_size,omitempty"`
}

// VoiceItem represents a voice message.
type VoiceItem struct {
	Media      *CDNMedia `json:"media,omitempty"`
	EncodeType int       `json:"encode_type,omitempty"`
	PlayTime   int       `json:"playtime,omitempty"`
	Text       string    `json:"text,omitempty"`
}

// FileItem represents a file attachment.
type FileItem struct {
	Media    *CDNMedia `json:"media,omitempty"`
	FileName string    `json:"file_name,omitempty"`
	MD5      string    `json:"md5,omitempty"`
	Len      string    `json:"len,omitempty"`
}

// VideoItem represents a video.
type VideoItem struct {
	Media      *CDNMedia `json:"media,omitempty"`
	ThumbMedia *CDNMedia `json:"thumb_media,omitempty"`
	VideoSize  int64     `json:"video_size,omitempty"`
	PlayLength int       `json:"play_length,omitempty"`
	VideoMD5   string    `json:"video_md5,omitempty"`
	ThumbSize  int64     `json:"thumb_size,omitempty"`
}

// CDNMedia represents encrypted CDN media reference.
type CDNMedia struct {
	EncryptQueryParam string `json:"encrypt_query_param,omitempty"`
	AESKey            string `json:"aes_key,omitempty"`
	EncryptType       int    `json:"encrypt_type,omitempty"` // 0=only fileid, 1=包含缩略图/中图等信息
	FullURL           string `json:"full_url,omitempty"`     // Server-returned complete download URL
}
