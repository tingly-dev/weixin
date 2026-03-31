// Package weixin provides a WeChat messaging SDK.
//
// This plugin implements the WeChat messaging protocol, supporting:
// - QR code login for account authorization
// - Long-polling for message synchronization
// - Text and media message sending
// - AES-128-ECB encrypted CDN media uploads
package weixin

import (
	"time"

	"github.com/tingly-dev/weixin/types"
)

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

// Re-export types from types package for backward compatibility.
type (
	ChatType           = types.ChatType
	Capabilities       = types.Capabilities
	Message            = types.Message
	Attachment         = types.Attachment
	OutboundMessage    = types.OutboundMessage
	OutboundResult     = types.OutboundResult
	Account            = types.Account
	AccountStatus      = types.AccountStatus
	DirectoryEntry     = types.DirectoryEntry
	DirectoryEntryKind = types.DirectoryEntryKind
	Reaction           = types.Reaction
	EventHandler       = types.EventHandler
	Event              = types.Event
)

// Re-export constants for backward compatibility.
const (
	ChatTypeDirect      = types.ChatTypeDirect
	ChatTypeGroup       = types.ChatTypeGroup
	DirectoryEntryUser  = types.DirectoryEntryUser
	DirectoryEntryGroup = types.DirectoryEntryGroup
)
