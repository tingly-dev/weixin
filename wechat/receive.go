// Package wechat provides WeChat ilink bot implementation.
package wechat

import (
	"context"
	"fmt"
	"sync"

	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/message"
	"github.com/tingly-dev/weixin/storage"
	"github.com/tingly-dev/weixin/types"
)

// GetUpdates fetches new messages using long-polling.
// This is a single poll request.
func (b *WechatBot) GetUpdates(ctx context.Context, syncBuf string) (*types.GetUpdatesResult, error) {
	if b.account == nil {
		return nil, fmt.Errorf("bot has no account configured")
	}

	account := b.account.WeChatAccount()
	if !account.Enabled || !account.Configured {
		return nil, fmt.Errorf("account not enabled or configured")
	}

	// Check session guard
	if err := message.AssertSessionActive(b.account.ID()); err != nil {
		return &types.GetUpdatesResult{
			ErrCode: message.SessionExpiredErrCode,
			ErrMsg:  err.Error(),
		}, nil
	}

	// Load sync buffer if not provided
	if syncBuf == "" {
		syncBuf, _ = storage.LoadSyncBuf(b.account.ID())
	}

	// Call getUpdates with timeout
	resp, err := b.account.Client().GetUpdates(ctx, syncBuf)
	if err != nil {
		return nil, fmt.Errorf("get updates: %w", err)
	}

	// Check for session expiration
	if resp.ErrCode == message.SessionExpiredErrCode {
		message.PauseSession(b.account.ID())
		return &types.GetUpdatesResult{
			ErrCode: int(resp.ErrCode),
			ErrMsg:  resp.ErrMsg,
		}, nil
	}

	// Save sync buffer
	if resp.GetUpdatesBuf != "" {
		if err := storage.SaveSyncBuf(b.account.ID(), resp.GetUpdatesBuf); err != nil {
			// Log but don't fail the request
			fmt.Printf("[weixin] failed to save sync buffer: %v\n", err)
		}
	}

	// Convert WeixinMessage to Message
	messages := make([]*types.Message, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		// Only process USER messages (ignore BOT messages)
		if msg.MessageType == api.MessageTypeUser {
			// Save context token for replies
			if msg.ContextToken != "" && msg.FromUserID != "" {
				message.SetContextToken(b.account.ID(), msg.FromUserID, msg.ContextToken)
			}

			channelMsg := message.ConvertInboundMessage(&msg, b.account.ID())
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

	return &types.GetUpdatesResult{
		Messages:           messages,
		SyncBuf:            resp.GetUpdatesBuf,
		LongPollingTimeout: resp.LongPollingTimeoutMs,
		ErrCode:            0,
	}, nil
}

// Monitor handles continuous monitoring for messages.
// The monitor internally receives api.WeixinMessage and converts them to
// types.Message before passing to the user's handler.
type Monitor struct {
	bot     *WechatBot
	monitor *message.Monitor
	handler func(ctx context.Context, msg *types.Message) error
	mu      sync.RWMutex
	running bool
}

// NewMonitor creates a new monitor for receiving messages continuously.
func (b *WechatBot) NewMonitor() *Monitor {
	return &Monitor{
		bot: b,
	}
}

// SetHandler sets the message handler for the monitor.
// The handler receives a types.Message (converted from the internal api.WeixinMessage).
func (m *Monitor) SetHandler(handler func(ctx context.Context, msg *types.Message) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = handler
}

// Start starts the monitor.
func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("monitor already running")
	}

	if m.bot.account == nil {
		return fmt.Errorf("bot has no account configured")
	}

	account := m.bot.account.WeChatAccount()

	// Create monitor
	m.monitor = message.NewMonitor(m.bot.account.ID(), account.BaseURL, account.BotToken)
	m.monitor.SetOnMessage(func(ctx context.Context, msg *api.WeixinMessage) error {
		// Convert to types.Message
		channelMsg := message.ConvertInboundMessage(msg, m.bot.account.ID())
		if channelMsg != nil && m.handler != nil {
			return m.handler(ctx, channelMsg)
		}
		return nil
	})
	m.monitor.SetOnError(func(err error) {
		fmt.Printf("[weixin] monitor error: %v\n", err)
	})
	m.monitor.SetOnSession(func(sessionID string) {
		fmt.Printf("[weixin] new session detected: %s\n", sessionID)
	})

	// Start monitor
	if err := m.monitor.Start(ctx); err != nil {
		return fmt.Errorf("start monitor: %w", err)
	}

	m.running = true
	return nil
}

// Stop stops the monitor.
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.monitor != nil {
		m.monitor.Stop()
		m.monitor = nil
	}
	m.running = false
}

// IsRunning returns whether the monitor is running.
func (m *Monitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}
