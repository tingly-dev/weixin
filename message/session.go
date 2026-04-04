// Package sessionguard provides session pause mechanism for WeChat channel.
package message

import (
	"fmt"
	"sync"
	"time"
)

const (
	// SessionPauseDuration is how long to pause after session expiration.
	SessionPauseDuration = 60 * time.Minute // 1 hour

	// SessionExpiredErrCode is the error code returned when session has expired.
	SessionExpiredErrCode = -14
)

var (
	// pauseUntilMap tracks when each account's session will be unpaused.
	pauseUntilMap sync.Map // map[string]time.Time
)

// PauseSession pauses all inbound/outbound API calls for accountId for one hour.
func PauseSession(accountID string) {
	until := time.Now().Add(SessionPauseDuration)
	pauseUntilMap.Store(accountID, until)
}

// IsSessionPaused returns true when the bot is still within its one-hour cooldown window.
func IsSessionPaused(accountID string) bool {
	value, ok := pauseUntilMap.Load(accountID)
	if !ok {
		return false
	}

	until, ok := value.(time.Time)
	if !ok {
		return false
	}

	if time.Now().After(until) {
		// Pause has expired, remove the entry
		pauseUntilMap.Delete(accountID)
		return false
	}

	return true
}

// GetRemainingPauseMs returns milliseconds remaining until the pause expires (0 when not paused).
func GetRemainingPauseMs(accountID string) time.Duration {
	value, ok := pauseUntilMap.Load(accountID)
	if !ok {
		return 0
	}

	until, ok := value.(time.Time)
	if !ok {
		return 0
	}

	remaining := time.Until(until)
	if remaining <= 0 {
		pauseUntilMap.Delete(accountID)
		return 0
	}

	return remaining
}

// AssertSessionActive throws an error if the session is currently paused.
// Call before any API request.
func AssertSessionActive(accountID string) error {
	if IsSessionPaused(accountID) {
		remaining := GetRemainingPauseMs(accountID)
		return &SessionPausedError{
			AccountID: accountID,
			Remaining: remaining,
			ErrCode:   SessionExpiredErrCode,
		}
	}
	return nil
}

// SessionPausedError indicates a session is currently paused.
type SessionPausedError struct {
	AccountID string
	Remaining time.Duration
	ErrCode   int
}

func (e *SessionPausedError) Error() string {
	return fmt.Sprintf("session paused for accountId=%s, %d min remaining (errcode %d)",
		e.AccountID, int(e.Remaining.Minutes()), e.ErrCode)
}
