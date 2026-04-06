// Package main demonstrates WeChat block streaming.
//
// This example shows how to simulate streaming text output by sending
// multiple text chunks via Send(). WeChat's ilink protocol doesn't support
// true streaming — instead, "block streaming" sends each chunk as a separate
// message.
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

	"github.com/tingly-dev/weixin/types"
	"github.com/tingly-dev/weixin/wechat"
	api "github.com/tingly-dev/weixin/wechat/api"
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

	// Create bot with pwd as data directory
	config := &types.WeChatConfig{
		BaseURL: defaultBaseURL,
		BotType: "3",
	}
	bot, err := wechat.NewWechatBotWithDataDir(config, ".")
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Resolve or create account
	accountID, err := ensureAccount(bot)
	if err != nil {
		log.Fatalf("Failed to get account: %v", err)
	}

	log.Printf("Using account: %s\n", accountID)

	// Start stream bot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go pollLoop(ctx, bot)

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
func ensureAccount(bot *wechat.WechatBot) (string, error) {
	ids, err := bot.AccountManager().ListIDs()
	if err != nil {
		return "", err
	}
	if len(ids) > 0 {
		// Load the first account
		if err := bot.LoadAccount(ids[0]); err != nil {
			return "", err
		}
		return ids[0], nil
	}

	return qrLogin(bot)
}

// qrLogin performs QR code login.
func qrLogin(bot *wechat.WechatBot) (string, error) {
	ctx := context.Background()
	accountID := "default"

	log.Println("No account found. Starting QR code login...")

	// Step 1: Get QR code
	qrResult, err := bot.LoginWithQrStart(ctx, accountID)
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
	waitResult, err := bot.LoginWithQrWait(ctx, accountID, qrResult.QrCodeID)
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
func pollLoop(ctx context.Context, bot *wechat.WechatBot) {
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

		result, err := bot.GetUpdates(ctx, syncBuf)
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
				handleMessage(ctx, bot, msg, idx)
			}(i)
		}
	}
}

// handleMessage processes a single message and sends a streamed reply.
func handleMessage(ctx context.Context, bot *wechat.WechatBot, msg *types.Message, idx int) {
	if msg == nil {
		return
	}

	log.Printf("Message #%d from %s: %q\n", idx, msg.SenderID, msg.Text)

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
	log.Printf("Starting stream to %s (%d chunks)\n", msg.SenderID, len(demoPhrases))
	for i, chunk := range demoPhrases {
		select {
		case <-ctx.Done():
			log.Printf("Stream cancelled at chunk %d\n", i)
			return
		default:
		}

		result, err := bot.Send(ctx, &types.OutboundMessage{
			To:           msg.To,
			Text:         chunk,
			ContextToken: contextToken,
		})
		if err != nil {
			log.Printf("Stream chunk %d error: %v\n", i, err)
			return
		}
		if !result.OK {
			log.Printf("Stream chunk %d failed: %s\n", i, result.Error)
			return
		}

		log.Printf("Stream chunk %d/%d sent\n", i+1, len(demoPhrases))

		// Simulate token generation delay (like an LLM producing output)
		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("Stream complete to %s\n", msg.SenderID)
}
