// Package wechat provides the WeChat ilink bot implementation.
package wechat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tingly-dev/weixin/types"
)

// newLifecycleServer returns an httptest server that counts hits on the
// notifyStart / notifyStop endpoints. The handler returns the supplied
// status code so tests can simulate transport-level failures.
func newLifecycleServer(t *testing.T, status int) (srv *httptest.Server, starts, stops *int32) {
	t.Helper()
	var s, p int32
	starts, stops = &s, &p

	mux := http.NewServeMux()
	mux.HandleFunc("/ilink/bot/msg/notifystart", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(starts, 1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"ret":0}`))
	})
	mux.HandleFunc("/ilink/bot/msg/notifystop", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(stops, 1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"ret":0}`))
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, starts, stops
}

func newConfiguredAccount(baseURL string) *types.WeChatAccount {
	return &types.WeChatAccount{
		ID:         "test-account",
		BotToken:   "test-token",
		BotID:      "test-bot",
		UserID:     "test-user",
		BaseURL:    baseURL,
		Enabled:    true,
		Configured: true,
	}
}

func TestConnectFiresNotifyStartByDefault(t *testing.T) {
	srv, starts, stops := newLifecycleServer(t, http.StatusOK)

	bot, err := NewWechatBot(WithAccount(newConfiguredAccount(srv.URL)))
	if err != nil {
		t.Fatalf("NewWechatBot: %v", err)
	}

	if err := bot.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if got := atomic.LoadInt32(starts); got != 1 {
		t.Fatalf("notifyStart hits = %d, want 1", got)
	}
	if got := atomic.LoadInt32(stops); got != 0 {
		t.Fatalf("notifyStop hits = %d, want 0", got)
	}

	if err := bot.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if got := atomic.LoadInt32(stops); got != 1 {
		t.Fatalf("notifyStop hits = %d, want 1", got)
	}
}

func TestLifecycleNotifyDisabled(t *testing.T) {
	srv, starts, stops := newLifecycleServer(t, http.StatusOK)

	bot, err := NewWechatBot(
		WithAccount(newConfiguredAccount(srv.URL)),
		WithLifecycleNotify(false),
	)
	if err != nil {
		t.Fatalf("NewWechatBot: %v", err)
	}

	if err := bot.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if err := bot.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	if got := atomic.LoadInt32(starts); got != 0 {
		t.Fatalf("notifyStart hits = %d, want 0", got)
	}
	if got := atomic.LoadInt32(stops); got != 0 {
		t.Fatalf("notifyStop hits = %d, want 0", got)
	}
}

func TestLifecycleTolerantOfServerErrors(t *testing.T) {
	srv, starts, stops := newLifecycleServer(t, http.StatusInternalServerError)

	bot, err := NewWechatBot(WithAccount(newConfiguredAccount(srv.URL)))
	if err != nil {
		t.Fatalf("NewWechatBot: %v", err)
	}

	if err := bot.Connect(context.Background()); err != nil {
		t.Fatalf("Connect should swallow notifyStart errors, got: %v", err)
	}
	if err := bot.Disconnect(); err != nil {
		t.Fatalf("Disconnect should swallow notifyStop errors, got: %v", err)
	}

	if got := atomic.LoadInt32(starts); got != 1 {
		t.Fatalf("notifyStart hits = %d, want 1", got)
	}
	if got := atomic.LoadInt32(stops); got != 1 {
		t.Fatalf("notifyStop hits = %d, want 1", got)
	}
}

func TestDisconnectUsesDetachedContext(t *testing.T) {
	srv, _, stops := newLifecycleServer(t, http.StatusOK)

	bot, err := NewWechatBot(WithAccount(newConfiguredAccount(srv.URL)))
	if err != nil {
		t.Fatalf("NewWechatBot: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := bot.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Cancel before Disconnect: NotifyStop must still reach the server
	// because Disconnect uses a detached context internally.
	cancel()
	time.Sleep(10 * time.Millisecond)

	if err := bot.Disconnect(); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if got := atomic.LoadInt32(stops); got != 1 {
		t.Fatalf("notifyStop hits = %d, want 1 (detached ctx should survive parent cancel)", got)
	}
}
