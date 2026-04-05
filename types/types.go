// Package types provides shared types for the WeChat/WeCom SDK.
package types

import (
	"context"
	"time"
)

// ChatType represents the type of conversation.
type ChatType string

const (
	ChatTypeDirect ChatType = "direct"
	ChatTypeGroup  ChatType = "group"
)

// Meta represents metadata about a bot.
type Meta struct {
	Label          string `json:"label"`
	SelectionLabel string `json:"selectionLabel"`
	DetailLabel    string `json:"detailLabel,omitempty"`
	Blurb          string `json:"blurb"`
	DocsPath       string `json:"docsPath"`
	SystemImage    string `json:"systemImage,omitempty"`
	Version        string `json:"version,omitempty"`
}

// Capabilities describes what a platform supports.
type Capabilities struct {
	// ChatTypes is the list of supported conversation types.
	ChatTypes []ChatType `json:"chatTypes"`

	// Text message support
	Text bool `json:"text"`

	// Rich message support
	Media     bool `json:"media"`
	Reactions bool `json:"reactions"`
	Edit      bool `json:"edit"`
	Unsend    bool `json:"unsend"`
	Threads   bool `json:"threads"`
	Polls     bool `json:"polls"`

	// Other features
	NativeCommands  bool `json:"nativeCommands"`
	BlockStreaming  bool `json:"blockStreaming"`
	Streaming       bool `json:"streaming"`
	GroupManagement bool `json:"groupManagement"`
	Effects         bool `json:"effects"`
}

// Message represents an incoming message.
type Message struct {
	// Metadata
	MessageID string    `json:"messageId"`
	AccountID string    `json:"accountId,omitempty"`
	ChatType  ChatType  `json:"chatType"`
	Timestamp time.Time `json:"timestamp"`

	// Content
	Text         string       `json:"text"`
	OriginalText string       `json:"originalText,omitempty"`
	Attachments  []Attachment `json:"attachments,omitempty"`

	// Sender
	From         string `json:"from"`
	SenderID     string `json:"senderId,omitempty"`
	SenderName   string `json:"senderName,omitempty"`
	SenderHandle string `json:"senderHandle,omitempty"`

	// Receiver
	To string `json:"to"`

	// Threading
	ThreadID  string `json:"threadId,omitempty"`
	ReplyToID string `json:"replyToId,omitempty"`
	ParentID  string `json:"parentId,omitempty"`

	// Session Context
	// ContextToken is used by some platforms (e.g., WeChat) to maintain
	// conversation context when sending replies. Must be passed back
	// when responding to this message.
	ContextToken string `json:"contextToken,omitempty"`

	// Streaming
	// StreamToken is an opaque token for initiating streaming replies.
	// Set by the adapter for platforms that support streaming.
	StreamToken string `json:"streamToken,omitempty"`

	// Additional context
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	NativeEvent interface{}            `json:"nativeEvent,omitempty"`
}

// Attachment represents a media attachment.
type Attachment struct {
	URL         string `json:"url,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	Size        int64  `json:"size,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

// OutboundMessage represents a message to be sent.
type OutboundMessage struct {
	// Target
	AccountID string `json:"accountId,omitempty"`
	To        string `json:"to"`

	// Content
	Text        string `json:"text"`
	MediaURL    string `json:"mediaUrl,omitempty"`
	MediaData   []byte `json:"mediaData,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	FilePath    string `json:"filePath,omitempty"` // Path to local file for upload

	// Threading
	ThreadID  string `json:"threadId,omitempty"`
	ReplyToID string `json:"replyToId,omitempty"`

	// Session Context
	// ContextToken should be set from the incoming message's ContextToken
	// when replying, for platforms that require it (e.g., WeChat).
	ContextToken string `json:"contextToken,omitempty"`

	// Streaming
	// StreamID identifies an ongoing stream. Empty on first chunk (adapter generates one).
	// Reuse the returned StreamID for subsequent chunks.
	StreamID     string `json:"streamId,omitempty"`
	StreamFinish bool   `json:"streamFinish,omitempty"`

	// Options
	Silent         bool   `json:"silent,omitempty"`
	ParseMode      string `json:"parseMode,omitempty"`
	DisablePreview bool   `json:"disablePreview,omitempty"`
	ForceDocument  bool   `json:"forceDocument,omitempty"`
	GifPlayback    bool   `json:"gifPlayback,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// OutboundResult represents the result of sending a message.
type OutboundResult struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
	MessageID string `json:"messageId,omitempty"`

	// Channel-specific IDs
	ChannelMessageID string `json:"channelMessageId,omitempty"`
	ChannelThreadID  string `json:"channelThreadId,omitempty"`
}

// Account represents a configured account.
type Account struct {
	ID         string            `json:"id"`
	Name       string            `json:"name,omitempty"`
	Enabled    bool              `json:"enabled"`
	Configured bool              `json:"configured"`
	Connected  bool              `json:"connected"`
	Config     map[string]string `json:"config,omitempty"`
}

// AccountStatus represents the runtime status of an account.
type AccountStatus struct {
	AccountID       string    `json:"accountId"`
	Running         bool      `json:"running"`
	Connected       bool      `json:"connected"`
	LastConnectedAt time.Time `json:"lastConnectedAt,omitempty"`
	LastMessageAt   time.Time `json:"lastMessageAt,omitempty"`
	LastError       string    `json:"lastError,omitempty"`
	RestartPending  bool      `json:"restartPending"`
}

// DirectoryEntry represents an entry in the directory.
type DirectoryEntry struct {
	Kind      DirectoryEntryKind `json:"kind"`
	ID        string             `json:"id"`
	Name      string             `json:"name,omitempty"`
	Handle    string             `json:"handle,omitempty"`
	AvatarURL string             `json:"avatarUrl,omitempty"`
	RawData   interface{}        `json:"raw,omitempty"`
}

// DirectoryEntryKind is the type of directory entry.
type DirectoryEntryKind string

const (
	DirectoryEntryUser  DirectoryEntryKind = "user"
	DirectoryEntryGroup DirectoryEntryKind = "group"
)

// Reaction represents a message reaction.
type Reaction struct {
	Emoji     string `json:"emoji"`
	MessageID string `json:"messageId"`
}

// EventHandler handles events.
type EventHandler interface {
	// OnMessage is called when a new message is received.
	OnMessage(ctx context.Context, msg *Message) error

	// OnReaction is called when a reaction is added/removed.
	OnReaction(ctx context.Context, reaction *Reaction) error

	// OnEdit is called when a message is edited.
	OnEdit(ctx context.Context, msg *Message) error

	// OnEvent is called when a protocol lifecycle event occurs.
	// EventType examples: "enter_chat", "disconnected", "card_click", "session_change"
	// Payload carries protocol-specific data — the framework does not define its schema.
	OnEvent(ctx context.Context, event *Event)

	// OnError is called when an error occurs.
	OnError(ctx context.Context, err error)
}

// Event represents a protocol lifecycle event.
type Event struct {
	EventType string                 `json:"eventType"`
	AccountID string                 `json:"accountId,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

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

// Upload media type constants.
const (
	UploadMediaTypeImage = iota + 1
	UploadMediaTypeVideo
	UploadMediaTypeFile
	UploadMediaTypeVoice
)

// ErrorType identifies a category of error.
type ErrorType string

const (
	ErrorAccountNotFound ErrorType = "account_not_found"
	ErrorNotSupported    ErrorType = "not_supported"
)

// Error represents a bot-related error.
type Error struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// QrCodeStartResult contains the QR code information for login.
type QrCodeStartResult struct {
	QrCodeID   string `json:"qrcodeId"`   // Unique ID for this QR code session
	QrCodeURL  string `json:"qrcodeUrl"`  // URL to the QR code image
	QrCodeData string `json:"qrcodeData"` // QR code data (base64 or text format)
	ExpiresIn  int    `json:"expiresIn"`  // Seconds until QR code expires
}

// QrCodeWaitResult contains the login result after QR code scan.
type QrCodeWaitResult struct {
	Success   bool   `json:"success"`   // True if login succeeded
	BotToken  string `json:"botToken"`  // Authentication token
	AccountID string `json:"accountId"` // Account ID
	BaseURL   string `json:"baseUrl"`   // Base URL for API requests
	UserID    string `json:"userId"`    // User ID
	Error     string `json:"error"`     // Error message if failed
}

// UploadURLRequest contains parameters for getting an upload URL.
type UploadURLRequest struct {
	FileKey   string `json:"filekey"`    // Unique file identifier
	MediaType int    `json:"media_type"` // 1=IMAGE, 2=VIDEO, 3=AUDIO, 4=FILE
	RawSize   int64  `json:"rawsize"`    // Original file size in bytes
	RawMD5    string `json:"rawfilemd5"` // MD5 hash of original file
	FileSize  int64  `json:"filesize"`   // Encrypted file size in bytes
	AESKey    string `json:"aeskey"`     // Base64-encoded AES key (if encryption used)
	FileName  string `json:"filename"`   // Original filename
}

// UploadURLResult contains the upload URL and related parameters.
type UploadURLResult struct {
	UploadParam string `json:"upload_param"` // CDN upload URL
	FileKey     string `json:"filekey"`      // File identifier
	AuthToken   string `json:"auth_token"`   // Authorization token for upload
}

// MediaUploadRequest contains parameters for uploading a media file.
type MediaUploadRequest struct {
	FilePath   string `json:"filepath"`   // Path to local file
	FileName   string `json:"filename"`   // Original filename
	MediaType  string `json:"mediaType"`  // "image", "video", "audio", "file"
	EncryptKey []byte `json:"encryptKey"` // AES key for encryption (nil = no encryption)
}

// MediaUploadResult contains the result of a media upload.
type MediaUploadResult struct {
	FileKey      string `json:"filekey"`      // File identifier
	FileSize     int64  `json:"filesize"`     // Original file size
	UploadURL    string `json:"uploadUrl"`    // CDN URL
	EncryptKey   []byte `json:"encryptKey"`   // Encryption key used
	EncryptQuery string `json:"encryptQuery"` // Query param for decryption
}

// GetUpdatesRequest contains parameters for long-polling getUpdates.
type GetUpdatesRequest struct {
	AccountID string `json:"accountId"` // Account identifier
	SyncBuf   string `json:"syncBuf"`   // Current sync buffer (cursor)
}

// GetUpdatesResult contains the result of a getUpdates call.
type GetUpdatesResult struct {
	Messages           []*Message `json:"messages"`           // New messages
	SyncBuf            string     `json:"syncBuf"`            // New sync buffer (next cursor)
	LongPollingTimeout int        `json:"longpollingTimeout"` // Suggested timeout for next request
	ErrCode            int        `json:"errcode"`            // Error code (0 = success)
	ErrMsg             string     `json:"errmsg"`             // Error message
}
