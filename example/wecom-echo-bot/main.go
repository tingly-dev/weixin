// Package main demonstrates a WeCom AI Bot echo bot.
//
// This example shows the recommended way to integrate with WeCom AI Bot:
//
//  1. Create a WecomBot
//  2. Configure account credentials (BotID/Secret)
//  3. Set an EventHandler to receive push messages via WebSocket
//  4. Send echo replies using ActionsAdapter.Send
//
// WeCom uses WebSocket (push-based), not HTTP long-polling.
// All connection lifecycle (heartbeat, reconnect) is handled by the adapters.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/tingly-dev/weixin/types"
	"github.com/tingly-dev/weixin/wecom"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println(strings.Repeat("=", 60))
	log.Println("WeCom AI Bot Echo Bot")
	log.Println(strings.Repeat("=", 60))

	botID := os.Getenv("WECOM_BOT_ID")
	secret := os.Getenv("WECOM_SECRET")
	if botID == "" || secret == "" {
		log.Fatal("WECOM_BOT_ID and WECOM_SECRET environment variables are required")
	}

	accountID := "default"

	// Create WeCom bot
	bot := wecom.NewWecomBot(&wecom.WecomConfig{
		Logger: nil,
	})

	// Configure account credentials
	bot.Gateway().SetAccountConfig(accountID, wecom.ClientConfig{
		BotID:  botID,
		Secret: secret,
	})

	// Set message handler
	handler := &echoHandler{bot: bot, accountID: accountID}
	bot.Gateway().SetEventHandler(accountID, handler)

	// Connect
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("Connecting to WeCom WebSocket (bot: %s)...\n", botID)
	if err := bot.Gateway().StartAccount(ctx, accountID); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	log.Println(strings.Repeat("=", 60))
	log.Println("Echo bot is running. Send a message to test.")
	log.Println("Press Ctrl+C to stop.")
	log.Println(strings.Repeat("=", 60))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	bot.Gateway().StopAccount(ctx, accountID)
	log.Println("Goodbye!")
}

// echoHandler implements types.EventHandler for the echo bot.
type echoHandler struct {
	bot       *wecom.WecomBot
	accountID string
}

// OnMessage handles incoming messages and echoes them back.
func (h *echoHandler) OnMessage(ctx context.Context, msg *types.Message) error {
	log.Printf("Message from %s: %q (attachments: %d)\n",
		msg.SenderID, msg.Text, len(msg.Attachments))

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
		return nil
	}

	result, err := h.bot.Actions().Send(ctx, &types.OutboundMessage{
		AccountID:    h.accountID,
		To:           msg.To,
		Text:         replyText,
		ContextToken: msg.ContextToken, // req_id for passive reply
	})
	if err != nil {
		return fmt.Errorf("send reply: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("send failed: %s", result.Error)
	}

	log.Printf("Echo sent to %s\n", msg.SenderID)
	return nil
}

// OnReaction handles reactions (not used).
func (h *echoHandler) OnReaction(ctx context.Context, reaction *types.Reaction) error {
	return nil
}

// OnEdit handles message edits (not supported by WeCom).
func (h *echoHandler) OnEdit(ctx context.Context, msg *types.Message) error {
	return nil
}

// OnEvent handles protocol events (enter_chat, card_click, etc.).
func (h *echoHandler) OnEvent(ctx context.Context, event *types.Event) {
	log.Printf("Event: %s (payload: %v)\n", event.EventType, event.Payload)
}

// OnError handles errors.
func (h *echoHandler) OnError(ctx context.Context, err error) {
	log.Printf("Error: %v\n", err)
}
