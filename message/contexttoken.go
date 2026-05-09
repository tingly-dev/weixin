// Package message provides context token management for WeChat channel.
package message

import (
	"sync"
)

// contextTokenStore is keyed by accountID then by toUserID. The reference
// implementation uses a single Map<string,string> with composite keys, but
// per-account locking keeps writes safe under concurrent inbound traffic
// without contending across accounts.
var (
	contextTokenMu    sync.RWMutex
	contextTokenStore = make(map[string]map[string]string)
)

// SetContextToken stores a context token for a given conversation.
// Empty tokens are dropped (mirrors upstream behaviour: only persist real
// tokens received from the server).
func SetContextToken(accountID, toUserID, token string) {
	if token == "" {
		return
	}
	contextTokenMu.Lock()
	defer contextTokenMu.Unlock()

	tokens, ok := contextTokenStore[accountID]
	if !ok {
		tokens = make(map[string]string)
		contextTokenStore[accountID] = tokens
	}
	tokens[toUserID] = token
}

// GetContextToken retrieves a context token for a given conversation.
// Returns empty string if not found.
func GetContextToken(accountID, toUserID string) string {
	contextTokenMu.RLock()
	defer contextTokenMu.RUnlock()

	tokens, ok := contextTokenStore[accountID]
	if !ok {
		return ""
	}
	return tokens[toUserID]
}

// ClearContextToken removes a context token for a given conversation.
func ClearContextToken(accountID, toUserID string) {
	contextTokenMu.Lock()
	defer contextTokenMu.Unlock()

	if tokens, ok := contextTokenStore[accountID]; ok {
		delete(tokens, toUserID)
	}
}

// ClearAccountTokens removes all context tokens for an account.
func ClearAccountTokens(accountID string) {
	contextTokenMu.Lock()
	defer contextTokenMu.Unlock()

	delete(contextTokenStore, accountID)
}

// ResetForTest clears internal state - only for tests.
func ResetForTest() {
	contextTokenMu.Lock()
	defer contextTokenMu.Unlock()

	contextTokenStore = make(map[string]map[string]string)
}
