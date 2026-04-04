// Package wecom provides the WeCom AI Bot implementation.
package wecom

import (
	"log"

	"github.com/tingly-dev/weixin/types"
	"github.com/tingly-dev/weixin/wechat"
)

// WecomConfig holds WeCom-specific configuration.
type WecomConfig struct {
	// Logger for logging output (nil for silent)
	Logger *log.Logger
}

// WecomBot is the WeCom AI Bot.
type WecomBot struct {
	*wechat.BaseBot
	config  *WecomConfig
	gateway *GatewayAdapter
	actions *ActionsAdapter
	upload  *UploadAdapter
}

// NewWecomBot creates a new WeCom bot.
func NewWecomBot(config *WecomConfig) *WecomBot {
	if config == nil {
		config = &WecomConfig{}
	}

	b := &WecomBot{
		config: config,
	}

	// Create gateway with logger
	b.gateway = NewGatewayAdapter(config.Logger)

	// Create adapters
	b.actions = NewActionsAdapter(b.gateway)
	b.upload = NewUploadAdapter(b.gateway)

	// Create base bot with metadata
	meta := &wechat.Meta{
		Label:          "WeCom",
		SelectionLabel: "WeCom (Enterprise WeChat)",
		DetailLabel:    "WeCom AI Bot",
		Blurb:          "WeCom Enterprise WeChat AI Bot integration",
		DocsPath:       "/docs/wecom",
		SystemImage:    "building.2.fill",
		Version:        "1.0.0",
	}

	capabilities := &types.Capabilities{
		ChatTypes: []types.ChatType{types.ChatTypeDirect, types.ChatTypeGroup},
		Text:      true,
		Media:     true,
		Streaming: true,
	}

	b.BaseBot = wechat.NewBasePlugin(meta, capabilities, &wecomConfigAdapter{bot: b})
	b.SetActions(b.actions)
	b.SetGateway(b.gateway)
	b.SetUpload(b.upload)

	return b
}

// Gateway returns the gateway adapter for connection management.
func (b *WecomBot) Gateway() *GatewayAdapter {
	return b.gateway
}

// Actions returns the actions adapter for sending messages.
func (b *WecomBot) Actions() *ActionsAdapter {
	return b.actions
}

// Upload returns the upload adapter for media uploads.
func (b *WecomBot) Upload() *UploadAdapter {
	return b.upload
}

// WecomConfig returns the bot configuration.
func (b *WecomBot) WecomConfig() *WecomConfig {
	return b.config
}

// wecomConfigAdapter implements types.ConfigAdapter for WeCom.
// WeCom uses pre-configured BotID/Secret credentials, not dynamic account provisioning.
type wecomConfigAdapter struct {
	bot *WecomBot
}

// ListAccountIDs returns all configured account IDs from the gateway.
func (a *wecomConfigAdapter) ListAccountIDs() ([]string, error) {
	ids := a.bot.gateway.ListAccountIDs()
	// Convert []string to []string explicitly
	result := make([]string, len(ids))
	copy(result, ids)
	return result, nil
}

// ResolveAccount resolves an account by ID.
func (a *wecomConfigAdapter) ResolveAccount(accountID string) (*types.Account, error) {
	client := a.bot.gateway.GetClient(accountID)
	if client == nil {
		return &types.Account{
			ID:         accountID,
			Enabled:    false,
			Configured: false,
			Connected:  false,
		}, nil
	}

	return &types.Account{
		ID:         accountID,
		Enabled:    true,
		Configured: true,
		Connected:  client.IsConnected(),
	}, nil
}

// DefaultAccount returns the default account ID.
func (a *wecomConfigAdapter) DefaultAccount() (string, error) {
	ids := a.bot.gateway.ListAccountIDs()
	if len(ids) == 0 {
		return "", &wechat.Error{
			Type:    wechat.ErrorAccountNotFound,
			Message: "no accounts configured",
		}
	}
	return ids[0], nil
}

// IsEnabled checks if an account is enabled.
func (a *wecomConfigAdapter) IsEnabled(accountID string) (bool, error) {
	client := a.bot.gateway.GetClient(accountID)
	return client != nil, nil
}

// IsConfigured checks if an account is configured.
func (a *wecomConfigAdapter) IsConfigured(accountID string) (bool, error) {
	client := a.bot.gateway.GetClient(accountID)
	return client != nil, nil
}
