// Package adapters provides adapter implementations for the WeChat channel.
package adapters

import (
	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/channel"
)

// ConfigAdapter handles account configuration for weixin.
type ConfigAdapter struct {
	plugin weixin.PluginInterface
}

// NewConfigAdapter creates a new config adapter.
func NewConfigAdapter(plugin weixin.PluginInterface) *ConfigAdapter {
	return &ConfigAdapter{plugin: plugin}
}

// ListAccountIDs returns all configured WeChat account IDs.
func (a *ConfigAdapter) ListAccountIDs() ([]string, error) {
	return a.plugin.Accounts().ListIDs()
}

// ResolveAccount resolves a WeChat account by ID.
func (a *ConfigAdapter) ResolveAccount(accountID string) (*channel.Account, error) {
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
func (a *ConfigAdapter) DefaultAccount() (string, error) {
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
func (a *ConfigAdapter) IsEnabled(accountID string) (bool, error) {
	account, err := a.plugin.Accounts().Get(accountID)
	if err != nil {
		return false, err
	}
	return account.Enabled, nil
}

// IsConfigured checks if a WeChat account is configured.
func (a *ConfigAdapter) IsConfigured(accountID string) (bool, error) {
	account, err := a.plugin.Accounts().Get(accountID)
	if err != nil {
		return false, err
	}
	return account.Configured, nil
}
