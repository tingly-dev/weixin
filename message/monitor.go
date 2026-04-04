// Package monitor provides long-poll monitoring for WeChat messages.
package message

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/storage"
)

const (
	// DefaultLongPollTimeout is the default timeout for long-poll requests.
	DefaultLongPollTimeout = 35 * time.Second

	// MaxConsecutiveFailures is the maximum consecutive failures before backing off.
	MaxConsecutiveFailures = 3

	// BackoffDelay is the delay after max consecutive failures.
	BackoffDelay = 30 * time.Second

	// RetryDelay is the delay between retry attempts.
	RetryDelay = 2 * time.Second
)

// Monitor handles continuous long-polling for WeChat messages.
type Monitor struct {
	accountID string
	baseURL   string
	token     string
	client    *api.Client

	// Event handlers
	onMessage func(ctx context.Context, msg *api.WeixinMessage) error
	onError   func(err error)
	onSession func(sessionID string) // Called when a new session is detected

	// State
	syncBuf         string
	nextTimeout     time.Duration
	consecutiveFail int
	running         bool
	mu              sync.RWMutex
}

// NewMonitor creates a new WeChat monitor.
func NewMonitor(accountID, baseURL, token string) *Monitor {
	return &Monitor{
		accountID:   accountID,
		baseURL:     baseURL,
		token:       token,
		client:      api.NewClient(baseURL, token),
		nextTimeout: DefaultLongPollTimeout,
	}
}

// SetOnMessage sets the handler for incoming messages.
func (m *Monitor) SetOnMessage(handler func(ctx context.Context, msg *api.WeixinMessage) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onMessage = handler
}

// SetOnError sets the handler for errors.
func (m *Monitor) SetOnError(handler func(err error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onError = handler
}

// SetOnSession sets the handler for session changes.
func (m *Monitor) SetOnSession(handler func(sessionID string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onSession = handler
}

// Start begins continuous monitoring.
func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	m.running = true
	m.mu.Unlock()

	// Load previous sync buffer
	if buf, err := storage.LoadSyncBuf(m.accountID); err != nil {
		log.Printf("[weixin] failed to load sync buffer: %v", err)
	} else if buf != "" {
		m.syncBuf = buf
		log.Printf("[weixin] resumed from previous sync buf (%d bytes)", len(buf))
	}

	// Start monitoring loop
	go m.monitorLoop(ctx)

	return nil
}

// Stop stops the monitoring loop.
func (m *Monitor) Stop() {
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()
}

// IsRunning returns true if the monitor is running.
func (m *Monitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// monitorLoop is the main long-polling loop.
func (m *Monitor) monitorLoop(ctx context.Context) {
	for {
		m.mu.RLock()
		running := m.running
		m.mu.RUnlock()

		if !running {
			return
		}

		// Check for session pause
		if IsSessionPaused(m.accountID) {
			remaining := GetRemainingPauseMs(m.accountID)
			log.Printf("[weixin] session paused for %v, waiting...", remaining)

			select {
			case <-ctx.Done():
				return
			case <-time.After(remaining):
				// Pause expired, continue
				log.Printf("[weixin] session pause expired, resuming")
			}
			continue
		}

		// Perform long-poll
		resp, err := m.poll(ctx)
		if err != nil {
			m.handleError(err)
			// Continue loop on error
			time.Sleep(RetryDelay)
			continue
		}

		// Update sync buffer
		if resp.GetUpdatesBuf != "" {
			m.syncBuf = resp.GetUpdatesBuf
			if err := storage.SaveSyncBuf(m.accountID, m.syncBuf); err != nil {
				log.Printf("[weixin] failed to save sync buffer: %v", err)
			}
		}

		// Update timeout from server response
		if resp.LongPollingTimeoutMs > 0 {
			m.nextTimeout = time.Duration(resp.LongPollingTimeoutMs) * time.Millisecond
		}

		// Process messages
		if len(resp.Messages) > 0 {
			m.consecutiveFail = 0
			for _, msg := range resp.Messages {
				if err := m.processMessage(ctx, &msg); err != nil {
					m.handleError(fmt.Errorf("process message: %w", err))
				}
			}
		}
	}
}

// poll performs a single long-poll request.
func (m *Monitor) poll(ctx context.Context) (*api.GetUpdatesResponse, error) {
	return m.client.GetUpdatesWithTimeout(ctx, m.syncBuf, m.nextTimeout)
}

// processMessage handles a single incoming message.
func (m *Monitor) processMessage(ctx context.Context, msg *api.WeixinMessage) error {
	m.mu.RLock()
	handler := m.onMessage
	sessionHandler := m.onSession
	m.mu.RUnlock()

	// Only process USER messages (ignore BOT messages)
	if msg.MessageType != api.MessageTypeUser {
		return nil
	}

	// Check for session change
	if msg.SessionID != "" && sessionHandler != nil {
		sessionHandler(msg.SessionID)
	}

	// Call message handler
	if handler != nil {
		return handler(ctx, msg)
	}

	return nil
}

// handleError handles an error from polling or message processing.
func (m *Monitor) handleError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.consecutiveFail++

	// Get error handler
	handler := m.onError
	if handler != nil {
		handler(err)
	}

	log.Printf("[weixin] monitor error (%d/%d): %v", m.consecutiveFail, MaxConsecutiveFailures, err)

	// Check for session expiration error
	if isSessionExpiredError(err) {
		PauseSession(m.accountID)
		m.consecutiveFail = 0
		return
	}

	// Backoff after max consecutive failures
	if m.consecutiveFail >= MaxConsecutiveFailures {
		log.Printf("[weixin] max consecutive failures reached, backing off for %v", BackoffDelay)
		m.consecutiveFail = 0
	}
}

// isSessionExpiredError checks if the error indicates a session expiration.
func isSessionExpiredError(err error) bool {
	return err != nil && err.Error() == fmt.Sprintf("ret=%d", SessionExpiredErrCode)
}

// GetSyncBuf returns the current sync buffer.
func (m *Monitor) GetSyncBuf() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.syncBuf
}

// GetNextTimeout returns the next long-poll timeout.
func (m *Monitor) GetNextTimeout() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nextTimeout
}
