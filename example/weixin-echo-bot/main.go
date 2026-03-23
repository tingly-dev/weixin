// Package main demonstrates a complete WeChat integration with an echo bot.
//
// This example implements the WeChat login and messaging flow based on the
// OpenClaw WeChat Plugin architecture (tencent-weixin-openclaw-weixin).
//
// Key Components:
// 1. **QR Code Login Flow** (performQRCodeLogin):
//   - Requests QR code from WeChat API (/ilink/bot/get_bot_qrcode)
//   - Displays QR code in terminal for user to scan with WeChat mobile app
//   - Polls status (/ilink/bot/get_qrcode_status) until confirmed
//   - Handles states: wait -> scaned -> confirmed (with auto-refresh on expired)
//   - Saves bot credentials (bot_token, ilink_bot_id, ilink_user_id, baseurl)
//
// 2. **Message Polling Loop** (pollMessages):
//   - Long-polling getUpdates with 35s timeout
//   - Maintains sync buffer (get_updates_buf) for incremental message fetching
//   - Handles session expiry (errcode=-14) with 30min backoff
//   - Processes only user messages (message_type=1)
//
// 3. **Message Handling** (handleMessage):
//   - Extracts context_token from incoming messages (required for replies)
//   - Builds response based on message type (text/image/voice/file/video)
//   - Sends reply via sendMessage with context_token
//
// 4. **Echo Bot Logic**:
//   - Text: Echoes back with "Echo: <text>"
//   - Image: Responds with "📷 Received your image!"
//   - Voice: Responds with "🎤 Received your voice message!"
//   - File: Responds with "📎 Received file: <filename>"
//   - Video: Responds with "🎬 Received your video!"
//
// Architecture Reference:
// - Protocol Adapter: Converts WeChat's HTTP JSON API to Go channel interface
// - Authentication: Bearer token + X-WECHAT-UIN headers
// - Long Polling: Server-side timeout with dynamic adjustment
// - Sync Buffer: Server-maintained cursor for incremental sync
// - Context Token: Session token for maintaining conversation context
//
// For detailed architecture analysis, see:
// tencent-weixin-openclaw-weixin/README.md
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/message"
)

const (
	// Default WeChat API endpoints
	defaultBaseURL    = "https://ilinkai.weixin.qq.com"
	defaultCDNBaseURL = "https://novac2c.cdn.weixin.qq.com/c2c"

	// Bot configuration
	botType          = "3" // iLink bot type
	longPollTimeout  = 35 * time.Second
	accountStorePath = "./wechat_accounts"
	mediaStorePath   = "./wechat_media"
)

// EchoBotService implements a simple echo bot that:
// - Echoes back text messages
// - Responds to images with "Received image: [filename]"
// - Responds to other media with appropriate messages
type EchoBotService struct {
	plugin      *weixin.Plugin
	baseURL     string
	cdnBaseURL  string
	mu          sync.RWMutex
	running     map[string]bool // accountID -> running status
	cancelFuncs map[string]context.CancelFunc
}

// NewEchoBotService creates a new echo bot service.
func NewEchoBotService(plugin *weixin.Plugin, baseURL, cdnBaseURL string) *EchoBotService {
	return &EchoBotService{
		plugin:      plugin,
		baseURL:     baseURL,
		cdnBaseURL:  cdnBaseURL,
		running:     make(map[string]bool),
		cancelFuncs: make(map[string]context.CancelFunc),
	}
}

// Start starts the echo bot for a specific account.
func (s *EchoBotService) Start(ctx context.Context, accountID string) error {
	s.mu.Lock()
	if s.running[accountID] {
		s.mu.Unlock()
		return fmt.Errorf("account %s is already running", accountID)
	}

	// Create cancellable context
	accountCtx, cancel := context.WithCancel(ctx)
	s.cancelFuncs[accountID] = cancel
	s.running[accountID] = true
	s.mu.Unlock()

	log.Printf("[%s] Starting echo bot...\n", accountID)

	// Get account
	account, err := s.plugin.Accounts().Get(accountID)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}

	if !account.Configured || !account.Enabled {
		return fmt.Errorf("account not configured or not enabled")
	}

	// Start message polling loop
	go s.pollMessages(accountCtx, account)

	log.Printf("[%s] Echo bot started successfully\n", accountID)
	return nil
}

// Stop stops the echo bot for a specific account.
func (s *EchoBotService) Stop(accountID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cancel, ok := s.cancelFuncs[accountID]; ok {
		cancel()
		delete(s.cancelFuncs, accountID)
	}
	delete(s.running, accountID)
	log.Printf("[%s] Echo bot stopped\n", accountID)
}

// pollMessages is the main message polling loop.
func (s *EchoBotService) pollMessages(ctx context.Context, account *weixin.WeChatAccount) {
	accountID := account.ID
	client := api.NewClient(account.BaseURL, account.BotToken)

	// Load previous sync buffer if exists
	syncBuf := s.loadSyncBuffer(accountID)
	consecutiveErrors := 0
	maxConsecutiveErrors := 3

	log.Printf("[%s] Starting message polling loop (syncBuf length: %d)\n", accountID, len(syncBuf))

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Polling loop stopped\n", accountID)
			return
		default:
		}

		// Get updates
		resp, err := client.GetUpdates(ctx, syncBuf)
		if err != nil {
			consecutiveErrors++
			log.Printf("[%s] GetUpdates error (%d/%d): %v\n", accountID, consecutiveErrors, maxConsecutiveErrors, err)

			if consecutiveErrors >= maxConsecutiveErrors {
				log.Printf("[%s] Too many consecutive errors, backing off 30s\n", accountID)
				time.Sleep(30 * time.Second)
				consecutiveErrors = 0
			} else {
				time.Sleep(2 * time.Second)
			}
			continue
		}

		// Reset error counter on success
		consecutiveErrors = 0

		// Update sync buffer
		if resp.GetUpdatesBuf != "" {
			syncBuf = resp.GetUpdatesBuf
			s.saveSyncBuffer(accountID, syncBuf)
		}

		// Process messages
		for _, msg := range resp.Messages {
			// Only process user messages
			if msg.MessageType != weixin.MessageTypeUser {
				continue
			}

			log.Printf("[%s] Received message from %s: %d items\n", accountID, msg.FromUserID, len(msg.ItemList))

			// Process in goroutine to avoid blocking polling
			go s.handleMessage(ctx, account, &msg)
		}
	}
}

// handleMessage processes a single incoming message and sends echo response.
func (s *EchoBotService) handleMessage(ctx context.Context, account *weixin.WeChatAccount, msg *weixin.WeixinMessage) {
	accountID := account.ID
	fromUser := msg.FromUserID
	contextToken := msg.ContextToken

	if contextToken == "" {
		log.Printf("[%s] Warning: message has no context_token, skipping\n", accountID)
		return
	}

	client := api.NewClient(account.BaseURL, account.BotToken)

	// Build echo response based on message items
	var responseItems []weixin.MessageItem

	for _, item := range msg.ItemList {
		switch item.Type {
		case weixin.MessageItemTypeText:
			// Echo text back
			if item.TextItem != nil && item.TextItem.Text != "" {
				text := item.TextItem.Text
				log.Printf("[%s] Text from %s: %s\n", accountID, fromUser, text)

				// Echo response
				echoText := fmt.Sprintf("Echo: %s", text)
				responseItems = append(responseItems, message.BuildTextItem(echoText))
			}

		case weixin.MessageItemTypeImage:
			// Respond to image
			log.Printf("[%s] Received image from %s\n", accountID, fromUser)
			responseItems = append(responseItems, message.BuildTextItem("📷 Received your image!"))

		case weixin.MessageItemTypeVoice:
			// Respond to voice
			log.Printf("[%s] Received voice from %s\n", accountID, fromUser)
			responseItems = append(responseItems, message.BuildTextItem("🎤 Received your voice message!"))

		case weixin.MessageItemTypeFile:
			// Respond to file
			fileName := "file"
			if item.FileItem != nil && item.FileItem.FileName != "" {
				fileName = item.FileItem.FileName
			}
			log.Printf("[%s] Received file from %s: %s\n", accountID, fromUser, fileName)
			responseItems = append(responseItems, message.BuildTextItem(fmt.Sprintf("📎 Received file: %s", fileName)))

		case weixin.MessageItemTypeVideo:
			// Respond to video
			log.Printf("[%s] Received video from %s\n", accountID, fromUser)
			responseItems = append(responseItems, message.BuildTextItem("🎬 Received your video!"))
		}
	}

	// Send response if we have items
	if len(responseItems) > 0 {
		err := s.sendMessage(ctx, client, fromUser, contextToken, responseItems)
		if err != nil {
			log.Printf("[%s] Failed to send echo response: %v\n", accountID, err)
		} else {
			log.Printf("[%s] Sent echo response to %s\n", accountID, fromUser)
		}
	}
}

// sendMessage sends a message with the given items.
func (s *EchoBotService) sendMessage(ctx context.Context, client *api.Client, toUserID, contextToken string, items []weixin.MessageItem) error {
	return client.SendMessage(ctx, toUserID, contextToken, items)
}

// loadSyncBuffer loads the sync buffer from disk.
func (s *EchoBotService) loadSyncBuffer(accountID string) string {
	path := filepath.Join(accountStorePath, accountID, "sync_buffer.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// saveSyncBuffer saves the sync buffer to disk.
func (s *EchoBotService) saveSyncBuffer(accountID, syncBuf string) {
	dir := filepath.Join(accountStorePath, accountID)
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "sync_buffer.txt")
	os.WriteFile(path, []byte(syncBuf), 0644)
}

// main demonstrates the complete WeChat integration flow.
//
// This example shows:
// 1. Plugin initialization with WeChat API configuration
// 2. QR code login flow (if no account exists)
// 3. Long-polling message loop for receiving messages
// 4. Echo bot implementation that responds to all message types
//
// Architecture:
// - Uses the OpenClaw WeChat Plugin protocol adapter
// - Communicates with WeChat backend via HTTP JSON API
// - Implements long-polling for message synchronization
// - Maintains sync buffer for incremental message fetching
func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println("")
	log.Println(strings.Repeat("=", 70))
	log.Println("WeChat Echo Bot Example")
	log.Println("Based on: tencent-weixin-openclaw-weixin plugin architecture")
	log.Println(strings.Repeat("=", 70))
	log.Println("")

	// Create necessary directories
	os.MkdirAll(accountStorePath, 0755)
	os.MkdirAll(mediaStorePath, 0755)
	log.Printf("📁 Storage directories:")
	log.Printf("   - Accounts: %s\n", accountStorePath)
	log.Printf("   - Media: %s\n", mediaStorePath)
	log.Println("")

	// Initialize WeChat plugin
	config := &weixin.WeChatConfig{
		BaseURL: defaultBaseURL,
		BotType: botType,
	}

	plugin := weixin.NewPlugin(config)
	log.Printf("✓ WeChat plugin initialized (base URL: %s, bot type: %s)\n", defaultBaseURL, botType)
	log.Println("")

	// Create echo bot service
	echoBot := NewEchoBotService(plugin, defaultBaseURL, defaultCDNBaseURL)

	// Check for existing accounts
	accounts, err := plugin.Accounts().ListIDs()
	if err != nil {
		log.Fatalf("❌ Failed to list accounts: %v", err)
	}

	var accountID string
	if len(accounts) == 0 {
		// No accounts, need to login via QR code
		log.Println("ℹ️  No accounts found in local storage.")
		log.Println("   Starting WeChat QR code login flow...")
		log.Println("")

		accountID, err = performQRCodeLogin(plugin, defaultBaseURL)
		if err != nil {
			log.Fatalf("❌ QR code login failed: %v", err)
		}

		log.Println("")
		log.Printf("🎉 Login successful! Account ID: %s\n", accountID)
	} else {
		// Use first available account
		accountID = accounts[0]
		log.Printf("ℹ️  Found %d existing account(s) in local storage\n", len(accounts))
		log.Printf("   Using account: %s\n", accountID)

		account, err := plugin.Accounts().Get(accountID)
		if err != nil {
			log.Fatalf("❌ Failed to get account: %v", err)
		}

		log.Println("")
		log.Println("Account Details:")
		log.Printf("  - ID: %s\n", account.ID)
		log.Printf("  - Name: %s\n", account.Name)
		log.Printf("  - Bot ID: %s\n", account.BotID)
		log.Printf("  - User ID: %s\n", account.UserID)
		log.Printf("  - Base URL: %s\n", account.BaseURL)
		log.Printf("  - Enabled: %v\n", account.Enabled)
		log.Printf("  - Configured: %v\n", account.Configured)
		log.Printf("  - Last Login: %s\n", account.LastLoginAt.Format("2006-01-02 15:04:05"))
	}

	log.Println("")
	log.Println(strings.Repeat("-", 70))
	log.Println("")

	// Start echo bot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("🚀 Starting echo bot for account: %s\n", accountID)
	if err := echoBot.Start(ctx, accountID); err != nil {
		log.Fatalf("❌ Failed to start echo bot: %v", err)
	}

	// Wait for interrupt signal
	log.Println("")
	log.Println(strings.Repeat("=", 70))
	log.Println("✅ Echo bot is now running!")
	log.Println(strings.Repeat("=", 70))
	log.Println("")
	log.Println("💬 How to test:")
	log.Println("   1. Open WeChat on your mobile device")
	log.Println("   2. Send a message to this bot")
	log.Println("   3. The bot will echo back your message")
	log.Println("")
	log.Println("Supported message types:")
	log.Println("   - Text: Will echo back with 'Echo: <your text>'")
	log.Println("   - Image: Will respond with '📷 Received your image!'")
	log.Println("   - Voice: Will respond with '🎤 Received your voice message!'")
	log.Println("   - File: Will respond with '📎 Received file: <filename>'")
	log.Println("   - Video: Will respond with '🎬 Received your video!'")
	log.Println("")
	log.Println("Press Ctrl+C to stop the bot")
	log.Println("")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("")
	log.Println(strings.Repeat("=", 70))
	log.Println("🛑 Shutdown signal received, stopping echo bot...")
	echoBot.Stop(accountID)
	log.Println("✓ Echo bot stopped successfully")
	log.Println("👋 Goodbye!")
	log.Println(strings.Repeat("=", 70))
}

// performQRCodeLogin performs the WeChat QR code login flow.
//
// Login Flow (based on OpenClaw WeChat Plugin architecture):
// 1. Request QR Code - Call /ilink/bot/get_bot_qrcode with bot_type parameter
// 2. Display QR Code - Show QR code image/text for user to scan with WeChat
// 3. Poll Status - Long-poll /ilink/bot/get_qrcode_status until status changes
// 4. Handle States:
//   - "wait": Initial state, waiting for scan
//   - "scaned": QR code scanned, waiting for user confirmation in WeChat
//   - "confirmed": Login successful, returns bot_token, ilink_bot_id, baseurl, ilink_user_id
//   - "expired": QR code expired (auto-refresh up to 3 times)
//
// Reference: tencent-weixin-openclaw-weixin/README.md lines 214-256
func performQRCodeLogin(plugin *weixin.Plugin, baseURL string) (string, error) {
	ctx := context.Background()
	client := api.NewClient(baseURL, "")

	const maxQRRefreshCount = 3
	qrRefreshCount := 0
	scannedPrinted := false

	log.Println("\n" + strings.Repeat("=", 70))
	log.Println("WeChat QR Code Login Flow")
	log.Println(strings.Repeat("=", 70))

	// Outer loop: QR code refresh on expiry
	for qrRefreshCount < maxQRRefreshCount {
		// === Step 1: Request QR Code ===
		if qrRefreshCount == 0 {
			log.Println("\n[Step 1/3] Requesting QR code from WeChat API...")
		} else {
			log.Printf("\n⏳ QR code expired, refreshing... (attempt %d/%d)\n", qrRefreshCount+1, maxQRRefreshCount)
		}

		qrResp, err := client.GetBotQRCode(ctx, botType)
		if err != nil {
			return "", fmt.Errorf("get QR code: %w", err)
		}

		// Validate response
		if qrResp.Qrcode == "" {
			return "", fmt.Errorf("API returned empty qrcode field (img_content: %d bytes)", len(qrResp.QrcodeImgContent))
		}

		log.Printf("✓ Received QR code token: %s (length: %d)\n", truncateString(qrResp.Qrcode, 40), len(qrResp.Qrcode))
		if qrResp.QrcodeImgContent != "" {
			log.Printf("✓ Received QR code image: %d bytes (base64)\n", len(qrResp.QrcodeImgContent))
		}

		// === Step 2: Display QR Code ===
		log.Println("\n[Step 2/3] Please scan the QR code below with WeChat:")
		log.Println("")

		// IMPORTANT: Use QrcodeImgContent (not Qrcode) to generate the scannable QR code
		// - qrcode: Server-side identifier/token for polling status
		// - qrcode_img_content: The actual data to encode in QR code (URL or encoded string)
		qrDataToDisplay := qrResp.QrcodeImgContent
		if qrDataToDisplay == "" {
			log.Println("⚠ Warning: API returned empty qrcode_img_content, falling back to qrcode field")
			qrDataToDisplay = qrResp.Qrcode
		}

		if err := api.DisplayQRCodeInTerminal(qrDataToDisplay, false); err != nil {
			log.Printf("⚠ Warning: Failed to render QR code in terminal: %v\n", err)
			log.Printf("   QR Code Data: %s\n", qrDataToDisplay)
			log.Println("   (You can manually generate a QR code from this data if needed)")
		} else {
			log.Println("")
			log.Printf("QR Code Token (for status polling): %s\n", truncateString(qrResp.Qrcode, 40))
			if qrResp.QrcodeImgContent != "" {
				log.Printf("QR Code Content (what you scan): %s\n", truncateString(qrResp.QrcodeImgContent, 60))
			}
		}

		log.Println("\n💡 Instructions:")
		log.Println("   1. Open WeChat on your mobile device")
		log.Println("   2. Go to 'Discover' -> 'Scan QR Code'")
		log.Println("   3. Scan the QR code displayed above")
		log.Println("   4. Confirm the login on your mobile device")
		log.Println("")

		// === Step 3: Poll Status Until Confirmed ===
		log.Println("[Step 3/3] Waiting for QR code scan and confirmation...")
		log.Println("          (This will timeout in 5 minutes if not scanned)")
		log.Println("")

		waitCtx, waitCancel := context.WithTimeout(ctx, 5*time.Minute)
		defer waitCancel()

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		// Inner loop: Poll status for this QR code
		qrExpired := false
		pollCount := 0

		for !qrExpired {
			select {
			case <-waitCtx.Done():
				return "", fmt.Errorf("login timeout after 5 minutes (QR code was not scanned)")
			case <-ticker.C:
				pollCount++

				statusResp, err := client.GetQRStatus(ctx, qrResp.Qrcode)
				if err != nil {
					log.Printf("⚠ Poll #%d - GetQRStatus error (will retry): %v\n", pollCount, err)
					continue
				}

				// Log status transitions
				switch statusResp.Status {
				case "wait":
					// Still waiting for scan
					if pollCount%10 == 0 { // Log every 20 seconds
						log.Printf("⏳ Still waiting for scan... (%d seconds elapsed)\n", pollCount*2)
					}
					continue

				case "scaned":
					// QR code has been scanned, waiting for user confirmation
					if !scannedPrinted {
						log.Println("")
						log.Println("👀 QR code scanned successfully!")
						log.Println("   Please confirm the login on your WeChat mobile app...")
						log.Println("")
						scannedPrinted = true
					}
					continue

				case "confirmed":
					// Login successful!
					log.Println("")
					log.Println("✅ Login confirmed successfully!")
					log.Println("")

					// Validate response fields
					if statusResp.BotToken == "" {
						return "", fmt.Errorf("login confirmed but no bot_token returned in response")
					}
					if statusResp.IlinkBotID == "" {
						return "", fmt.Errorf("login confirmed but no ilink_bot_id returned in response")
					}

					// Use server-provided baseURL if available, otherwise fallback to default
					accountBaseURL := baseURL
					if statusResp.BaseURL != "" {
						accountBaseURL = statusResp.BaseURL
						log.Printf("📍 Using server-provided base URL: %s\n", accountBaseURL)
					}

					log.Println("Account credentials received:")
					log.Printf("  - Bot ID: %s\n", statusResp.IlinkBotID)
					log.Printf("  - User ID: %s\n", statusResp.IlinkUserID)
					log.Printf("  - Bot Token: %s...%s\n", statusResp.BotToken[:8], statusResp.BotToken[len(statusResp.BotToken)-8:])
					log.Printf("  - Base URL: %s\n", accountBaseURL)
					log.Println("")

					// Create account
					account := &weixin.WeChatAccount{
						ID:          normalizeAccountID(statusResp.IlinkBotID),
						Name:        statusResp.IlinkBotID,
						BotToken:    statusResp.BotToken,
						BotID:       statusResp.IlinkBotID,
						UserID:      statusResp.IlinkUserID,
						BaseURL:     accountBaseURL,
						Enabled:     true,
						Configured:  true,
						CreatedAt:   time.Now(),
						LastLoginAt: time.Now(),
					}

					// Save account
					log.Println("💾 Saving account credentials...")
					if err := plugin.Accounts().Save(account); err != nil {
						return "", fmt.Errorf("save account: %w", err)
					}

					log.Printf("✓ Account saved successfully: %s\n", account.ID)
					log.Println(strings.Repeat("=", 70))
					return account.ID, nil

				case "expired":
					// QR code expired, will refresh in outer loop
					log.Println("")
					log.Println("⏰ QR code expired (QR codes expire after ~2 minutes of inactivity)")
					qrExpired = true
					scannedPrinted = false // Reset for next QR
					log.Println("")

				default:
					log.Printf("⚠ Unknown QR status received: %s (will continue polling)\n", statusResp.Status)
				}
			}
		}

		// Increment refresh counter and try again
		qrRefreshCount++
	}

	// Exceeded max refresh attempts
	return "", fmt.Errorf("login failed: QR code expired %d times without being scanned", maxQRRefreshCount)
}

// truncateString truncates a string to maxLen and adds "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// normalizeAccountID normalizes an account ID for filesystem safety.
func normalizeAccountID(id string) string {
	// Replace @ with - and other special chars
	normalized := strings.ReplaceAll(id, "@", "-")
	normalized = strings.ReplaceAll(normalized, ".", "-")
	return normalized
}
