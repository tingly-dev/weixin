// Package main demonstrates WeChat block streaming using the channel abstraction layer.
//
// This example shows how to simulate streaming text output by sending
// multiple text chunks via Send(). WeChat's ilink protocol doesn't support
// true streaming — instead, "block streaming" sends each chunk as a separate
// message. The framework coalesces chunks locally (minChars: 200, idleMs: 3000)
// before delivering them.
//
// Usage: go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/plugin"
)

const (
	defaultBaseURL        = "https://ilinkai.weixin.qq.com"
	longPollTimeoutMs     = 35000
	maxConcurrentHandlers = 10
)

// Streaming demo responses.
var demoPhrases = []string{
	"在测试流式输出！每条消息都是一个独立的块...\n\n",
	"这是第二块。你可以看到消息是逐条到达的，",
	"而不是一次性全部出现。\n\n",
	"这就是 WeChat 的 block streaming 模式 —— ",
	"每个 chunk 都通过 sendMessage API 发送，",
	"协议本身不支持真正的流式传输。\n\n",
	"流式输出完成！",
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println(strings.Repeat("=", 60))
	log.Println("WeChat Stream Bot (block streaming demo)")
	log.Println(strings.Repeat("=", 60))

	// Create plugin with pwd as data directory and initialize adapters
	config := &weixin.WeChatConfig{
		BaseURL: defaultBaseURL,
		BotType: "3",
	}
	plugin := plugin.NewPluginWithDataDir(config, ".")

	// Resolve or create account
	accountID, err := ensureAccount(plugin)
	if err != nil {
		log.Fatalf("Failed to get account: %v", err)
	}

	log.Printf("Using account: %s\n", accountID)

	// Start stream bot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go pollLoop(ctx, plugin, accountID)

	log.Println(strings.Repeat("=", 60))
	log.Println("Stream bot is running. Send a message to test streaming.")
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
func ensureAccount(plugin *plugin.Plugin) (string, error) {
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
func qrLogin(plugin *plugin.Plugin) (string, error) {
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

// pollLoop continuously polls for messages and responds with streamed chunks.
func pollLoop(ctx context.Context, plugin *plugin.Plugin, accountID string) {
	syncBuf := ""
	backoff := 2 * time.Second
	const maxBackoff = 30 * time.Second
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentHandlers)

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		default:
		}

		result, err := plugin.LongPoll().GetUpdates(ctx, &weixin.GetUpdatesRequest{
			AccountID: accountID,
			SyncBuf:   syncBuf,
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

		// Process each message with bounded concurrency
		for i, msg := range result.Messages {
			msg := msg
			sem <- struct{}{} // acquire slot
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				defer func() { <-sem }()
				handleMessage(ctx, plugin, accountID, msg, idx)
			}(i)
		}
	}
}

// handleMessage processes a single message and sends a streamed reply.
func handleMessage(ctx context.Context, plugin *plugin.Plugin, accountID string, msg *weixin.Message, idx int) {
	if msg == nil {
		return
	}

	log.Printf("[%s] Message #%d from %s: %q\n", accountID, idx, msg.SenderID, msg.Text)

	if msg.Text == "" {
		return
	}

	// Resolve context token
	contextToken := msg.ContextToken
	if contextToken == "" {
		if ct, ok := msg.Metadata["context_token"]; ok {
			if s, ok := ct.(string); ok {
				contextToken = s
			}
		}
	}

	// Simulate block streaming: send each phrase as a separate message
	log.Printf("[%s] Starting stream to %s (%d chunks)\n", accountID, msg.SenderID, len(demoPhrases))
	for i, chunk := range demoPhrases {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Stream cancelled at chunk %d\n", accountID, i)
			return
		default:
		}

		result, err := plugin.Actions().Send(ctx, &weixin.OutboundMessage{
			AccountID:    accountID,
			To:           msg.To,
			Text:         chunk,
			ContextToken: contextToken,
		})
		if err != nil {
			log.Printf("[%s] Stream chunk %d error: %v\n", accountID, i, err)
			return
		}
		if !result.OK {
			log.Printf("[%s] Stream chunk %d failed: %s\n", accountID, i, result.Error)
			return
		}

		log.Printf("[%s] Stream chunk %d/%d sent\n", accountID, i+1, len(demoPhrases))

		// Simulate token generation delay (like an LLM producing output)
		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("[%s] Stream complete to %s\n", accountID, msg.SenderID)
}
