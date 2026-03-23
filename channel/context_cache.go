// Package channel provides core types and interfaces for the AgentChannel SDK.
package channel

import (
	"time"
)

// ContextTokenCache manages context tokens for session continuity.
// Context tokens are required by some channel plugins (e.g., WeChat) to
// maintain conversation context when sending replies.
type ContextTokenCache struct {
	store map[string]cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	token     string
	expiresAt time.Time
}

// NewContextTokenCache creates a new context token cache.
func NewContextTokenCache(ttl time.Duration) *ContextTokenCache {
	return &ContextTokenCache{
		store: make(map[string]cacheEntry),
		ttl:   ttl,
	}
}

// Set stores a context token for a specific account and user pair.
func (c *ContextTokenCache) Set(accountID, userID, token string) {
	c.store[cacheKey(accountID, userID)] = cacheEntry{
		token:     token,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Get retrieves a context token for a specific account and user pair.
// Returns the token and true if found and not expired, otherwise ("", false).
func (c *ContextTokenCache) Get(accountID, userID string) (string, bool) {
	entry, exists := c.store[cacheKey(accountID, userID)]
	if !exists || time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.token, true
}

// Delete removes a context token from the cache.
func (c *ContextTokenCache) Delete(accountID, userID string) {
	delete(c.store, cacheKey(accountID, userID))
}

// Clear removes all entries from the cache.
func (c *ContextTokenCache) Clear() {
	c.store = make(map[string]cacheEntry)
}

// Cleanup removes expired entries from the cache.
func (c *ContextTokenCache) Cleanup() int {
	now := time.Now()
	removed := 0
	for key, entry := range c.store {
		if now.After(entry.expiresAt) {
			delete(c.store, key)
			removed++
		}
	}
	return removed
}

// cacheKey generates a unique cache key for account and user.
func cacheKey(accountID, userID string) string {
	return accountID + ":" + userID
}
