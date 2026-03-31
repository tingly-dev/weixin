package plugin

import (
	"context"
)

// gatewayAdapter handles the gateway lifecycle for weixin.
type gatewayAdapter struct {
	plugin *Plugin
}

// newGatewayAdapter creates a new gateway adapter.
func newGatewayAdapter(plugin *Plugin) *gatewayAdapter {
	return &gatewayAdapter{plugin: plugin}
}

// StartAccount starts listening for messages for a WeChat account.
func (a *gatewayAdapter) StartAccount(ctx context.Context, accountID string) error {
	// Mark account as running
	a.plugin.SetRunning(accountID, true)

	// The actual long-polling is handled by the gateway's PollingManager
	// This adapter just tracks the running state
	return nil
}

// StopAccount stops listening for messages for a WeChat account.
func (a *gatewayAdapter) StopAccount(ctx context.Context, accountID string) error {
	// Mark account as not running
	a.plugin.SetRunning(accountID, false)

	return nil
}

// IsRunning checks if a WeChat account is running.
func (a *gatewayAdapter) IsRunning(accountID string) bool {
	return a.plugin.IsRunningByID(accountID)
}
