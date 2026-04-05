// Package wechat provides the WeChat ilink bot implementation.
package wechat

import (
	"context"

	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/bot"
	"github.com/tingly-dev/weixin/types"
)

// WechatBot is the WeChat ilink bot implementation.
// One bot manages one account with one API client.
type WechatBot struct {
	*bot.BaseBot
	config         *types.WeChatConfig
	account        *Account        // Single account for this bot
	accountManager *AccountManager // For loading/saving accounts
}

// NewWechatBot creates a new WeChat bot.
func NewWechatBot(config *types.WeChatConfig) (*WechatBot, error) {
	return NewWechatBotWithDataDir(config, "")
}

// NewWechatBotWithDataDir creates a new WeChat bot with a custom data directory.
// If dataDir is empty, uses the default ~/.weixin/accounts.
func NewWechatBotWithDataDir(config *types.WeChatConfig, dataDir string) (*WechatBot, error) {
	if config == nil {
		config = &types.WeChatConfig{}
	}

	b := &WechatBot{
		config: config,
	}

	// Create account manager with custom or default directory
	if dataDir != "" {
		b.accountManager = NewAccountManagerWithDir(dataDir)
	} else {
		b.accountManager = NewAccountManager()
	}

	// Create base bot with metadata
	meta := &types.Meta{
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

	b.BaseBot = bot.NewBaseBot(meta, capabilities)

	return b, nil
}

// NewWechatBotWithAccount creates a new WeChat bot with an existing account.
func NewWechatBotWithAccount(config *types.WeChatConfig, wcAccount *types.WeChatAccount) (*WechatBot, error) {
	b, err := NewWechatBot(config)
	if err != nil {
		return nil, err
	}

	// Set account
	b.account = NewAccount(wcAccount)

	return b, nil
}

// LoadAccount loads an account from the account manager by ID.
func (b *WechatBot) LoadAccount(accountID string) error {
	wcAccount, err := b.accountManager.Get(accountID)
	if err != nil {
		return &types.Error{
			Type:    types.ErrorAccountNotFound,
			Message: "account not found: " + accountID,
			Err:     err,
		}
	}

	b.account = NewAccount(wcAccount)
	return nil
}

// Account returns the current account.
func (b *WechatBot) Account() *Account {
	return b.account
}

// Client returns the underlying API client.
func (b *WechatBot) Client() *api.Client {
	if b.account == nil {
		return nil
	}
	return b.account.Client()
}

// AccountManager returns the account manager (for loading/saving accounts).
func (b *WechatBot) AccountManager() *AccountManager {
	return b.accountManager
}

// Config returns the bot configuration.
func (b *WechatBot) Config() *types.WeChatConfig {
	return b.config
}

// IsConnected returns whether the bot is connected (account is configured).
func (b *WechatBot) IsConnected() bool {
	return b.account != nil && b.account.IsConfigured()
}

// Connect activates the bot with a loaded account.
// This is a no-op for WeChat as it uses HTTP API, not persistent connections.
// The account must be loaded first via LoadAccount() or NewWechatBotWithAccount().
func (b *WechatBot) Connect(ctx context.Context) error {
	if b.account == nil {
		return &types.Error{
			Type:    types.ErrorAccountNotFound,
			Message: "no account loaded, call LoadAccount() first",
		}
	}
	if !b.account.IsConfigured() {
		return &types.Error{
			Type:    types.ErrorAccountNotFound,
			Message: "account not configured",
		}
	}
	return nil
}

// Disconnect deactivates the bot.
// This is a no-op for WeChat as it uses HTTP API, not persistent connections.
func (b *WechatBot) Disconnect() error {
	b.account = nil
	return nil
}
