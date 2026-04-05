// Package wechat provides the WeChat ilink bot implementation.
package wechat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/types"
)

// Account represents a single WeChat account with its API client.
// One account = one API client = one bot instance.
type Account struct {
	id      string
	client  *api.Client
	account *types.WeChatAccount
}

// NewAccount creates a new account with API client from a WeChatAccount.
func NewAccount(wcAccount *types.WeChatAccount) *Account {
	return &Account{
		id:      wcAccount.ID,
		client:  api.NewClient(wcAccount.BaseURL, wcAccount.BotToken),
		account: wcAccount,
	}
}

// NewAccountWithClient creates a new account with an existing API client.
func NewAccountWithClient(id string, client *api.Client, wcAccount *types.WeChatAccount) *Account {
	return &Account{
		id:      id,
		client:  client,
		account: wcAccount,
	}
}

// ID returns the account ID.
func (a *Account) ID() string {
	return a.id
}

// Client returns the underlying API client.
func (a *Account) Client() *api.Client {
	return a.client
}

// WeChatAccount returns the WeChat account details.
func (a *Account) WeChatAccount() *types.WeChatAccount {
	return a.account
}

// BaseURL returns the API base URL.
func (a *Account) BaseURL() string {
	return a.account.BaseURL
}

// BotToken returns the bot token.
func (a *Account) BotToken() string {
	return a.account.BotToken
}

// BotID returns the bot ID.
func (a *Account) BotID() string {
	return a.account.BotID
}

// UserID returns the user ID.
func (a *Account) UserID() string {
	return a.account.UserID
}

// IsEnabled returns whether the account is enabled.
func (a *Account) IsEnabled() bool {
	return a.account.Enabled
}

// IsConfigured returns whether the account is configured.
func (a *Account) IsConfigured() bool {
	return a.account.Configured
}

// AccountManager manages WeChat account persistence.
type AccountManager struct {
	baseDir string
	mu      sync.RWMutex
}

// NewAccountManager creates a new account manager.
func NewAccountManager() *AccountManager {
	// Use default base directory
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".weixin", "accounts")
	os.MkdirAll(baseDir, 0700)

	return &AccountManager{
		baseDir: baseDir,
	}
}

// NewAccountManagerWithDir creates a new account manager with a custom base directory.
func NewAccountManagerWithDir(baseDir string) *AccountManager {
	os.MkdirAll(baseDir, 0700)
	return &AccountManager{
		baseDir: baseDir,
	}
}

// accountPath returns the path to an account file.
func (m *AccountManager) accountPath(accountID string) string {
	return filepath.Join(m.baseDir, accountID+".json")
}

// Save saves an account to disk.
func (m *AccountManager) Save(account *types.WeChatAccount) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return err
	}

	path := m.accountPath(account.ID)
	return os.WriteFile(path, data, 0600)
}

// Get retrieves an account by ID.
func (m *AccountManager) Get(accountID string) (*types.WeChatAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := m.accountPath(accountID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var account types.WeChatAccount
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// ListIDs returns all account IDs.
func (m *AccountManager) ListIDs() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) > 5 && name[len(name)-5:] == ".json" {
			ids = append(ids, name[:len(name)-5])
		}
	}

	return ids, nil
}

// Delete removes an account.
func (m *AccountManager) Delete(accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.accountPath(accountID)
	return os.Remove(path)
}
