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
