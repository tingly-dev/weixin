// Package adapters provides adapter implementations for the WeChat channel.
package adapters

import (
	"context"
	"fmt"
	"sync"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/channel"
	"github.com/tingly-dev/weixin/contexttoken"
	"github.com/tingly-dev/weixin/message"
	"github.com/tingly-dev/weixin/monitor"
	"github.com/tingly-dev/weixin/sessionguard"
	"github.com/tingly-dev/weixin/storage"
)

// LongPollAdapter handles long-polling message synchronization for weixin.
type LongPollAdapter struct {
	plugin   weixin.PluginInterface
	monitors map[string]*monitor.Monitor // accountID -> monitor
	mu       sync.RWMutex
}

// NewLongPollAdapter creates a new long-poll adapter.
func NewLongPollAdapter(plugin weixin.PluginInterface) *LongPollAdapter {
	return &LongPollAdapter{
		plugin:   plugin,
		monitors: make(map[string]*monitor.Monitor),
	}
}

// GetUpdates fetches new messages using long-polling.
// This is a single poll request - for continuous monitoring use StartMonitoring.
func (a *LongPollAdapter) GetUpdates(ctx context.Context, req *channel.GetUpdatesRequest) (*channel.GetUpdatesResult, error) {
	// Get account
	account, err := a.plugin.Accounts().Get(req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	if !account.Enabled || !account.Configured {
		return nil, &channel.ChannelError{
			Type:    channel.ErrorAccountNotFound,
			Message: "account not enabled or configured",
			Channel: channel.ChannelIDWeChat,
		}
	}

	// Check session guard
	if err := sessionguard.AssertSessionActive(req.AccountID); err != nil {
		return &channel.GetUpdatesResult{
			ErrCode: sessionguard.SessionExpiredErrCode,
			ErrMsg:  err.Error(),
		}, nil
	}

	// Create API client
	client := api.NewClient(account.BaseURL, account.BotToken)

	// Load sync buffer if not provided
	syncBuf := req.SyncBuf
	if syncBuf == "" {
		syncBuf, _ = storage.LoadSyncBuf(req.AccountID)
	}

	// Call getUpdates with timeout
	resp, err := client.GetUpdates(ctx, syncBuf)
	if err != nil {
		return nil, fmt.Errorf("get updates: %w", err)
	}

	// Check for session expiration
	if resp.ErrCode == sessionguard.SessionExpiredErrCode {
		sessionguard.PauseSession(req.AccountID)
		return &channel.GetUpdatesResult{
			ErrCode: int(resp.ErrCode),
			ErrMsg:  resp.ErrMsg,
		}, nil
	}

	// Save sync buffer
	if resp.GetUpdatesBuf != "" {
		if err := storage.SaveSyncBuf(req.AccountID, resp.GetUpdatesBuf); err != nil {
			// Log but don't fail the request
			fmt.Printf("[weixin] failed to save sync buffer: %v\n", err)
		}
	}

	// Convert WeixinMessage to channel.Message
	messages := make([]*channel.Message, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		// Only process USER messages (ignore BOT messages)
		if msg.MessageType == weixin.MessageTypeUser {
			// Save context token for replies
			if msg.ContextToken != "" && msg.FromUserID != "" {
				contexttoken.SetContextToken(req.AccountID, msg.FromUserID, msg.ContextToken)
			}

			channelMsg := message.ConvertInboundMessage(&msg, req.AccountID)
			if channelMsg != nil {
				// Include context token in message metadata
				if channelMsg.Metadata == nil {
					channelMsg.Metadata = make(map[string]interface{})
				}
				channelMsg.Metadata["context_token"] = msg.ContextToken

				messages = append(messages, channelMsg)
			}
		}
	}

	return &channel.GetUpdatesResult{
		Messages:           messages,
		SyncBuf:            resp.GetUpdatesBuf,
		LongPollingTimeout: resp.LongPollingTimeoutMs,
		ErrCode:            0,
	}, nil
}

// StartMonitoring starts continuous monitoring for an account.
func (a *LongPollAdapter) StartMonitoring(ctx context.Context, accountID string, handler func(ctx context.Context, msg *weixin.WeixinMessage) error) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if already monitoring
	if _, exists := a.monitors[accountID]; exists {
		return fmt.Errorf("already monitoring account %s", accountID)
	}

	// Get account
	account, err := a.plugin.Accounts().Get(accountID)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}

	// Create monitor
	m := monitor.NewMonitor(accountID, account.BaseURL, account.BotToken)
	m.SetOnMessage(handler)
	m.SetOnError(func(err error) {
		fmt.Printf("[weixin] monitor error for %s: %v\n", accountID, err)
	})
	m.SetOnSession(func(sessionID string) {
		fmt.Printf("[weixin] new session detected for %s: %s\n", accountID, sessionID)
	})

	// Start monitor
	if err := m.Start(ctx); err != nil {
		return fmt.Errorf("start monitor: %w", err)
	}

	a.monitors[accountID] = m
	return nil
}

// StopMonitoring stops continuous monitoring for an account.
func (a *LongPollAdapter) StopMonitoring(accountID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if m, exists := a.monitors[accountID]; exists {
		m.Stop()
		delete(a.monitors, accountID)
	}
}

// GetMonitor returns the monitor for an account (if running).
func (a *LongPollAdapter) GetMonitor(accountID string) *monitor.Monitor {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.monitors[accountID]
}
