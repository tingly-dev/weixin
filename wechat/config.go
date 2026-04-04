package wechat

import (
	"github.com/tingly-dev/weixin/types"
)

// ConfigAdapter implements ConfigAdapter using the bot's account manager.
type ConfigAdapter struct {
	Bot *WechatBot
}

// ListAccountIDs returns all configured WeChat account IDs.
func (a *ConfigAdapter) ListAccountIDs() ([]string, error) {
	return a.Bot.Accounts().ListIDs()
}

// ResolveAccount resolves a WeChat account by ID.
func (a *ConfigAdapter) ResolveAccount(accountID string) (*types.Account, error) {
	wcAccount, err := a.Bot.Accounts().Get(accountID)
	if err != nil {
		return nil, &Error{
			Type:    ErrorAccountNotFound,
			Message: "account not found: " + accountID,
		}
	}

	return &types.Account{
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
	ids, err := a.Bot.Accounts().ListIDs()
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", &Error{
			Type:    ErrorAccountNotFound,
			Message: "no WeChat accounts configured",
		}
	}
	return ids[0], nil
}

// IsEnabled checks if a WeChat account is enabled.
func (a *ConfigAdapter) IsEnabled(accountID string) (bool, error) {
	account, err := a.Bot.Accounts().Get(accountID)
	if err != nil {
		return false, err
	}
	return account.Enabled, nil
}

// IsConfigured checks if a WeChat account is configured.
func (a *ConfigAdapter) IsConfigured(accountID string) (bool, error) {
	account, err := a.Bot.Accounts().Get(accountID)
	if err != nil {
		return false, err
	}
	return account.Configured, nil
}
