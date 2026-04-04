// package weixin provides the main WeChat bot.
package wechat

import (
	"sync"

	"github.com/tingly-dev/weixin/types"
)

// WechatBot is the WeChat bot.
type WechatBot struct {
	*BaseBot
	config   *types.WeChatConfig
	accounts *AccountManager
	running  map[string]bool // accountID -> running
	mu       sync.RWMutex    // protects running map
}

// NewWeixinBot creates a new WeChat bot.
func NewWeixinBot(config *types.WeChatConfig) *WechatBot {
	return NewWechatBotWithDataDir(config, "")
}

// NewWechatBotWithDataDir creates a new WeChat bot with a custom data directory.
// If dataDir is empty, uses the default ~/.weixin/accounts.
func NewWechatBotWithDataDir(config *types.WeChatConfig, dataDir string) *WechatBot {
	p := &WechatBot{
		config:  config,
		running: make(map[string]bool),
	}

	// Create account manager with custom or default directory
	if dataDir != "" {
		p.accounts = NewAccountManagerWithDir(dataDir)
	} else {
		p.accounts = NewAccountManager()
	}

	// Create base bot with metadata
	meta := &Meta{
		Label:          "WeChat",
		SelectionLabel: "WeChat",
		DetailLabel:    "WeChat",
		Blurb:          "Send and receive messages via WeChat",
		DocsPath:       "/docs/wechat",
		SystemImage:    "message.fill",
		Version:        "1.0.0",
	}

	capabilities := &types.Capabilities{
		ChatTypes:      []types.ChatType{types.ChatTypeDirect},
		Text:           true,
		Media:          true,
		BlockStreaming: true,
	}

	p.BaseBot = NewBasePlugin(meta, capabilities, &ConfigAdapter{Bot: p})
	p.SetActions(NewActionsAdapter(p))
	p.SetGateway(NewGatewayAdapter(p))
	p.SetLongPoll(NewLongPollAdapter(p))
	p.SetUpload(NewUploadAdapter(p))
	p.SetPairing(NewPairingAdapter(p))

	return p
}

// Accounts returns the account manager.
func (p *WechatBot) Accounts() *AccountManager {
	return p.accounts
}

// WeChatConfig returns the bot configuration.
func (p *WechatBot) WeChatConfig() *types.WeChatConfig {
	return p.config
}

// SetRunning sets the running state for an account.
func (p *WechatBot) SetRunning(accountID string, running bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if running {
		p.running[accountID] = true
	} else {
		delete(p.running, accountID)
	}
}

// IsRunningByID checks if an account is running.
func (p *WechatBot) IsRunningByID(accountID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running[accountID]
}

// Meta represents metadata about a bot.
type Meta struct {
	Label string `json:"label"`

	SelectionLabel string `json:"selectionLabel"`
	DetailLabel    string `json:"detailLabel,omitempty"`
	DocsPath       string `json:"docsPath"`
	DocsLabel      string `json:"docsLabel,omitempty"`
	Blurb          string `json:"blurb"`

	Order       int      `json:"order,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	SystemImage string   `json:"systemImage,omitempty"`
	Version     string   `json:"version,omitempty"`
}

// BaseBot provides a default implementation of the WechatBot interface.
type BaseBot struct {
	meta         *Meta
	capabilities *types.Capabilities
	config       types.ConfigAdapter
	actions      types.ActionsAdapter
	gateway      types.GatewayAdapter
	pairing      types.PairingAdapter
	upload       types.UploadAdapter
	longPoll     types.LongPollAdapter
}

// NewBasePlugin creates a new base bot.
func NewBasePlugin(meta *Meta, capabilities *types.Capabilities, config types.ConfigAdapter) *BaseBot {
	return &BaseBot{
		meta:         meta,
		capabilities: capabilities,
		config:       config,
	}
}

// Meta returns the bot metadata.
func (p *BaseBot) Meta() *Meta { return p.meta }

// Capabilities returns the bot capabilities.
func (p *BaseBot) Capabilities() *types.Capabilities { return p.capabilities }

// Config returns the config adapter.
func (p *BaseBot) Config() types.ConfigAdapter { return p.config }

// Actions returns the actions adapter.
func (p *BaseBot) Actions() types.ActionsAdapter { return p.actions }

// Gateway returns the gateway adapter.
func (p *BaseBot) Gateway() types.GatewayAdapter { return p.gateway }

// Pairing returns the pairing adapter.
func (p *BaseBot) Pairing() types.PairingAdapter { return p.pairing }

// Upload returns the upload adapter.
func (p *BaseBot) Upload() types.UploadAdapter { return p.upload }

// LongPoll returns the long-polling adapter.
func (p *BaseBot) LongPoll() types.LongPollAdapter { return p.longPoll }

// SetActions sets the actions adapter.
func (p *BaseBot) SetActions(actions types.ActionsAdapter) { p.actions = actions }

// SetGateway sets the gateway adapter.
func (p *BaseBot) SetGateway(gateway types.GatewayAdapter) { p.gateway = gateway }

// SetPairing sets the pairing adapter.
func (p *BaseBot) SetPairing(pairing types.PairingAdapter) { p.pairing = pairing }

// SetUpload sets the upload adapter.
func (p *BaseBot) SetUpload(upload types.UploadAdapter) { p.upload = upload }

// SetLongPoll sets the long-polling adapter.
func (p *BaseBot) SetLongPoll(longPoll types.LongPollAdapter) { p.longPoll = longPoll }

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

func (e *Error) Unwrap() error { return e.Err }
