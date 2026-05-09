// Package api provides WeChat API implementations.
package api

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// CachedConfig is the subset of GetConfig fields used by callers.
type CachedConfig struct {
	TypingTicket string
}

const (
	configCacheTTL          = 24 * time.Hour
	configCacheInitialRetry = 2 * time.Second
	configCacheMaxRetry     = 1 * time.Hour
)

type configCacheEntry struct {
	config        CachedConfig
	everSucceeded bool
	nextFetchAt   time.Time
	retryDelay    time.Duration
}

// ConfigManager is a per-user getConfig cache with periodic random refresh
// (within 24h) and exponential-backoff retry (up to 1h) on failure.
//
// This mirrors the upstream WeixinConfigManager behaviour described in the
// reference implementation (config-cache.ts).
type ConfigManager struct {
	client *Client
	logf   func(format string, args ...interface{})
	mu     sync.Mutex
	cache  map[string]*configCacheEntry
}

// NewConfigManager constructs a ConfigManager bound to the given Client.
// `logf` may be nil (logging is suppressed in that case).
func NewConfigManager(c *Client, logf func(format string, args ...interface{})) *ConfigManager {
	return &ConfigManager{
		client: c,
		logf:   logf,
		cache:  make(map[string]*configCacheEntry),
	}
}

func (m *ConfigManager) log(format string, args ...interface{}) {
	if m.logf != nil {
		m.logf(format, args...)
	}
}

// GetForUser returns a cached typing ticket for `userID`, refreshing it
// from the server when stale. On failure the previous value (if any) is
// returned with a backoff applied to the next retry.
func (m *ConfigManager) GetForUser(ctx context.Context, userID, contextToken string) CachedConfig {
	m.mu.Lock()
	entry, ok := m.cache[userID]
	now := time.Now()
	shouldFetch := !ok || now.After(entry.nextFetchAt)
	m.mu.Unlock()

	if !shouldFetch {
		return entry.config
	}

	resp, err := m.client.GetConfig(ctx, userID, contextToken)

	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok = m.cache[userID]

	fetchOK := err == nil && resp != nil && resp.Ret == 0
	if fetchOK {
		ttl := time.Duration(rand.Int63n(int64(configCacheTTL)))
		m.cache[userID] = &configCacheEntry{
			config:        CachedConfig{TypingTicket: resp.TypingTicket},
			everSucceeded: true,
			nextFetchAt:   now.Add(ttl),
			retryDelay:    configCacheInitialRetry,
		}
		if ok && entry.everSucceeded {
			m.log("[weixin] config refreshed for %s", userID)
		} else {
			m.log("[weixin] config cached for %s", userID)
		}
		return m.cache[userID].config
	}

	if err != nil {
		m.log("[weixin] getConfig failed for %s (ignored): %v", userID, err)
	}

	prevDelay := configCacheInitialRetry
	if ok {
		prevDelay = entry.retryDelay
	}
	nextDelay := prevDelay * 2
	if nextDelay > configCacheMaxRetry {
		nextDelay = configCacheMaxRetry
	}

	if ok {
		entry.nextFetchAt = now.Add(nextDelay)
		entry.retryDelay = nextDelay
		return entry.config
	}

	m.cache[userID] = &configCacheEntry{
		config:        CachedConfig{},
		everSucceeded: false,
		nextFetchAt:   now.Add(configCacheInitialRetry),
		retryDelay:    configCacheInitialRetry,
	}
	return m.cache[userID].config
}
