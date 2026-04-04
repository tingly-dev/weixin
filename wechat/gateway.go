package wechat

import (
	"context"
)

// GatewayAdapter handles the gateway lifecycle for weixin.
type GatewayAdapter struct {
	bot *WechatBot
}

// NewGatewayAdapter creates a new gateway adapter.
func NewGatewayAdapter(bot *WechatBot) *GatewayAdapter {
	return &GatewayAdapter{bot: bot}
}

// StartAccount starts listening for messages for a WeChat account.
func (a *GatewayAdapter) StartAccount(ctx context.Context, accountID string) error {
	// Mark account as running
	a.bot.SetRunning(accountID, true)

	// The actual long-polling is handled by the gateway's PollingManager
	// This adapter just tracks the running state
	return nil
}

// StopAccount stops listening for messages for a WeChat account.
func (a *GatewayAdapter) StopAccount(ctx context.Context, accountID string) error {
	// Mark account as not running
	a.bot.SetRunning(accountID, false)

	return nil
}

// IsRunning checks if a WeChat account is running.
func (a *GatewayAdapter) IsRunning(accountID string) bool {
	return a.bot.IsRunningByID(accountID)
}
