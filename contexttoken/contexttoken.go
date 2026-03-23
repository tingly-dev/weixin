// Package contexttoken provides context token management for WeChat channel.
package contexttoken

import (
	"sync"
)

var (
	// contextTokenMap stores context tokens per (accountID, toUserID) pair.
	// contextTokenMap[accountID][toUserID] = contextToken
	contextTokenMap sync.Map // map[string]map[string]string
)

// SetContextToken stores a context token for a given conversation.
func SetContextToken(accountID, toUserID, token string) {
	if token == "" {
		return
	}

	// Get or create the account's token map
	accountTokens, _ := contextTokenMap.LoadOrStore(accountID, make(map[string]string))
	tokens := accountTokens.(map[string]string)

	// Store the token (use a mutex for the inner map)
	// Note: In production, use a more sophisticated locking mechanism
	tokens[toUserID] = token
}

// GetContextToken retrieves a context token for a given conversation.
// Returns empty string if not found.
func GetContextToken(accountID, toUserID string) string {
	value, ok := contextTokenMap.Load(accountID)
	if !ok {
		return ""
	}

	tokens := value.(map[string]string)
	return tokens[toUserID]
}

// ClearContextToken removes a context token for a given conversation.
func ClearContextToken(accountID, toUserID string) {
	value, ok := contextTokenMap.Load(accountID)
	if !ok {
		return
	}

	tokens := value.(map[string]string)
	delete(tokens, toUserID)
}

// ClearAccountTokens removes all context tokens for an account.
func ClearAccountTokens(accountID string) {
	contextTokenMap.Delete(accountID)
}

// ResetForTest clears internal state - only for tests.
func ResetForTest() {
	contextTokenMap = sync.Map{}
}