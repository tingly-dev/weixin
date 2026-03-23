// package weixin provides the main WeChat channel plugin.
package weixin

import (
	"sync"

	"github.com/tingly-dev/weixin/channel"
)

// tempConfigAdapter is a temporary config adapter used during plugin initialization.
type tempConfigAdapter struct {
	plugin *Plugin
}

// ListAccountIDs returns all configured WeChat account IDs.
func (a *tempConfigAdapter) ListAccountIDs() ([]string, error) {
	return a.plugin.Accounts().ListIDs()
}

// ResolveAccount resolves a WeChat account by ID.
func (a *tempConfigAdapter) ResolveAccount(accountID string) (*channel.Account, error) {
	wcAccount, err := a.plugin.Accounts().Get(accountID)
	if err != nil {
		return nil, &channel.ChannelError{
			Type:    channel.ErrorAccountNotFound,
			Message: "account not found: " + accountID,
			Channel: channel.ChannelIDWeChat,
		}
	}

	return &channel.Account{
		ID:         wcAccount.ID,
		Name:       wcAccount.Name,
		Enabled:    wcAccount.Enabled,
		Configured: wcAccount.Configured,
		Config: map[string]string{
			"botId":   wcAccount.BotID,
			"userId":  wcAccount.UserID,
			"baseUrl": wcAccount.BaseURL,
		},
	}, nil
}

// DefaultAccount returns the default WeChat account ID.
func (a *tempConfigAdapter) DefaultAccount() (string, error) {
	ids, err := a.plugin.Accounts().ListIDs()
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", &channel.ChannelError{
			Type:    channel.ErrorAccountNotFound,
			Message: "no WeChat accounts configured",
			Channel: channel.ChannelIDWeChat,
		}
	}
	return ids[0], nil
}

// IsEnabled checks if a WeChat account is enabled.
func (a *tempConfigAdapter) IsEnabled(accountID string) (bool, error) {
	account, err := a.plugin.Accounts().Get(accountID)
	if err != nil {
		return false, err
	}
	return account.Enabled, nil
}

// IsConfigured checks if a WeChat account is configured.
func (a *tempConfigAdapter) IsConfigured(accountID string) (bool, error) {
	account, err := a.plugin.Accounts().Get(accountID)
	if err != nil {
		return false, err
	}
	return account.Configured, nil
}

// Plugin is the WeChat channel plugin.
type Plugin struct {
	*channel.BasePlugin
	config   *WeChatConfig
	accounts *AccountManager
	running  map[string]bool // accountID -> running
	mu       sync.RWMutex    // protects running map
}

// NewPlugin creates a new WeChat plugin.
func NewPlugin(config *WeChatConfig) *Plugin {
	p := &Plugin{
		config:   config,
		accounts: NewAccountManager(),
		running:  make(map[string]bool),
	}

	// Create base plugin with metadata
	meta := &channel.Meta{
		ID:             channel.ChannelIDWeChat,
		Label:          "WeChat",
		SelectionLabel: "WeChat",
		DetailLabel:    "WeChat",
		Blurb:          "Send and receive messages via WeChat",
		DocsPath:       "/docs/wechat",
		SystemImage:    "message.fill",
		Version:        "1.0.0",
	}

	capabilities := &channel.Capabilities{
		ChatTypes:      []channel.ChatType{channel.ChatTypeDirect},
		Text:           true,
		Media:          true,
		BlockStreaming: true,
	}

	p.BasePlugin = channel.NewBasePlugin(meta, capabilities, &tempConfigAdapter{plugin: p})

	return p
}

// Accounts returns the account manager.
func (p *Plugin) Accounts() *AccountManager {
	return p.accounts
}

// WeChatConfig returns the plugin configuration.
func (p *Plugin) WeChatConfig() *WeChatConfig {
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

// IsRunningByID checks if an account is running (implements PluginInterface).
func (p *Plugin) IsRunningByID(accountID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running[accountID]
}

// IsRunning checks if an account is running (regular method).
func (p *Plugin) IsRunning(accountID string) bool {
	return p.IsRunningByID(accountID)
}

// Config returns the config adapter (overrides BasePlugin.Config).
func (p *Plugin) Config() channel.ConfigAdapter {
	// Return the config adapter from BasePlugin
	// This will be set after InitPlugin is called
	return p.BasePlugin.Config()
}
