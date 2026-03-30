// Package wecom implements the WeCom AI Bot WebSocket protocol adapter.
package wecom

import "time"

// ---------------------------------------------------------------------------
// Protocol command constants
// ---------------------------------------------------------------------------

const (
	CmdSubscribe         = "aibot_subscribe"
	CmdHeartbeat         = "ping"
	CmdResponse          = "aibot_respond_msg"
	CmdResponseWelcome   = "aibot_respond_welcome_msg"
	CmdResponseUpdate    = "aibot_respond_update_msg"
	CmdSendMsg           = "aibot_send_msg"
	CmdUploadMediaInit   = "aibot_upload_media_init"
	CmdUploadMediaChunk  = "aibot_upload_media_chunk"
	CmdUploadMediaFinish = "aibot_upload_media_finish"

	CmdCallback      = "aibot_msg_callback"
	CmdEventCallback = "aibot_event_callback"
)

// ---------------------------------------------------------------------------
// Message type constants
// ---------------------------------------------------------------------------

const (
	MsgTypeText           = "text"
	MsgTypeImage          = "image"
	MsgTypeMixed          = "mixed"
	MsgTypeVoice          = "voice"
	MsgTypeFile           = "file"
	MsgTypeVideo          = "video"
	MsgTypeStream         = "stream"
	MsgTypeMarkdown       = "markdown"
	MsgTypeTemplateCard   = "template_card"
	MsgTypeStreamWithCard = "stream_with_template_card"
)

// ---------------------------------------------------------------------------
// Event type constants
// ---------------------------------------------------------------------------

const (
	EventEnterChat    = "enter_chat"
	EventCardClick    = "template_card_event"
	EventFeedback     = "feedback_event"
	EventDisconnected = "disconnected_event"
)

// ---------------------------------------------------------------------------
// Wire frame (all WS communication uses this JSON structure)
// ---------------------------------------------------------------------------

// WsFrame is the universal frame format for WeCom AI Bot WebSocket protocol.
// All frames (in both directions) use this structure.
type WsFrame struct {
	Cmd     string         `json:"cmd,omitempty"`
	Headers WsFrameHeaders `json:"headers"`
	Body    interface{}    `json:"body,omitempty"`
	ErrCode int            `json:"errcode,omitempty"`
	ErrMsg  string         `json:"errmsg,omitempty"`
}

// WsFrameHeaders contains the per-frame routing header.
type WsFrameHeaders struct {
	ReqID string `json:"req_id"`
}

// ---------------------------------------------------------------------------
// Incoming message body
// ---------------------------------------------------------------------------

// IncomingMessage is the parsed body of an aibot_msg_callback frame.
type IncomingMessage struct {
	MsgID       string    `json:"msgid"`
	AIBotID     string    `json:"aibotid"`
	ChatID      string    `json:"chatid,omitempty"`
	ChatType    string    `json:"chattype,omitempty"` // "single" | "group"
	From        MsgFrom   `json:"from"`
	CreateTime  int64     `json:"create_time,omitempty"`
	ResponseURL string    `json:"response_url,omitempty"`
	MsgType     string    `json:"msgtype"`
	Quote       *MsgQuote `json:"quote,omitempty"`

	Text  *TextContent  `json:"text,omitempty"`
	Image *ImageContent `json:"image,omitempty"`
	Mixed *MixedContent `json:"mixed,omitempty"`
	Voice *VoiceContent `json:"voice,omitempty"`
	File  *FileContent  `json:"file,omitempty"`
	Video *VideoContent `json:"video,omitempty"`
}

// MsgFrom identifies the message sender.
type MsgFrom struct {
	UserID string `json:"userid"`
	CorpID string `json:"corpid,omitempty"`
}

// TextContent holds a text message body.
type TextContent struct {
	Content string `json:"content"`
}

// ImageContent holds an image message body.
type ImageContent struct {
	URL    string `json:"url"`
	AESKey string `json:"aeskey,omitempty"`
}

// MixedContent holds a mixed message body (multiple items).
type MixedContent struct {
	Items []MixedItem `json:"msg_item"`
}

// MixedItem is a single item within a mixed message.
type MixedItem struct {
	MsgType string        `json:"msgtype"`
	Text    *TextContent  `json:"text,omitempty"`
	Image   *ImageContent `json:"image,omitempty"`
}

// VoiceContent holds a voice message body.
type VoiceContent struct {
	Content string `json:"content"` // speech-to-text
}

// FileContent holds a file message body.
type FileContent struct {
	URL    string `json:"url"`
	AESKey string `json:"aeskey,omitempty"`
}

// VideoContent holds a video message body.
type VideoContent struct {
	URL    string `json:"url"`
	AESKey string `json:"aeskey,omitempty"`
}

// MsgQuote holds a quoted/replied-to message.
type MsgQuote struct {
	MsgType string        `json:"msgtype"`
	Text    *TextContent  `json:"text,omitempty"`
	Image   *ImageContent `json:"image,omitempty"`
	Mixed   *MixedContent `json:"mixed,omitempty"`
	Voice   *VoiceContent `json:"voice,omitempty"`
	File    *FileContent  `json:"file,omitempty"`
}

// ---------------------------------------------------------------------------
// Incoming event body
// ---------------------------------------------------------------------------

// IncomingEvent is the parsed body of an aibot_event_callback frame.
type IncomingEvent struct {
	MsgID      string      `json:"msgid"`
	AIBotID    string      `json:"aibotid"`
	ChatID     string      `json:"chatid,omitempty"`
	ChatType   string      `json:"chattype,omitempty"`
	From       MsgFrom     `json:"from"`
	CreateTime int64       `json:"create_time,omitempty"`
	MsgType    string      `json:"msgtype"` // always "event"
	Event      EventDetail `json:"event"`
}

// EventDetail holds the event-specific data.
type EventDetail struct {
	EventType string `json:"eventtype"`
	EventKey  string `json:"event_key,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
}

// ---------------------------------------------------------------------------
// Outgoing reply bodies
// ---------------------------------------------------------------------------

// StreamBody is the body for streaming text replies.
type StreamBody struct {
	ID       string       `json:"id"`
	Finish   bool         `json:"finish,omitempty"`
	Content  string       `json:"content,omitempty"`
	MsgItems []StreamItem `json:"msg_item,omitempty"`
	Feedback *Feedback    `json:"feedback,omitempty"`
}

// StreamItem is a media item within a streaming reply (only on finish=true).
type StreamItem struct {
	MsgType string       `json:"msgtype"`
	Image   *StreamImage `json:"image,omitempty"`
}

// StreamImage is a base64-encoded image in a stream reply.
type StreamImage struct {
	Base64 string `json:"base64"`
	MD5    string `json:"md5"`
}

// Feedback sets a feedback button on the first stream reply.
type Feedback struct {
	ID string `json:"id"`
}

// MediaReplyBody is the body for media replies (passive).
type MediaReplyBody struct {
	MediaID string `json:"media_id"`
}

// VideoReplyBody extends media reply with optional metadata.
type VideoReplyBody struct {
	MediaID     string `json:"media_id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// MarkdownBody is the body for proactive markdown messages.
type MarkdownBody struct {
	Content string `json:"content"`
}

// ---------------------------------------------------------------------------
// Upload types
// ---------------------------------------------------------------------------

// UploadInitBody is the body for the upload init step.
type UploadInitBody struct {
	Type        string `json:"type"` // "file" | "image" | "voice" | "video"
	Filename    string `json:"filename"`
	TotalSize   int64  `json:"total_size"`
	TotalChunks int    `json:"total_chunks"`
	MD5         string `json:"md5,omitempty"`
}

// UploadInitResult is the response from the upload init step.
type UploadInitResult struct {
	UploadID string `json:"upload_id"`
}

// UploadChunkBody is the body for each chunk step.
type UploadChunkBody struct {
	UploadID   string `json:"upload_id"`
	ChunkIndex int    `json:"chunk_index"`
	Base64Data string `json:"base64_data"`
}

// UploadFinishBody is the body for the upload finish step.
type UploadFinishBody struct {
	UploadID string `json:"upload_id"`
}

// UploadFinishResult is the response from the upload finish step.
type UploadFinishResult struct {
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt string `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Config defaults
// ---------------------------------------------------------------------------

const (
	DefaultWsURL                = "wss://openws.work.weixin.qq.com"
	DefaultHeartbeatInterval    = 30 * time.Second
	DefaultReconnectBaseDelay   = 1 * time.Second
	DefaultReconnectMaxDelay    = 30 * time.Second
	DefaultMaxReconnectAttempts = 10
	DefaultMaxAuthFailures      = 5
	DefaultReplyAckTimeout      = 5 * time.Second
	DefaultMaxReplyQueueSize    = 500
	ChunkSize                   = 512 * 1024 // 512 KB
	MaxChunks                   = 100
	MaxStreamContentBytes       = 20480
)
