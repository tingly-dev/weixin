package wechat_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/tingly-dev/weixin/message"
	"github.com/tingly-dev/weixin/types"
	"github.com/tingly-dev/weixin/wechat"
	"github.com/tingly-dev/weixin/wechat/api"
)

// loadTestAccount loads account from default.json in the project root.
func loadTestAccount(t *testing.T) *types.WeChatAccount {
	t.Helper()

	// Locate project root (one level up from this file's directory)
	_, thisFile, _, _ := runtime.Caller(0)
	configPath := filepath.Join(filepath.Dir(thisFile), "..", "default.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Skipf("skip e2e: default.json not found: %v", err)
	}

	var account types.WeChatAccount
	if err := json.Unmarshal(data, &account); err != nil {
		t.Fatalf("parse default.json: %v", err)
	}

	if account.BotToken == "" || account.UserID == "" {
		t.Skip("skip e2e: default.json missing botToken or userId")
	}

	if account.CDNBaseURL == "" {
		account.CDNBaseURL = wechat.DefaultCDNBaseURL
	}

	return &account
}

func newTestBot(t *testing.T, account *types.WeChatAccount) *wechat.WechatBot {
	t.Helper()
	bot, err := wechat.NewWechatBot(wechat.WithAccount(account))
	if err != nil {
		t.Fatalf("create bot: %v", err)
	}
	return bot
}

func TestE2E_SendText(t *testing.T) {
	account := loadTestAccount(t)
	bot := newTestBot(t, account)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := bot.Send(ctx, &types.OutboundMessage{
		To:   account.UserID,
		Text: "e2e test: text message @ " + time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !result.OK {
		t.Fatalf("Send not OK: %s", result.Error)
	}
	t.Log("text message sent OK")
}

func TestE2E_SendFile_README(t *testing.T) {
	account := loadTestAccount(t)
	bot := newTestBot(t, account)

	// Locate README.md
	_, thisFile, _, _ := runtime.Caller(0)
	readmePath := filepath.Join(filepath.Dir(thisFile), "..", "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		t.Skipf("skip: README.md not found: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := bot.SendMedia(ctx, &types.OutboundMessage{
		To:       account.UserID,
		FilePath: readmePath,
		FileName: "README.md",
		// ContentType empty → defaults to file
	})
	if err != nil {
		t.Fatalf("SendMedia (file): %v", err)
	}
	if !result.OK {
		t.Fatalf("SendMedia not OK: %s", result.Error)
	}
	t.Log("README.md file sent OK")
}

// TestE2E_GetUpdatesParsesWire verifies that a real getupdates response
// unmarshals cleanly and the sync cursor advances. This is the regression
// guard for the "can send but not receive" bug: the server sends
// item_list[].msg_id as a prefixed string ("v1:<digits>"), which a uint64
// field would reject, silently dropping every inbound message.
//
// It uses its own cursor (starting from an empty get_updates_buf) and never
// touches the persisted per-account sync buffer, so it does not interfere
// with a concurrently running monitor.
func TestE2E_GetUpdatesParsesWire(t *testing.T) {
	account := loadTestAccount(t)
	bot := newTestBot(t, account)
	client := bot.Client()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Poll 1: empty buf acquires a fresh cursor (usually returns immediately).
	r1, err := client.GetUpdatesWithTimeout(ctx, "", 20*time.Second)
	if err != nil {
		t.Fatalf("GetUpdates(empty buf): %v", err)
	}
	if r1.GetUpdatesBuf == "" {
		t.Skip("skip: server did not return a cursor within the poll window")
	}
	t.Logf("cursor acquired (%d bytes)", len(r1.GetUpdatesBuf))

	// Poll 2: with cursor. Any backlog of user messages must parse cleanly;
	// a timeout with no messages is also a pass (nothing to receive).
	r2, err := client.GetUpdatesWithTimeout(ctx, r1.GetUpdatesBuf, 10*time.Second)
	if err != nil {
		t.Fatalf("GetUpdates(with cursor): %v", err)
	}
	for _, msg := range r2.Messages {
		for _, item := range msg.ItemList {
			t.Logf("parsed msg id=%s item type=%d msg_id=%s", msg.MessageID, item.Type, item.MsgID)
		}
	}
	t.Logf("wire parse OK: %d message(s)", len(r2.Messages))
}

// TestE2E_ReceiveInteractive is a human-in-the-loop receive test: it sends a
// prompt to the paired user, long-polls until that user replies, then echoes
// the reply back so the human can see the round-trip completed.
//
// Skipped unless WEIXIN_E2E_RECEIVE=1 is set, since it needs a human to reply
// from the WeChat client:
//
//	WEIXIN_E2E_RECEIVE=1 go test ./wechat -run TestE2E_ReceiveInteractive -v
func TestE2E_ReceiveInteractive(t *testing.T) {
	if os.Getenv("WEIXIN_E2E_RECEIVE") == "" {
		t.Skip("skip interactive receive e2e: set WEIXIN_E2E_RECEIVE=1 to run")
	}

	account := loadTestAccount(t)
	bot := newTestBot(t, account)
	client := bot.Client()

	const replyWindow = 90 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), replyWindow+30*time.Second)
	defer cancel()

	// Acquire a cursor BEFORE sending the prompt so the reply is guaranteed
	// to be ahead of it.
	r1, err := client.GetUpdatesWithTimeout(ctx, "", 20*time.Second)
	if err != nil {
		t.Fatalf("GetUpdates(empty buf): %v", err)
	}
	buf := r1.GetUpdatesBuf
	if buf == "" {
		t.Fatal("no cursor acquired")
	}

	prompt := "【e2e 收消息测试】请在 90 秒内回复任意消息 @ " + time.Now().Format(time.RFC3339)
	if _, err := bot.Send(ctx, &types.OutboundMessage{To: account.UserID, Text: prompt}); err != nil {
		t.Fatalf("send prompt: %v", err)
	}
	t.Log("prompt sent, waiting for a human reply from WeChat...")

	deadline := time.Now().Add(replyWindow)
	for time.Now().Before(deadline) {
		resp, err := client.GetUpdatesWithTimeout(ctx, buf, 30*time.Second)
		if err != nil {
			t.Fatalf("GetUpdates: %v", err)
		}
		if resp.GetUpdatesBuf != "" {
			buf = resp.GetUpdatesBuf
		}
		for _, msg := range resp.Messages {
			if msg.MessageType != api.MessageTypeUser {
				continue
			}
			converted := message.ConvertInboundMessage(&msg, account.ID, account.CDNBaseURL)
			if converted == nil {
				t.Fatalf("received user message %s but conversion returned nil", msg.MessageID)
			}
			t.Logf("received reply: id=%s text=%q", converted.MessageID, converted.Text)

			// Echo back so the human sees the test completed. Reply with the
			// received context_token so it lands in the same conversation
			// context.
			echo := &types.OutboundMessage{
				To:           account.UserID,
				Text:         "【e2e 收消息测试通过】已收到你的回复: " + converted.Text,
				ContextToken: msg.ContextToken,
			}
			if _, err := bot.Send(ctx, echo); err != nil {
				t.Fatalf("send completion echo: %v", err)
			}
			t.Log("completion echo sent")
			return
		}
	}
	t.Fatalf("no user reply received within %v", replyWindow)
}
