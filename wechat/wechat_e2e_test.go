package wechat_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/tingly-dev/weixin/types"
	"github.com/tingly-dev/weixin/wechat"
)

const (
	defaultCDNBaseURL = "https://novac2c.cdn.weixin.qq.com/c2c"
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
		account.CDNBaseURL = defaultCDNBaseURL
	}

	return &account
}

func newTestBot(t *testing.T, account *types.WeChatAccount) *wechat.WechatBot {
	t.Helper()
	bot, err := wechat.NewWechatBotWithAccount(&types.WeChatConfig{BaseURL: account.BaseURL}, account)
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
