// Package weixin provides adapter interfaces for WeChat/WeCom implementations.
package types

import (
	"context"
)

// ConfigAdapter handles account configuration.
type ConfigAdapter interface {
	// ListAccountIDs returns all configured account IDs.
	ListAccountIDs() ([]string, error)

	// ResolveAccount resolves an account by ID.
	ResolveAccount(accountID string) (*Account, error)

	// DefaultAccount returns the default account ID.
	DefaultAccount() (string, error)

	// IsEnabled checks if an account is enabled.
	IsEnabled(accountID string) (bool, error)

	// IsConfigured checks if an account is configured.
	IsConfigured(accountID string) (bool, error)
}

// ActionsAdapter handles message actions.
type ActionsAdapter interface {
	// Send sends a text message.
	Send(ctx context.Context, msg *OutboundMessage) (*OutboundResult, error)

	// SendStream sends a streaming text chunk.
	// msg.StreamID identifies the ongoing stream (empty on first chunk, adapter generates).
	// msg.StreamFinish = true marks the final chunk.
	// Platforms that don't support streaming return an ErrorNotSupported error.
	SendStream(ctx context.Context, msg *OutboundMessage) (*OutboundResult, error)

	// SendMedia sends a media message.
	SendMedia(ctx context.Context, msg *OutboundMessage) (*OutboundResult, error)

	// React sends a reaction to a message.
	React(ctx context.Context, reaction *Reaction) (*OutboundResult, error)

	// Edit edits a message.
	Edit(ctx context.Context, messageID string, text string) (*OutboundResult, error)

	// Unsend unsends (deletes) a message.
	Unsend(ctx context.Context, messageID string) (*OutboundResult, error)
}

// GatewayAdapter handles the gateway lifecycle.
type GatewayAdapter interface {
	// StartAccount starts listening for messages for an account.
	StartAccount(ctx context.Context, accountID string) error

	// StopAccount stops listening for messages for an account.
	StopAccount(ctx context.Context, accountID string) error

	// IsRunning checks if an account is running.
	IsRunning(accountID string) bool
}

// PairingAdapter handles device pairing flows.
type PairingAdapter interface {
	// IDLabel returns the label used to store the paired user ID.
	IDLabel() string

	// NormalizeAllowEntry normalizes an entry for the allowlist.
	NormalizeAllowEntry(entry string) string

	// NotifyApproval sends a notification that pairing was approved.
	NotifyApproval(ctx context.Context, id string, message string) error

	// LoginWithQrStart initiates QR code login flow.
	// Returns QR code information that the user should scan.
	LoginWithQrStart(ctx context.Context, accountID string) (*QrCodeStartResult, error)

	// LoginWithQrWait waits for QR code scan confirmation.
	// Should be called after LoginWithQrStart with the returned QrCodeID.
	LoginWithQrWait(ctx context.Context, accountID, qrID string) (*QrCodeWaitResult, error)
}

// UploadAdapter handles media file uploads to external CDNs.
type UploadAdapter interface {
	// GetUploadURL retrieves a pre-signed URL for uploading media.
	GetUploadURL(ctx context.Context, req *UploadURLRequest) (*UploadURLResult, error)

	// UploadMedia uploads a media file and returns the reference.
	UploadMedia(ctx context.Context, req *MediaUploadRequest) (*MediaUploadResult, error)
}

// LongPollAdapter handles long-polling message synchronization.
type LongPollAdapter interface {
	// GetUpdates fetches new messages using long-polling.
	// Returns messages, new sync buffer, and any error.
	GetUpdates(ctx context.Context, req *GetUpdatesRequest) (*GetUpdatesResult, error)
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
