package wecom

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/tingly-dev/weixin/channel"
)

// GatewayAdapter implements channel.GatewayAdapter for WeCom AI Bot.
// It manages the WebSocket connection lifecycle per account.
type GatewayAdapter struct {
	mu       sync.Mutex
	clients  map[string]*Client // accountID -> Client
	running  map[string]bool
	handlers map[string]channel.EventHandler
	logger   *log.Logger
}

// NewGatewayAdapter creates a new WeCom gateway adapter.
func NewGatewayAdapter(logger *log.Logger) *GatewayAdapter {
	return &GatewayAdapter{
		clients:  make(map[string]*Client),
		running:  make(map[string]bool),
		handlers: make(map[string]channel.EventHandler),
		logger:   logger,
	}
}

// StartAccount starts the WebSocket connection for an account.
func (g *GatewayAdapter) StartAccount(ctx context.Context, accountID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.running[accountID] {
		return fmt.Errorf("account %s is already running", accountID)
	}

	// Account config should have been set via SetAccountConfig
	client, ok := g.clients[accountID]
	if !ok || client == nil {
		return fmt.Errorf("account %s not configured", accountID)
	}

	if handler, ok := g.handlers[accountID]; ok {
		client.SetEventHandler(handler)
	}

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect wecom: %w", err)
	}

	g.running[accountID] = true
	return nil
}

// StopAccount stops the WebSocket connection for an account.
func (g *GatewayAdapter) StopAccount(ctx context.Context, accountID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	client, ok := g.clients[accountID]
	if !ok {
		g.running[accountID] = false
		return nil
	}

	if client != nil {
		client.Disconnect()
	}

	g.running[accountID] = false
	return nil
}

// IsRunning checks if an account's connection is active.
func (g *GatewayAdapter) IsRunning(accountID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.running[accountID] {
		return false
	}

	client, ok := g.clients[accountID]
	if !ok || client == nil {
		return false
	}
	return client.IsConnected()
}

// SetAccountConfig creates a WeCom client for an account with the given config.
// This must be called before StartAccount.
func (g *GatewayAdapter) SetAccountConfig(accountID string, cfg ClientConfig) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.logger != nil {
		cfg.Logger = g.logger
	}

	g.clients[accountID] = NewClient(cfg)
}

// SetEventHandler sets the event handler for a specific account.
func (g *GatewayAdapter) SetEventHandler(accountID string, handler channel.EventHandler) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.handlers[accountID] = handler

	if client, ok := g.clients[accountID]; ok && client != nil {
		client.SetEventHandler(handler)
	}
}

// GetClient returns the raw WeCom client for an account.
// Used by ActionsAdapter to send replies.
func (g *GatewayAdapter) GetClient(accountID string) *Client {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.clients[accountID]
}
