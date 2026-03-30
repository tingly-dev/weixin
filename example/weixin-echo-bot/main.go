// Package main demonstrates a WeChat echo bot using the channel abstraction layer.
//
// This example shows the recommended way to integrate with WeChat:
//
//  1. Create a plugin and initialize its adapters
//  2. Login via QR code using PairingAdapter (if no account exists)
//  3. Poll messages using LongPollAdapter.GetUpdates
//  4. Send echo replies using ActionsAdapter.Send
//
// All low-level details (sync buffer persistence, context token caching,
// session guard, error backoff) are handled by the adapters.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/adapters"
	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/channel"
)

const (
	defaultBaseURL    = "https://ilinkai.weixin.qq.com"
	longPollTimeoutMs = 35000
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println(strings.Repeat("=", 60))
	log.Println("WeChat Echo Bot (channel abstraction)")
	log.Println(strings.Repeat("=", 60))

	// Create plugin with pwd as data directory and initialize adapters
	config := &weixin.WeChatConfig{
		BaseURL: defaultBaseURL,
		BotType: "3",
	}
	plugin := weixin.NewPluginWithDataDir(config, ".")
	adapters.InitPlugin(plugin)

	// Resolve or create account
	accountID, err := ensureAccount(plugin)
	if err != nil {
		log.Fatalf("Failed to get account: %v", err)
	}

	log.Printf("Using account: %s\n", accountID)

	// Start echo bot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go pollLoop(ctx, plugin, accountID)

	log.Println(strings.Repeat("=", 60))
	log.Println("Echo bot is running. Send a message to test.")
	log.Println("Press Ctrl+C to stop.")
	log.Println(strings.Repeat("=", 60))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()
	log.Println("Goodbye!")
}

// ensureAccount returns an existing account ID or runs QR login to create one.
func ensureAccount(plugin *weixin.Plugin) (string, error) {
	ids, err := plugin.Accounts().ListIDs()
	if err != nil {
		return "", err
	}
	if len(ids) > 0 {
		return ids[0], nil
	}

	return qrLogin(plugin)
}

// qrLogin performs QR code login via the PairingAdapter.
func qrLogin(plugin *weixin.Plugin) (string, error) {
	ctx := context.Background()
	accountID := "default"

	log.Println("No account found. Starting QR code login...")

	// Step 1: Get QR code
	qrResult, err := plugin.Pairing().LoginWithQrStart(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("get QR code: %w", err)
	}

	log.Println("Scan this QR code with WeChat:")
	if err := api.DisplayQRCodeInTerminal(qrResult.QrCodeData, false); err != nil {
		log.Printf("Failed to render QR code: %v\n", err)
		log.Printf("QR data: %s\n", qrResult.QrCodeData)
	} else {
		log.Println("")
	}

	// Step 2: Wait for confirmation
	log.Println("Waiting for scan and confirmation...")
	waitResult, err := plugin.Pairing().LoginWithQrWait(ctx, accountID, qrResult.QrCodeID)
	if err != nil {
		return "", fmt.Errorf("QR login: %w", err)
	}
	if !waitResult.Success {
		return "", fmt.Errorf("QR login failed: %s", waitResult.Error)
	}

	log.Printf("Login successful! Account: %s\n", accountID)
	return accountID, nil
}

// pollLoop continuously polls for messages and echoes them back.
func pollLoop(ctx context.Context, plugin *weixin.Plugin, accountID string) {
	syncBuf := ""
	backoff := 2 * time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		result, err := plugin.LongPoll().GetUpdates(ctx, &channel.GetUpdatesRequest{
			AccountID: accountID,
			SyncBuf:   syncBuf,
			TimeoutMs: longPollTimeoutMs,
		})
		if err != nil {
			log.Printf("GetUpdates error: %v\n", err)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Reset backoff on success
		backoff = 2 * time.Second

		// Check for session errors
		if result.ErrCode != 0 {
			log.Printf("Session error (code=%d): %s\n", result.ErrCode, result.ErrMsg)
			time.Sleep(30 * time.Second)
			continue
		}

		// Advance sync buffer
		if result.SyncBuf != "" {
			syncBuf = result.SyncBuf
		}

		// Process each message
		for i, msg := range result.Messages {
			msg := msg // capture loop variable
			go handleMessage(ctx, plugin, accountID, msg, i)
		}
	}
}

// handleMessage processes a single message and sends an echo reply.
func handleMessage(ctx context.Context, plugin *weixin.Plugin, accountID string, msg *channel.Message, idx int) {
	log.Printf("[%s] Message #%d from %s: %q (attachments: %d)\n",
		accountID, idx, msg.SenderID, msg.Text, len(msg.Attachments))

	// Build echo response
	var replyText string
	switch {
	case msg.Text != "":
		replyText = fmt.Sprintf("Echo: %s", msg.Text)
	case len(msg.Attachments) > 0:
		att := msg.Attachments[0]
		switch att.ContentType {
		case "image":
			replyText = "Received your image!"
		case "audio":
			replyText = "Received your voice message!"
		case "video":
			replyText = "Received your video!"
		default:
			replyText = fmt.Sprintf("Received file: %s", att.FileName)
		}
	default:
		return // nothing to echo
	}

	// Get context token — prefer from message, fall back to metadata
	contextToken := msg.ContextToken
	if contextToken == "" {
		if ct, ok := msg.Metadata["context_token"]; ok {
			if s, ok := ct.(string); ok {
				contextToken = s
			}
		}
	}

	result, err := plugin.Actions().Send(ctx, &channel.OutboundMessage{
		AccountID:    accountID,
		To:           msg.To,
		Text:         replyText,
		ContextToken: contextToken,
	})
	if err != nil {
		log.Printf("[%s] Send error: %v\n", accountID, err)
		return
	}
	if !result.OK {
		log.Printf("[%s] Send failed: %s\n", accountID, result.Error)
		return
	}
	log.Printf("[%s] Echo sent to %s\n", accountID, msg.SenderID)
}
