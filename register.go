// package weixin provides plugin registration for the WeChat channel.
package weixin

import (
	"github.com/tingly-dev/weixin/channel"
)

// RegisterPlugin registers the WeChat plugin with a channel registry.
// Note: You must call InitAdapters() on the returned plugin before using it.
func RegisterPlugin(registry *channel.Registry, baseURL string) (*Plugin, error) {
	config := &WeChatConfig{
		BaseURL: baseURL,
		BotType: "3",
	}

	plugin := NewPlugin(config)
	return plugin, registry.Register(plugin)
}

// NewPluginWithAdapters creates a new WeChat plugin with all adapters initialized.
// Note: Due to import cycles, adapters must be initialized by calling
// the InitAdapters function from the adapters package:
//
//	plugin := weixin.NewPlugin(config)
//	adapters.InitPlugin(plugin)
func NewPluginWithAdapters(config *WeChatConfig) *Plugin {
	return NewPlugin(config)
}
