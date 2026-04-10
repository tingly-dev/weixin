// Package wechat provides the WeChat ilink bot implementation.
package wechat

import (
	"context"

	"github.com/tingly-dev/weixin/types"
	"github.com/tingly-dev/weixin/wechat/api"
)

// Default WeChat service URLs.
const (
	DefaultBaseURL    = "https://ilinkai.weixin.qq.com"
	DefaultCDNBaseURL = "https://novac2c.cdn.weixin.qq.com/c2c"
)

// WechatBot is the WeChat ilink bot implementation.
// One bot manages one account with one API client.
type WechatBot struct {
	*types.BaseBot
	config  *types.WeChatConfig
	account *Account
	store   types.AccountStore
}

// Option configures a WechatBot.
type Option func(*botOptions)

type botOptions struct {
	baseURL  string
	botType  string
	dataDir  string
	store    types.AccountStore
	account  *types.WeChatAccount
}

// WithBaseURL overrides the default API base URL.
func WithBaseURL(url string) Option {
	return func(o *botOptions) { o.baseURL = url }
}

// WithDataDir sets a custom directory for account persistence.
func WithDataDir(dir string) Option {
	return func(o *botOptions) { o.dataDir = dir }
}

// WithStore sets a custom account store (overrides WithDataDir).
func WithStore(store types.AccountStore) Option {
	return func(o *botOptions) { o.store = store }
}

// WithAccount sets a pre-configured account (skips store/login).
func WithAccount(account *types.WeChatAccount) Option {
	return func(o *botOptions) { o.account = account }
}

// NewWechatBot creates a WeChat bot. All settings have sensible defaults.
//
// Examples:
//
//	bot, err := wechat.NewWechatBot()                          // all defaults
//	bot, err := wechat.NewWechatBot(wechat.WithDataDir("."))   // custom data dir
//	bot, err := wechat.NewWechatBot(wechat.WithAccount(acct))  // existing account
func NewWechatBot(opts ...Option) (*WechatBot, error) {
	o := &botOptions{
		baseURL: DefaultBaseURL,
		botType: defaultBotType,
	}
	for _, opt := range opts {
		opt(o)
	}

	// Resolve store
	var store types.AccountStore
	if o.store != nil {
		store = o.store
	} else if o.account != nil {
		store = NewNoopStore()
	} else if o.dataDir != "" {
		store = NewAccountManagerWithDir(o.dataDir)
	} else {
		store = NewAccountManager()
	}

	config := &types.WeChatConfig{
		BaseURL: o.baseURL,
		BotType: o.botType,
	}

	b := &WechatBot{
		config: config,
		store:  store,
	}

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
	b.BaseBot = types.NewBaseBot(meta, capabilities)

	if o.account != nil {
		b.account = NewAccount(o.account)
	}

	return b, nil
}

// LoadAccount loads an account from the store by ID.
func (b *WechatBot) LoadAccount(accountID string) error {
	wcAccount, err := b.store.Get(accountID)
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

// SaveAccount saves the current account to the store.
func (b *WechatBot) SaveAccount(account *types.WeChatAccount) error {
	return b.store.Save(account)
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

// Store returns the account store (for loading/saving accounts).
func (b *WechatBot) Store() types.AccountStore {
	return b.store
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
