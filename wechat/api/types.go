// Package api provides WeChat API implementations and types.
package api

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
	EncryptType       int    `json:"encrypt_type,omitempty"` // 0=only fileid, 1=includes thumb/mid info
	FullURL           string `json:"full_url,omitempty"`     // Server-returned complete download URL
}

// GetUpdatesRequest represents the getUpdates request.
type GetUpdatesRequest struct {
	GetUpdatesBuf string    `json:"get_updates_buf"`
	BaseInfo      *BaseInfo `json:"base_info,omitempty"`
}

// GetUpdatesResponse represents the getUpdates response.
type GetUpdatesResponse struct {
	Ret                  int32           `json:"ret"`
	ErrCode              int32           `json:"errcode,omitempty"`
	ErrMsg               string          `json:"errmsg,omitempty"`
	Messages             []WeixinMessage `json:"msgs,omitempty"`
	GetUpdatesBuf        string          `json:"get_updates_buf,omitempty"`
	LongPollingTimeoutMs int             `json:"longpolling_timeout_ms,omitempty"`
}

// SendMessageRequest represents the sendMessage request.
type SendMessageRequest struct {
	Msg      *WeixinMessageWrapper `json:"msg"`
	BaseInfo *BaseInfo             `json:"base_info,omitempty"`
}

// WeixinMessageWrapper wraps WeixinMessage for sending.
type WeixinMessageWrapper struct {
	FromUserID   string        `json:"from_user_id"`  // Bot ID (sender)
	ToUserID     string        `json:"to_user_id"`    // User ID (recipient)
	ClientID     string        `json:"client_id"`     // Unique client ID
	MessageType  int           `json:"message_type"`  // 2 = BOT
	MessageState int           `json:"message_state"` // 2 = FINISH
	ContextToken string        `json:"context_token,omitempty"`
	ItemList     []MessageItem `json:"item_list"`
}

// GetUploadURLRequest represents the getUploadUrl request.
type GetUploadURLRequest struct {
	FileKey       string    `json:"filekey,omitempty"`
	MediaType     int       `json:"media_type,omitempty"`
	ToUserID      string    `json:"to_user_id,omitempty"`
	RawSize       int64     `json:"rawsize,omitempty"`
	RawMD5        string    `json:"rawfilemd5,omitempty"`
	FileSize      int64     `json:"filesize,omitempty"`
	ThumbRawSize  int64     `json:"thumb_rawsize,omitempty"`
	ThumbRawMD5   string    `json:"thumb_rawfilemd5,omitempty"`
	ThumbFileSize int64     `json:"thumb_filesize,omitempty"`
	NoNeedThumb   bool      `json:"no_need_thumb,omitempty"`
	AESKey        string    `json:"aeskey,omitempty"`
	BaseInfo      *BaseInfo `json:"base_info,omitempty"`
}

// GetUploadURLResponse represents the getUploadUrl response.
type GetUploadURLResponse struct {
	UploadParam      string `json:"upload_param,omitempty"`
	ThumbUploadParam string `json:"thumb_upload_param,omitempty"`
	UploadFullURL    string `json:"upload_full_url,omitempty"`
}

// GetConfigRequest represents the getConfig request.
type GetConfigRequest struct {
	IlinkUserID  string    `json:"ilink_user_id,omitempty"`
	ContextToken string    `json:"context_token,omitempty"`
	BaseInfo     *BaseInfo `json:"base_info,omitempty"`
}

// GetConfigResponse represents the getConfig response.
type GetConfigResponse struct {
	Ret          int32  `json:"ret"`
	ErrMsg       string `json:"errmsg,omitempty"`
	TypingTicket string `json:"typing_ticket,omitempty"`
}

// SendTypingRequest represents the sendTyping request.
type SendTypingRequest struct {
	IlinkUserID  string    `json:"ilink_user_id,omitempty"`
	TypingTicket string    `json:"typing_ticket,omitempty"`
	Status       int       `json:"status,omitempty"` // 1=typing, 2=cancel
	BaseInfo     *BaseInfo `json:"base_info,omitempty"`
}

// QRCodeRequest represents the get_bot_qrcode request (query params).
type QRCodeRequest struct {
	BotType string `json:"bot_type,omitempty"` // Default: "3"
}

// QRCodeResponse represents the get_bot_qrcode response.
type QRCodeResponse struct {
	Qrcode           string `json:"qrcode,omitempty"`
	QrcodeImgContent string `json:"qrcode_img_content,omitempty"`
}

// QRStatusResponse represents the get_qrcode_status response.
type QRStatusResponse struct {
	Status      string `json:"status,omitempty"` // wait, scaned, confirmed, expired
	BotToken    string `json:"bot_token,omitempty"`
	IlinkBotID  string `json:"ilink_bot_id,omitempty"`
	BaseURL     string `json:"baseurl,omitempty"`
	IlinkUserID string `json:"ilink_user_id,omitempty"`
}
