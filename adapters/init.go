// Package adapters provides adapter initialization.
// This file is imported for its side effect of initializing adapters on a plugin.
package adapters

import "github.com/tingly-dev/weixin"

// InitPlugin initializes all adapters on a WeChat plugin.
// This should be called after creating a new plugin.
func InitPlugin(plugin *weixin.Plugin) {
	// Note: The config adapter is already set during plugin creation via tempConfigAdapter
	// We just need to set the other adapters

	// Set other adapters
	plugin.SetActions(NewActionsAdapter(plugin))
	plugin.SetGateway(NewGatewayAdapter(plugin))
	plugin.SetLongPoll(NewLongPollAdapter(plugin))
	plugin.SetUpload(NewUploadAdapter(plugin))
	plugin.SetPairing(NewPairingAdapter(plugin))
}
