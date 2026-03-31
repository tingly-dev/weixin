// package weixin provides the main WeChat plugin.
package plugin

import (
	"sync"

	"github.com/tingly-dev/weixin"
)

// Plugin is the WeChat plugin.
type Plugin struct {
	*BasePlugin
	config   *weixin.WeChatConfig
	accounts *weixin.AccountManager
	running  map[string]bool // accountID -> running
	mu       sync.RWMutex    // protects running map
}

// NewPlugin creates a new WeChat plugin.
func NewPlugin(config *weixin.WeChatConfig) *Plugin {
	return NewPluginWithDataDir(config, "")
}

// NewPluginWithDataDir creates a new WeChat plugin with a custom data directory.
// If dataDir is empty, uses the default ~/.weixin/accounts.
func NewPluginWithDataDir(config *weixin.WeChatConfig, dataDir string) *Plugin {
	p := &Plugin{
		config:  config,
		running: make(map[string]bool),
	}

	// Create account manager with custom or default directory
	if dataDir != "" {
		p.accounts = weixin.NewAccountManagerWithDir(dataDir)
	} else {
		p.accounts = weixin.NewAccountManager()
	}

	// Create base plugin with metadata
	meta := &Meta{
		Label:          "WeChat",
		SelectionLabel: "WeChat",
		DetailLabel:    "WeChat",
		Blurb:          "Send and receive messages via WeChat",
		DocsPath:       "/docs/wechat",
		SystemImage:    "message.fill",
		Version:        "1.0.0",
	}

	capabilities := &weixin.Capabilities{
		ChatTypes:      []weixin.ChatType{weixin.ChatTypeDirect},
		Text:           true,
		Media:          true,
		BlockStreaming: true,
	}

	p.BasePlugin = NewBasePlugin(meta, capabilities, &configAdapter{plugin: p})
	p.SetActions(newActionsAdapter(p))
	p.SetGateway(newGatewayAdapter(p))
	p.SetLongPoll(newLongPollAdapter(p))
	p.SetUpload(newUploadAdapter(p))
	p.SetPairing(newPairingAdapter(p))

	return p
}

// Accounts returns the account manager.
func (p *Plugin) Accounts() *weixin.AccountManager {
	return p.accounts
}

// WeChatConfig returns the plugin configuration.
func (p *Plugin) WeChatConfig() *weixin.WeChatConfig {
	return p.config
}

// SetRunning sets the running state for an account.
func (p *Plugin) SetRunning(accountID string, running bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if running {
		p.running[accountID] = true
	} else {
		delete(p.running, accountID)
	}
}

// IsRunningByID checks if an account is running.
func (p *Plugin) IsRunningByID(accountID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running[accountID]
}

// Meta represents metadata about a plugin.
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

// BasePlugin provides a default implementation of the Plugin interface.
type BasePlugin struct {
	meta         *Meta
	capabilities *weixin.Capabilities
	config       weixin.ConfigAdapter
	actions      weixin.ActionsAdapter
	gateway      weixin.GatewayAdapter
	pairing      weixin.PairingAdapter
	upload       weixin.UploadAdapter
	longPoll     weixin.LongPollAdapter
}

// NewBasePlugin creates a new base plugin.
func NewBasePlugin(meta *Meta, capabilities *weixin.Capabilities, config weixin.ConfigAdapter) *BasePlugin {
	return &BasePlugin{
		meta:         meta,
		capabilities: capabilities,
		config:       config,
	}
}

// Meta returns the plugin metadata.
func (p *BasePlugin) Meta() *Meta { return p.meta }

// Capabilities returns the plugin capabilities.
func (p *BasePlugin) Capabilities() *weixin.Capabilities { return p.capabilities }

// Config returns the config adapter.
func (p *BasePlugin) Config() weixin.ConfigAdapter { return p.config }

// Actions returns the actions adapter.
func (p *BasePlugin) Actions() weixin.ActionsAdapter { return p.actions }

// Gateway returns the gateway adapter.
func (p *BasePlugin) Gateway() weixin.GatewayAdapter { return p.gateway }

// Pairing returns the pairing adapter.
func (p *BasePlugin) Pairing() weixin.PairingAdapter { return p.pairing }

// Upload returns the upload adapter.
func (p *BasePlugin) Upload() weixin.UploadAdapter { return p.upload }

// LongPoll returns the long-polling adapter.
func (p *BasePlugin) LongPoll() weixin.LongPollAdapter { return p.longPoll }

// SetActions sets the actions adapter.
func (p *BasePlugin) SetActions(actions weixin.ActionsAdapter) { p.actions = actions }

// SetGateway sets the gateway adapter.
func (p *BasePlugin) SetGateway(gateway weixin.GatewayAdapter) { p.gateway = gateway }

// SetPairing sets the pairing adapter.
func (p *BasePlugin) SetPairing(pairing weixin.PairingAdapter) { p.pairing = pairing }

// SetUpload sets the upload adapter.
func (p *BasePlugin) SetUpload(upload weixin.UploadAdapter) { p.upload = upload }

// SetLongPoll sets the long-polling adapter.
func (p *BasePlugin) SetLongPoll(longPoll weixin.LongPollAdapter) { p.longPoll = longPoll }

// ErrorType identifies a category of error.
type ErrorType string

const (
	ErrorAccountNotFound ErrorType = "account_not_found"
	ErrorNotSupported    ErrorType = "not_supported"
)

// Error represents a plugin-related error.
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
