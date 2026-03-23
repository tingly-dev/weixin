// Package adapters provides adapter implementations for the WeChat channel.
package adapters

import (
	"context"

	"github.com/tingly-dev/weixin"
)

// GatewayAdapter handles the gateway lifecycle for weixin.
type GatewayAdapter struct {
	plugin weixin.PluginInterface
}

// NewGatewayAdapter creates a new gateway adapter.
func NewGatewayAdapter(plugin weixin.PluginInterface) *GatewayAdapter {
	return &GatewayAdapter{plugin: plugin}
}

// StartAccount starts listening for messages for a WeChat account.
func (a *GatewayAdapter) StartAccount(ctx context.Context, accountID string) error {
	// Mark account as running
	a.plugin.SetRunning(accountID, true)

	// The actual long-polling is handled by the gateway's PollingManager
	// This adapter just tracks the running state
	return nil
}

// StopAccount stops listening for messages for a WeChat account.
func (a *GatewayAdapter) StopAccount(ctx context.Context, accountID string) error {
	// Mark account as not running
	a.plugin.SetRunning(accountID, false)

	return nil
}

// IsRunning checks if a WeChat account is running.
func (a *GatewayAdapter) IsRunning(accountID string) bool {
	return a.plugin.IsRunningByID(accountID)
}
