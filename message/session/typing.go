// Package typing provides typing indicator support for WeChat.
package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tingly-dev/weixin/api"
)

const (
	// TypingKeepaliveInterval is how often to send typing keepalive.
	TypingKeepaliveInterval = 5 * time.Second
)

// TypingStatus constants from WeChat API.
const (
	TypingStatusTyping = 1
	TypingStatusCancel = 2
)

// TypingManager manages typing indicators for WeChat conversations.
type TypingManager struct {
	client    *api.Client
	active    map[string]*typingState // accountID+userID -> state
	mu        sync.RWMutex
	stopChans map[string]chan struct{}
}

type typingState struct {
	ilinkUserID  string
	typingTicket string
}

// NewTypingManager creates a new typing manager.
func NewTypingManager() *TypingManager {
	return &TypingManager{
		active:    make(map[string]*typingState),
		stopChans: make(map[string]chan struct{}),
	}
}

// SetClient sets the API client to use for sending typing indicators.
func (tm *TypingManager) SetClient(client *api.Client) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.client = client
}

// StartTyping starts sending typing indicators for a conversation.
// Returns a function that should be called to stop typing.
func (tm *TypingManager) StartTyping(ctx context.Context, accountID, userID string) func() {
	key := accountID + ":" + userID

	tm.mu.Lock()

	// Check if already typing
	if _, exists := tm.active[key]; exists {
		tm.mu.Unlock()
		// Return a no-op function
		return func() {}
	}

	// Create stop channel
	stopChan := make(chan struct{})
	tm.stopChans[key] = stopChan
	tm.mu.Unlock()

	// Send initial typing indicator
	tm.sendTyping(ctx, accountID, userID, TypingStatusTyping)

	// Start keepalive loop
	go tm.keepaliveLoop(ctx, key, userID, stopChan)

	// Return stop function
	return func() {
		tm.StopTyping(ctx, accountID, userID)
	}
}

// StopTyping stops typing indicators for a conversation.
func (tm *TypingManager) StopTyping(ctx context.Context, accountID, userID string) {
	key := accountID + ":" + userID

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Send cancel indicator
	tm.sendTyping(ctx, accountID, userID, TypingStatusCancel)

	// Stop keepalive loop
	if stopChan, exists := tm.stopChans[key]; exists {
		close(stopChan)
		delete(tm.stopChans, key)
	}

	// Remove active state
	delete(tm.active, key)
}

// SetTypingTicket sets the typing ticket for a conversation.
func (tm *TypingManager) SetTypingTicket(accountID, userID, ticket string) {
	key := accountID + ":" + userID

	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.active[key] = &typingState{
		ilinkUserID:  userID,
		typingTicket: ticket,
	}
}

// sendTyping sends a typing indicator.
func (tm *TypingManager) sendTyping(ctx context.Context, accountID, userID string, status int) {
	tm.mu.RLock()
	state, exists := tm.active[accountID+":"+userID]
	client := tm.client
	tm.mu.RUnlock()

	if !exists || client == nil {
		return
	}

	// Send using client method
	if err := client.SendTyping(ctx, state.ilinkUserID, state.typingTicket, status); err != nil {
		fmt.Printf("[weixin] failed to send typing indicator: %v\n", err)
	}
}

// keepaliveLoop sends periodic typing keepalive messages.
func (tm *TypingManager) keepaliveLoop(ctx context.Context, key, userID string, stopChan <-chan struct{}) {
	ticker := time.NewTicker(TypingKeepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			// Send typing keepalive
			parts := splitKey(key)
			if len(parts) == 2 {
				tm.sendTyping(ctx, parts[0], parts[1], TypingStatusTyping)
			}
		case <-ctx.Done():
			return
		}
	}
}

// splitKey splits a key into accountID and userID.
func splitKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key}
}
