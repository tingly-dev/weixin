// package weixin provides configuration management for the WeChat channel.
package weixin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// AccountManager manages WeChat account persistence.
type AccountManager struct {
	baseDir string
	mu      sync.RWMutex
}

// NewAccountManager creates a new account manager.
func NewAccountManager() *AccountManager {
	// Use default base directory
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".agentchannel", "accounts")
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
func (m *AccountManager) Save(account *WeChatAccount) error {
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
func (m *AccountManager) Get(accountID string) (*WeChatAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path := m.accountPath(accountID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var account WeChatAccount
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
