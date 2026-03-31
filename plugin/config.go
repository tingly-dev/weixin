package plugin

import "github.com/tingly-dev/weixin"

// configAdapter implements ConfigAdapter using the plugin's account manager.
type configAdapter struct {
	plugin *Plugin
}

// ListAccountIDs returns all configured WeChat account IDs.
func (a *configAdapter) ListAccountIDs() ([]string, error) {
	return a.plugin.Accounts().ListIDs()
}

// ResolveAccount resolves a WeChat account by ID.
func (a *configAdapter) ResolveAccount(accountID string) (*weixin.Account, error) {
	wcAccount, err := a.plugin.Accounts().Get(accountID)
	if err != nil {
		return nil, &Error{
			Type:    ErrorAccountNotFound,
			Message: "account not found: " + accountID,
		}
	}

	return &weixin.Account{
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
func (a *configAdapter) DefaultAccount() (string, error) {
	ids, err := a.plugin.Accounts().ListIDs()
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
func (a *configAdapter) IsEnabled(accountID string) (bool, error) {
	account, err := a.plugin.Accounts().Get(accountID)
	if err != nil {
		return false, err
	}
	return account.Enabled, nil
}

// IsConfigured checks if a WeChat account is configured.
func (a *configAdapter) IsConfigured(accountID string) (bool, error) {
	account, err := a.plugin.Accounts().Get(accountID)
	if err != nil {
		return false, err
	}
	return account.Configured, nil
}
