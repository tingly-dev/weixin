package wecom

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// newTestWsServer starts an httptest server upgrading every request to a
// WebSocket, and returns the ws:// URL to dial plus a cleanup func.
// connHandler is invoked (in its own goroutine) once per accepted connection.
func newTestWsServer(t *testing.T, connHandler func(t *testing.T, conn *websocket.Conn, connNum int)) (wsURL string, cleanup func()) {
	t.Helper()
	upgrader := websocket.Upgrader{}
	var connCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		n := int(atomic.AddInt32(&connCount, 1))
		connHandler(t, conn, n)
	}))

	wsURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	return wsURL, srv.Close
}

// readFrame reads and decodes a single WsFrame from conn. A connection close
// (expected once the client under test disconnects, which can happen from
// t.Cleanup after the test function itself has already returned) is treated
// as "no more frames" rather than a test failure — calling t.Errorf from a
// server goroutine that outlives the test would panic ("Log in goroutine
// after Test has completed").
func readFrame(t *testing.T, conn *websocket.Conn) *WsFrame {
	t.Helper()
	var frame WsFrame
	if err := conn.ReadJSON(&frame); err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) ||
			strings.Contains(err.Error(), "use of closed network connection") {
			return nil
		}
		t.Errorf("server: read frame: %v", err)
		return nil
	}
	return &frame
}

func writeAck(t *testing.T, conn *websocket.Conn, reqID string, errcode int, errmsg string, body interface{}) {
	t.Helper()
	ack := WsFrame{
		Headers: WsFrameHeaders{ReqID: reqID},
		ErrCode: errcode,
		ErrMsg:  errmsg,
		Body:    body,
	}
	if err := conn.WriteJSON(ack); err != nil {
		t.Errorf("server: write ack: %v", err)
	}
}

// acceptAuth reads the subscribe frame and replies with a success ack.
func acceptAuth(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	authFrame := readFrame(t, conn)
	if authFrame == nil {
		return
	}
	if authFrame.Cmd != CmdSubscribe {
		t.Errorf("expected auth frame cmd %q, got %q", CmdSubscribe, authFrame.Cmd)
	}
	writeAck(t, conn, authFrame.Headers.ReqID, 0, "ok", nil)
}

func newConnectedTestClient(t *testing.T, wsURL string) *Client {
	t.Helper()
	client := NewClient(ClientConfig{
		BotID:           "bot-1",
		Secret:          "secret-1",
		WsURL:           wsURL,
		ReplyAckTimeout: 2 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(client.Disconnect)
	return client
}

func TestClient_SendRaw_ReturnsRealAckBody(t *testing.T) {
	wsURL, cleanup := newTestWsServer(t, func(t *testing.T, conn *websocket.Conn, connNum int) {
		acceptAuth(t, conn)
		frame := readFrame(t, conn)
		if frame == nil {
			return
		}
		// Mirrors a real aibot_upload_media_init ack carrying the server's
		// own upload_id — this is exactly the body our client must surface
		// to the caller instead of fabricating one locally.
		writeAck(t, conn, frame.Headers.ReqID, 0, "", map[string]interface{}{
			"upload_id": "server-issued-upload-id",
		})
	})
	defer cleanup()

	client := newConnectedTestClient(t, wsURL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ack, err := client.SendRaw(ctx, &WsFrame{
		Cmd:     CmdUploadMediaInit,
		Headers: WsFrameHeaders{ReqID: generateReqID(CmdUploadMediaInit)},
		Body:    UploadInitBody{Type: "file", Filename: "a.txt", TotalSize: 1, TotalChunks: 1},
	})
	if err != nil {
		t.Fatalf("SendRaw: %v", err)
	}

	var result UploadInitResult
	if err := parseFrameBody(ack.Body, &result); err != nil {
		t.Fatalf("parse ack body: %v", err)
	}
	if result.UploadID != "server-issued-upload-id" {
		t.Fatalf("UploadID = %q, want the server-issued id", result.UploadID)
	}
}

func TestClient_SendReply_NonZeroErrCode_ReturnsError(t *testing.T) {
	wsURL, cleanup := newTestWsServer(t, func(t *testing.T, conn *websocket.Conn, connNum int) {
		acceptAuth(t, conn)
		frame := readFrame(t, conn)
		if frame == nil {
			return
		}
		writeAck(t, conn, frame.Headers.ReqID, 1, "rate limited", nil)
	})
	defer cleanup()

	client := newConnectedTestClient(t, wsURL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := client.SendReply(ctx, "req-123", map[string]string{"msgtype": "text"})
	if err == nil {
		t.Fatal("expected error for non-zero errcode ack, got nil")
	}
	if !strings.Contains(err.Error(), "errcode=1") || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("error %q missing errcode/errmsg detail", err.Error())
	}
}

func TestClient_ReconnectsAfterConnectionDrop(t *testing.T) {
	authCount := int32(0)
	wsURL, cleanup := newTestWsServer(t, func(t *testing.T, conn *websocket.Conn, connNum int) {
		acceptAuth(t, conn)
		atomic.AddInt32(&authCount, 1)
		if connNum == 1 {
			// Simulate an unexpected drop right after auth.
			conn.Close()
			return
		}
		// Second connection (the reconnect): stay up so the client can be
		// observed as connected again.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer cleanup()

	client := NewClient(ClientConfig{
		BotID:                "bot-1",
		Secret:               "secret-1",
		WsURL:                wsURL,
		ReplyAckTimeout:      2 * time.Second,
		ReconnectBaseDelay:   10 * time.Millisecond,
		ReconnectMaxDelay:    50 * time.Millisecond,
		MaxReconnectAttempts: 3,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("initial Connect: %v", err)
	}
	defer client.Disconnect()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&authCount) >= 2 && client.IsConnected() {
			return // success: reconnected and re-authenticated
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("client did not reconnect: authCount=%d connected=%v", atomic.LoadInt32(&authCount), client.IsConnected())
}

func TestClient_Disconnect_StopsReconnecting(t *testing.T) {
	authCount := int32(0)
	wsURL, cleanup := newTestWsServer(t, func(t *testing.T, conn *websocket.Conn, connNum int) {
		acceptAuth(t, conn)
		atomic.AddInt32(&authCount, 1)
		conn.Close() // always drop immediately
	})
	defer cleanup()

	client := NewClient(ClientConfig{
		BotID:                "bot-1",
		Secret:               "secret-1",
		WsURL:                wsURL,
		ReplyAckTimeout:      2 * time.Second,
		ReconnectBaseDelay:   10 * time.Millisecond,
		ReconnectMaxDelay:    20 * time.Millisecond,
		MaxReconnectAttempts: -1, // would retry forever if not for Disconnect
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("initial Connect: %v", err)
	}

	// Let a couple of reconnects happen, then explicitly disconnect.
	time.Sleep(80 * time.Millisecond)
	client.Disconnect()

	// A reconnect attempt already past its backoff wait and mid-dial when
	// Disconnect() fires can still complete one more auth round-trip before
	// its socket gets torn down -- that's an inherent, harmless race (the
	// socket is closed either way, and manualClose stops anything further),
	// not a bug. So allow one in-flight attempt to land before asserting the
	// count has genuinely stopped growing.
	time.Sleep(50 * time.Millisecond)
	settled := atomic.LoadInt32(&authCount)
	time.Sleep(200 * time.Millisecond)
	if got := atomic.LoadInt32(&authCount); got != settled {
		t.Fatalf("auth count kept growing after Disconnect settled (%d -> %d); reconnect loop was not stopped", settled, got)
	}
}
