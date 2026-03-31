package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/message"
	"github.com/tingly-dev/weixin/message/contexttoken"
	"github.com/tingly-dev/weixin/message/session"
	"github.com/tingly-dev/weixin/storage"
)

// longPollAdapter handles long-polling message synchronization for weixin.
type longPollAdapter struct {
	plugin   *Plugin
	monitors map[string]*message.Monitor // accountID -> monitor
	mu       sync.RWMutex
}

// newLongPollAdapter creates a new long-poll adapter.
func newLongPollAdapter(plugin *Plugin) *longPollAdapter {
	return &longPollAdapter{
		plugin:   plugin,
		monitors: make(map[string]*message.Monitor),
	}
}

// GetUpdates fetches new messages using long-polling.
// This is a single poll request - for continuous monitoring use StartMonitoring.
func (a *longPollAdapter) GetUpdates(ctx context.Context, req *weixin.GetUpdatesRequest) (*weixin.GetUpdatesResult, error) {
	fmt.Printf("[weixin] LongPollAdapter.GetUpdates called: accountID=%s, syncBuf=%q\n", req.AccountID, req.SyncBuf)

	// Get account
	account, err := a.plugin.Accounts().Get(req.AccountID)
	if err != nil {
		fmt.Printf("[weixin] GetUpdates failed to get account: %v\n", err)
		return nil, fmt.Errorf("get account: %w", err)
	}

	fmt.Printf("[weixin] Account: Enabled=%v, Configured=%v\n", account.Enabled, account.Configured)

	if !account.Enabled || !account.Configured {
		return nil, &Error{
			Type:    ErrorAccountNotFound,
			Message: "account not enabled or configured",
		}
	}

	// Check session guard
	if err := session.AssertSessionActive(req.AccountID); err != nil {
		fmt.Printf("[weixin] Session guard blocked: %v\n", err)
		return &weixin.GetUpdatesResult{
			ErrCode: session.SessionExpiredErrCode,
			ErrMsg:  err.Error(),
		}, nil
	}

	// Create API client
	client := api.NewClient(account.BaseURL, account.BotToken)
	fmt.Printf("[weixin] Calling API GetUpdates: baseURL=%q\n", account.BaseURL)

	// Load sync buffer if not provided
	syncBuf := req.SyncBuf
	if syncBuf == "" {
		syncBuf, _ = storage.LoadSyncBuf(req.AccountID)
		fmt.Printf("[weixin] Loaded syncBuf from storage: %q\n", syncBuf)
	}

	// Call getUpdates with timeout
	resp, err := client.GetUpdates(ctx, syncBuf)
	if err != nil {
		fmt.Printf("[weixin] API GetUpdates error: %v\n", err)
		return nil, fmt.Errorf("get updates: %w", err)
	}

	fmt.Printf("[weixin] API GetUpdates success: Ret=%d, ErrCode=%d, Messages=%d, SyncBuf=%q\n",
		resp.Ret, resp.ErrCode, len(resp.Messages), resp.GetUpdatesBuf)

	// Check for session expiration
	if resp.ErrCode == session.SessionExpiredErrCode {
		session.PauseSession(req.AccountID)
		return &weixin.GetUpdatesResult{
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
	messages := make([]*weixin.Message, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		// Only process USER messages (ignore BOT messages)
		if msg.MessageType == api.MessageTypeUser {
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

	return &weixin.GetUpdatesResult{
		Messages:           messages,
		SyncBuf:            resp.GetUpdatesBuf,
		LongPollingTimeout: resp.LongPollingTimeoutMs,
		ErrCode:            0,
	}, nil
}

// StartMonitoring starts continuous monitoring for an account.
func (a *longPollAdapter) StartMonitoring(ctx context.Context, accountID string, handler func(ctx context.Context, msg *api.WeixinMessage) error) error {
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
	m := message.NewMonitor(accountID, account.BaseURL, account.BotToken)
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
func (a *longPollAdapter) StopMonitoring(accountID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if m, exists := a.monitors[accountID]; exists {
		m.Stop()
		delete(a.monitors, accountID)
	}
}

// GetMonitor returns the monitor for an account (if running).
func (a *longPollAdapter) GetMonitor(accountID string) *message.Monitor {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.monitors[accountID]
}
