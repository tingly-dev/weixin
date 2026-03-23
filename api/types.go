// Package api provides WeChat API types.
package api

import "github.com/tingly-dev/weixin"

// GetUpdatesRequest represents the getUpdates request.
type GetUpdatesRequest struct {
	GetUpdatesBuf string    `json:"get_updates_buf"`
	BaseInfo      *BaseInfo `json:"base_info,omitempty"`
}

// GetUpdatesResponse represents the getUpdates response.
type GetUpdatesResponse struct {
	Ret                  int32                  `json:"ret"`
	ErrCode              int32                  `json:"errcode,omitempty"`
	ErrMsg               string                 `json:"errmsg,omitempty"`
	Messages             []weixin.WeixinMessage `json:"msgs,omitempty"`
	GetUpdatesBuf        string                 `json:"get_updates_buf,omitempty"`
	LongPollingTimeoutMs int                    `json:"longpolling_timeout_ms,omitempty"`
}

// SendMessageRequest represents the sendMessage request.
type SendMessageRequest struct {
	Msg      *WeixinMessageWrapper `json:"msg"`
	BaseInfo *BaseInfo             `json:"base_info,omitempty"`
}

// WeixinMessageWrapper wraps WeixinMessage for sending.
type WeixinMessageWrapper struct {
	FromUserID   string               `json:"from_user_id"`    // Bot ID (sender)
	ToUserID     string               `json:"to_user_id"`      // User ID (recipient)
	ClientID     string               `json:"client_id"`       // Unique client ID
	MessageType  int                  `json:"message_type"`    // 2 = BOT
	MessageState int                  `json:"message_state"`   // 2 = FINISH
	ContextToken string               `json:"context_token,omitempty"`
	ItemList     []weixin.MessageItem `json:"item_list"`
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
