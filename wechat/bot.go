// Package wechat provides the WeChat ilink bot implementation.
package wechat

import (
	"context"
	"log"

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
	config          *types.WeChatConfig
	account         *Account
	store           types.AccountStore
	botAgent        string
	lifecycleNotify bool
}

// Option configures a WechatBot.
type Option func(*botOptions)

type botOptions struct {
	baseURL         string
	botType         string
	dataDir         string
	botAgent        string
	lifecycleNotify bool
	store           types.AccountStore
	account         *types.WeChatAccount
}

// WithBaseURL overrides the default API base URL.
func WithBaseURL(url string) Option {
	return func(o *botOptions) { o.baseURL = url }
}

// WithBotAgent sets the BaseInfo.bot_agent value sent on every API request.
// The value is sanitized into UA-style `Name/Version` tokens before being
// transmitted; pass empty to fall back to api.DefaultBotAgent.
func WithBotAgent(agent string) Option {
	return func(o *botOptions) { o.botAgent = agent }
}

// WithLifecycleNotify toggles automatic NotifyStart on Connect and
// NotifyStop on Disconnect. Enabled by default; pass false to take full
// manual control via Client.NotifyStart / Client.NotifyStop.
//
// Notify failures are logged but never returned: lifecycle reporting must
// not block bot startup or shutdown.
func WithLifecycleNotify(enabled bool) Option {
	return func(o *botOptions) { o.lifecycleNotify = enabled }
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
		baseURL:         DefaultBaseURL,
		botType:         defaultBotType,
		lifecycleNotify: true,
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
		config:          config,
		store:           store,
		botAgent:        o.botAgent,
		lifecycleNotify: o.lifecycleNotify,
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
		b.applyBotAgent()
	}

	return b, nil
}

// applyBotAgent propagates the configured bot agent (if any) to the
// current account's API client. Safe to call when no account is loaded.
func (b *WechatBot) applyBotAgent() {
	if b.account == nil || b.botAgent == "" {
		return
	}
	if c := b.account.Client(); c != nil {
		c.SetBotAgent(b.botAgent)
	}
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
	b.applyBotAgent()
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

// Connect activates the bot with a loaded account. WeChat uses an HTTP
// API rather than a persistent connection, so Connect only validates the
// account state and (when WithLifecycleNotify is enabled) sends a
// best-effort NotifyStart so the upstream server knows the channel client
// just came online.
//
// The account must be loaded first via LoadAccount() or by passing
// WithAccount to NewWechatBot.
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

	if b.lifecycleNotify {
		if c := b.account.Client(); c != nil {
			resp, err := c.NotifyStart(ctx)
			switch {
			case err != nil:
				log.Printf("[weixin] notifyStart failed during startup (ignored): %v", err)
			case resp != nil && resp.Ret != 0:
				log.Printf("[weixin] notifyStart: ret=%d errmsg=%q", resp.Ret, resp.ErrMsg)
			}
		}
	}

	return nil
}

// Disconnect deactivates the bot. When WithLifecycleNotify is enabled this
// sends a best-effort NotifyStop on a detached context so the call can
// finish even when the parent context is already cancelled (typical
// Ctrl+C / shutdown path).
func (b *WechatBot) Disconnect() error {
	if b.lifecycleNotify && b.account != nil {
		if c := b.account.Client(); c != nil {
			ctx, cancel := context.WithTimeout(context.Background(), api.DefaultConfigTimeout)
			resp, err := c.NotifyStop(ctx)
			cancel()
			switch {
			case err != nil:
				log.Printf("[weixin] notifyStop failed during shutdown (ignored): %v", err)
			case resp != nil && resp.Ret != 0:
				log.Printf("[weixin] notifyStop: ret=%d errmsg=%q", resp.Ret, resp.ErrMsg)
			}
		}
	}

	b.account = nil
	return nil
}
