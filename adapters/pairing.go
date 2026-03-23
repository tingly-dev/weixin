// Package adapters provides adapter implementations for the WeChat channel.
package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/channel"
)

// PairingAdapter handles QR code login for weixin.
type PairingAdapter struct {
	plugin weixin.PluginInterface
}

// NewPairingAdapter creates a new pairing adapter.
func NewPairingAdapter(plugin weixin.PluginInterface) *PairingAdapter {
	return &PairingAdapter{plugin: plugin}
}

// activeLogin represents an active QR code login session.
type activeLogin struct {
	sessionKey string
	qrID       string
	qrURL      string
	startedAt  time.Time
}

var (
	activeLogins = make(map[string]*activeLogin)
	loginMutex   sync.RWMutex
)

const (
	activeLoginTTL    = 5 * time.Minute
	defaultBotType    = "3"
	qrPollTimeout     = 35 * time.Second
	maxQRRefreshCount = 3
)

// LoginWithQrStart initiates QR code login flow.
func (a *PairingAdapter) LoginWithQrStart(ctx context.Context, accountID string) (*channel.QrCodeStartResult, error) {
	// Get plugin config
	config := a.plugin.WeChatConfig()

	// Create API client without token (for login)
	client := api.NewClient(config.BaseURL, "")

	// Fetch QR code
	qrResp, err := client.GetBotQRCode(ctx, defaultBotType)
	if err != nil {
		return nil, fmt.Errorf("fetch QR code: %w", err)
	}

	// Store active login session
	login := &activeLogin{
		sessionKey: accountID,
		qrID:       qrResp.Qrcode,
		qrURL:      qrResp.QrcodeImgContent,
		startedAt:  time.Now(),
	}

	loginMutex.Lock()
	activeLogins[accountID] = login
	loginMutex.Unlock()

	return &channel.QrCodeStartResult{
		QrCodeID:   qrResp.Qrcode,
		QrCodeURL:  qrResp.QrcodeImgContent,
		QrCodeData: qrResp.QrcodeImgContent,
		ExpiresIn:  int(activeLoginTTL.Seconds()),
	}, nil
}

// LoginWithQrWait waits for QR code scan confirmation.
func (a *PairingAdapter) LoginWithQrWait(ctx context.Context, accountID, qrID string) (*channel.QrCodeWaitResult, error) {
	// Get plugin config
	config := a.plugin.WeChatConfig()
	client := api.NewClient(config.BaseURL, "")

	// Check for active login
	loginMutex.RLock()
	login, exists := activeLogins[accountID]
	loginMutex.RUnlock()

	if !exists || login.qrID != qrID {
		return nil, fmt.Errorf("no active login session for QR code: %s", qrID)
	}

	// Check if login has expired
	if time.Since(login.startedAt) > activeLoginTTL {
		loginMutex.Lock()
		delete(activeLogins, accountID)
		loginMutex.Unlock()
		return &channel.QrCodeWaitResult{
			Success: false,
			Error:   "QR code expired",
		}, nil
	}

	// Poll for QR status with timeout
	deadline := time.Now().Add(8 * time.Minute) // Total wait time
	refreshCount := 0

	for time.Now().Before(deadline) {
		// Poll QR status
		statusResp, err := client.GetQRStatus(ctx, qrID)
		if err != nil {
			loginMutex.Lock()
			delete(activeLogins, accountID)
			loginMutex.Unlock()
			return &channel.QrCodeWaitResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		switch statusResp.Status {
		case "wait":
			// Still waiting, continue polling
			time.Sleep(2 * time.Second)
			continue

		case "scaned":
			// User scanned but hasn't confirmed yet
			time.Sleep(2 * time.Second)
			continue

		case "expired":
			// QR code expired, refresh it
			refreshCount++
			if refreshCount > maxQRRefreshCount {
				loginMutex.Lock()
				delete(activeLogins, accountID)
				loginMutex.Unlock()
				return &channel.QrCodeWaitResult{
					Success: false,
					Error:   "QR code expired too many times",
				}, nil
			}

			// Fetch new QR code
			qrResp, err := client.GetBotQRCode(ctx, defaultBotType)
			if err != nil {
				loginMutex.Lock()
				delete(activeLogins, accountID)
				loginMutex.Unlock()
				return &channel.QrCodeWaitResult{
					Success: false,
					Error:   fmt.Sprintf("refresh QR code: %v", err),
				}, nil
			}

			// Update active login
			loginMutex.Lock()
			login.qrID = qrResp.Qrcode
			login.qrURL = qrResp.QrcodeImgContent
			login.startedAt = time.Now()
			loginMutex.Unlock()

			return &channel.QrCodeWaitResult{
				Success: false,
				Error:   "QR code expired, please scan again",
			}, nil

		case "confirmed":
			// Login successful!
			loginMutex.Lock()
			delete(activeLogins, accountID)
			loginMutex.Unlock()

			// Save account credentials
			account := &weixin.WeChatAccount{
				ID:          accountID,
				Name:        accountID,
				BotToken:    statusResp.BotToken,
				BotID:       statusResp.IlinkBotID,
				UserID:      statusResp.IlinkUserID,
				BaseURL:     statusResp.BaseURL,
				Enabled:     true,
				Configured:  true,
				CreatedAt:   time.Now(),
				LastLoginAt: time.Now(),
			}

			if err := a.plugin.Accounts().Save(account); err != nil {
				return &channel.QrCodeWaitResult{
					Success: false,
					Error:   fmt.Sprintf("save account: %v", err),
				}, nil
			}

			return &channel.QrCodeWaitResult{
				Success:   true,
				BotToken:  statusResp.BotToken,
				AccountID: statusResp.IlinkBotID,
				BaseURL:   statusResp.BaseURL,
				UserID:    statusResp.IlinkUserID,
			}, nil

		default:
			// Unknown status
			time.Sleep(2 * time.Second)
			continue
		}
	}

	// Timeout
	loginMutex.Lock()
	delete(activeLogins, accountID)
	loginMutex.Unlock()

	return &channel.QrCodeWaitResult{
		Success: false,
		Error:   "Login timeout",
	}, nil
}

// IDLabel returns the label used to store the paired user ID.
func (a *PairingAdapter) IDLabel() string {
	return "wechat_user_id"
}

// NormalizeAllowEntry normalizes an entry for the allowlist.
func (a *PairingAdapter) NormalizeAllowEntry(entry string) string {
	return entry
}

// NotifyApproval sends a notification that pairing was approved.
func (a *PairingAdapter) NotifyApproval(ctx context.Context, id string, message string) error {
	// Not implemented for WeChat
	return nil
}
