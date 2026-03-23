// Package channel provides core types and interfaces for messaging channels.
//
// This package defines the fundamental abstractions used throughout
// AgentChannel, based on the OpenClaw channel protocol.
package channel

import (
	"context"
	"time"
)

// ChannelID is a unique identifier for a channel type.
type ChannelID string

// Common channel IDs
const (
	ChannelIDTelegram    ChannelID = "telegram"
	ChannelIDDiscord     ChannelID = "discord"
	ChannelIDSlack       ChannelID = "slack"
	ChannelIDSignal      ChannelID = "signal"
	ChannelIDWhatsApp    ChannelID = "whatsapp"
	ChannelIDiMessage    ChannelID = "imessage"
	ChannelIDBlueBubbles ChannelID = "bluebubbles"
	ChannelIDWeChat      ChannelID = "wechat"
	ChannelIDFeishu      ChannelID = "feishu"
	ChannelIDLine        ChannelID = "line"
)

// ChatType represents the type of conversation.
type ChatType string

const (
	ChatTypeDirect  ChatType = "direct"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

// MessageAction represents a type of message action.
type MessageAction string

const (
	ActionSend      MessageAction = "send"
	ActionSendMedia MessageAction = "send_media"
	ActionReact     MessageAction = "react"
	ActionEdit      MessageAction = "edit"
	ActionUnsend    MessageAction = "unsend"
	ActionSendPoll  MessageAction = "send_poll"
)

// Capabilities describes what a channel supports.
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
	GroupManagement bool `json:"groupManagement"`
	Effects         bool `json:"effects"`
}

// SupportsAction checks if the channel supports a specific action.
func (c *Capabilities) SupportsAction(action MessageAction) bool {
	switch action {
	case ActionSend, ActionSendMedia:
		return c.Text || c.Media
	case ActionReact:
		return c.Reactions
	case ActionEdit:
		return c.Edit
	case ActionUnsend:
		return c.Unsend
	case ActionSendPoll:
		return c.Polls
	default:
		return false
	}
}

// Message represents an incoming message from a channel.
type Message struct {
	// Metadata
	MessageID string    `json:"messageId"`
	ChannelID ChannelID `json:"channel"`
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
	// ContextToken is used by some channels (e.g., WeChat) to maintain
	// conversation context when sending replies. Must be passed back
	// when responding to this message.
	ContextToken string `json:"contextToken,omitempty"`

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

// OutboundMessage represents a message to be sent to a channel.
type OutboundMessage struct {
	// Target
	ChannelID ChannelID `json:"channelId"`
	AccountID string    `json:"accountId,omitempty"`
	To        string    `json:"to"`

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
	// when replying, for channels that require it (e.g., WeChat).
	ContextToken string `json:"contextToken,omitempty"`

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

// Account represents a configured channel account.
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

// MessageContext provides context for message handlers.
type MessageContext struct {
	// Message details
	Message *Message

	// Routing information
	SessionKey string `json:"sessionKey"`
	AgentID    string `json:"agentId,omitempty"`

	// Additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DirectoryEntry represents an entry in the channel directory.
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
	DirectoryEntryUser    DirectoryEntryKind = "user"
	DirectoryEntryGroup   DirectoryEntryKind = "group"
	DirectoryEntryChannel DirectoryEntryKind = "channel"
)

// Reaction represents a message reaction.
type Reaction struct {
	Emoji     string `json:"emoji"`
	MessageID string `json:"messageId"`
}

// Poll represents a poll message.
type Poll struct {
	Question       string   `json:"question"`
	Options        []string `json:"options"`
	IsAnonymous    bool     `json:"isAnonymous"`
	MultipleChoice bool     `json:"multipleChoice"`
}

// EventHandler handles events from a channel.
type EventHandler interface {
	// OnMessage is called when a new message is received.
	OnMessage(ctx context.Context, msg *Message) error

	// OnReaction is called when a reaction is added/removed.
	OnReaction(ctx context.Context, reaction *Reaction) error

	// OnEdit is called when a message is edited.
	OnEdit(ctx context.Context, msg *Message) error

	// OnError is called when an error occurs.
	OnError(ctx context.Context, err error)
}
