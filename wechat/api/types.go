// Package api provides WeChat API implementations and types.
package api

import "encoding/json"

// LosslessID is a message/server ID that may arrive either as a JSON number
// (uint64, e.g. 7507094999942111240) or as a JSON string (e.g.
// "v1:15492892230605430853"). The upstream openclaw-weixin plugin routes the
// fields message_id, msg_id and svr_id through a lossless parser that quotes
// raw uint64 tokens into strings before JSON.parse (api.ts LOSSLESS_ID_FIELDS);
// this type is the Go equivalent: numbers are captured verbatim as their
// decimal text, strings are taken as-is, so no precision or prefix is lost.
type LosslessID string

// UnmarshalJSON accepts either a JSON string or a raw number token.
func (id *LosslessID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*id = ""
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*id = LosslessID(s)
		return nil
	}
	// Raw number token: keep the literal digits to avoid float64 precision loss.
	*id = LosslessID(data)
	return nil
}

// MarshalJSON always emits a JSON string, matching the upstream plugin's
// post-lossless-parse representation.
func (id LosslessID) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(id))
}

// Message type constants from WeChat API.
const (
	MessageTypeNone = iota
	MessageTypeUser
	MessageTypeBot
)

// Message item type constants.
//
// The numeric IDs are defined by the upstream ilink protocol. TEXT..VIDEO are
// contiguous (1..5), but TOOL_CALL_START/RESULT jump to 11/12; they are
// therefore given explicit values rather than relying on iota, so the wire
// format is self-documenting and stable across reorders.
const (
	MessageItemTypeNone  = 0
	MessageItemTypeText  = 1
	MessageItemTypeImage = 2
	MessageItemTypeVoice = 3
	MessageItemTypeFile  = 4
	MessageItemTypeVideo = 5

	MessageItemTypeToolCallStart  = 11 // TOOL_CALL_START (since openclaw-weixin v2.4.4)
	MessageItemTypeToolCallResult = 12 // TOOL_CALL_RESULT (since openclaw-weixin v2.4.4)
)

// Message state constants.
const (
	MessageStateNew = iota
	MessageStateGenerating
	MessageStateFinish
)

// WeixinMessage represents a message from WeChat API.
//
// MessageID uses LosslessID: the server historically sent it as a raw uint64
// JSON number, but ID fields are migrating to prefixed strings (msg_id is
// already "v1:<digits>" on the wire), so all three lossless fields
// (message_id, msg_id, svr_id) accept both encodings — mirroring the upstream
// plugin's LOSSLESS_ID_FIELDS handling.
type WeixinMessage struct {
	Seq          int64         `json:"seq,omitempty"`
	MessageID    LosslessID    `json:"message_id,omitempty"`
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
	MsgID     LosslessID `json:"msg_id,omitempty"`
	TextItem  *TextItem  `json:"text_item,omitempty"`
	ImageItem *ImageItem `json:"image_item,omitempty"`
	VoiceItem *VoiceItem `json:"voice_item,omitempty"`
	FileItem  *FileItem  `json:"file_item,omitempty"`
	VideoItem *VideoItem `json:"video_item,omitempty"`

	// RefMsg is set when this item quotes/replies to an earlier message
	// (since openclaw-weixin v2.4.9-beta.0). Newer WeChat clients send
	// ID-only quotes (RefMsg.SvrID + optional RefMsg.PartialText) instead of
	// inline quoted content; see RefMessage for details.
	RefMsg *RefMessage `json:"ref_msg,omitempty"`

	// Tool call progress items (since openclaw-weixin v2.4.4).
	// Sent as standalone MessageItems with Type=11/12 to surface AI tool
	// usage to the user in real time.
	ToolCallStartItem  *ToolCallStartItem  `json:"tool_call_start_item,omitempty"`
	ToolCallResultItem *ToolCallResultItem `json:"tool_call_result_item,omitempty"`
}

// RefMessage describes a quoted/replied-to message attached to a MessageItem
// (since openclaw-weixin v2.4.9-beta.0).
//
// Older WeChat clients populate MessageItem/Title with the quoted content
// inline. Newer clients instead send only SvrID (the quoted message's server
// ID) and, optionally, PartialText describing a highlighted sub-range of it.
// This SDK does not maintain a local message-history cache, so ID-only
// quotes cannot be resolved to a body here; see message.ConvertInboundMessage
// and README.md for how this is surfaced to callers.
type RefMessage struct {
	MessageItem *MessageItem `json:"message_item,omitempty"`
	Title       string       `json:"title,omitempty"` // 摘要
	// SvrID is the quoted message's server ID, used when newer clients omit
	// the quoted content. Number or string on the wire (see WeixinMessage.MessageID).
	SvrID LosslessID `json:"svr_id,omitempty"`
	// PartialText describes a selected substring of the quoted message, if any.
	PartialText *PartialText `json:"partial_text,omitempty"`
}

// PartialText describes a highlighted sub-range of a quoted message.
// Resolving it against the quoted message's body requires the quoted body,
// which this SDK does not cache; the field is preserved on the wire for
// callers that maintain their own message history.
type PartialText struct {
	Start      string `json:"start,omitempty"`
	End        string `json:"end,omitempty"`
	StartIndex int    `json:"startindex,omitempty"`
	EndIndex   int    `json:"endindex,omitempty"`
	QuoteMD5   string `json:"quotemd5,omitempty"`
}

// ToolCallStartItem is the payload for a TOOL_CALL_START message item (type 11).
type ToolCallStartItem struct {
	ToolName   string `json:"tool_name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolCallResultItem is the payload for a TOOL_CALL_RESULT message item (type 12).
type ToolCallResultItem struct {
	ToolName   string `json:"tool_name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Status     string `json:"status,omitempty"` // "completed" | "failed" | "blocked" | "unknown"
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

// SendMessageResponse represents the response to a sendMessage call.
// Since openclaw-weixin v2.4.5, sendMessage parses ret/errmsg and fails on
// non-zero ret instead of fire-and-forget.
type SendMessageResponse struct {
	Ret    int32  `json:"ret,omitempty"`
	ErrMsg string `json:"errmsg,omitempty"`
	// MessageID is the server-assigned ID for the sent message (since
	// openclaw-weixin v2.4.9-beta.0). Number or string on the wire; see
	// WeixinMessage.MessageID.
	MessageID LosslessID `json:"message_id,omitempty"`
}

// WeixinMessageWrapper wraps WeixinMessage for sending.
type WeixinMessageWrapper struct {
	FromUserID   string        `json:"from_user_id"`            // Bot ID (sender)
	ToUserID     string        `json:"to_user_id"`              // User ID (recipient)
	ClientID     string        `json:"client_id"`               // Unique client ID
	MessageType  int           `json:"message_type"`            // 2 = BOT
	MessageState int           `json:"message_state"`           // 2 = FINISH
	ContextToken string        `json:"context_token,omitempty"`
	ItemList     []MessageItem `json:"item_list"`
	RunID        string        `json:"run_id,omitempty"` // Correlates all msgs in one reply turn (since v2.4.4)
}

// SendOptions carries per-send metadata shared across all outbound message paths.
// A caller pins one RunID across text + media + tool-progress messages so the
// server can correlate them as a single reply turn.
type SendOptions struct {
	ContextToken string
	RunID        string
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
	Ret              int32  `json:"ret"`
	ErrMsg           string `json:"errmsg,omitempty"`
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

// Typing status constants for SendTyping.
const (
	TypingStatusTyping = 1
	TypingStatusCancel = 2
)

// NotifyStartRequest is sent when the channel client starts.
// proto: NotifyStartReq.
type NotifyStartRequest struct {
	BaseInfo *BaseInfo `json:"base_info,omitempty"`
}

// NotifyStartResponse is the response to a NotifyStart call.
type NotifyStartResponse struct {
	Ret    int32  `json:"ret,omitempty"`
	ErrMsg string `json:"errmsg,omitempty"`
}

// NotifyStopRequest is sent when the channel client stops.
// proto: NotifyStopReq.
type NotifyStopRequest struct {
	BaseInfo *BaseInfo `json:"base_info,omitempty"`
}

// NotifyStopResponse is the response to a NotifyStop call.
type NotifyStopResponse struct {
	Ret    int32  `json:"ret,omitempty"`
	ErrMsg string `json:"errmsg,omitempty"`
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

// BaseInfo is attached to every outgoing CGI request.
//
// BotAgent mirrors HTTP `User-Agent`: a self-declared identifier of the
// upstream bot/app for observability purposes only (not for auth or routing).
// Format: UA-style `Name/Version` tokens, optionally followed by
// `(comment)`. ASCII only, total length <= 256 bytes after sanitization.
// Defaults to "OpenClaw" when unset on the wire.
type BaseInfo struct {
	ChannelVersion string `json:"channel_version,omitempty"`
	BotAgent       string `json:"bot_agent,omitempty"`
}

// QRStatusResponse represents the get_qrcode_status response.
type QRStatusResponse struct {
	Status      string `json:"status,omitempty"` // wait, scaned, confirmed, expired, binded_redirect
	BotToken    string `json:"bot_token,omitempty"`
	IlinkBotID  string `json:"ilink_bot_id,omitempty"`
	BaseURL     string `json:"baseurl,omitempty"`
	IlinkUserID string `json:"ilink_user_id,omitempty"`
}

// QR status values returned by get_qrcode_status.
const (
	// QRStatusWait: still waiting for a scan.
	QRStatusWait = "wait"
	// QRStatusScanned: user scanned but hasn't confirmed yet.
	QRStatusScanned = "scaned"
	// QRStatusConfirmed: login confirmed, credentials returned.
	QRStatusConfirmed = "confirmed"
	// QRStatusExpired: QR code expired, needs refresh.
	QRStatusExpired = "expired"
	// QRStatusBindedRedirect: the scanned bot is already bound to this host;
	// treated as a successful no-op (since openclaw-weixin v2.4.3).
	QRStatusBindedRedirect = "binded_redirect"
)
