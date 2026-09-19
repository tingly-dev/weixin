package wecom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tingly-dev/weixin/types"
)

// errAuthFailed marks an error returned by authenticate as an auth rejection
// (non-zero errcode on the subscribe ack), as opposed to a network/dial
// failure. Reconnect scheduling uses this to pick the auth-failure budget
// (MaxAuthFailures) instead of the network-drop budget (MaxReconnectAttempts),
// mirroring the official SDK's two separate reconnect counters.
var errAuthFailed = errors.New("wecom: auth failed")

// ClientConfig holds configuration for the WeCom AI Bot WebSocket client.
type ClientConfig struct {
	BotID                string
	Secret               string
	WsURL                string // default: wss://openws.work.weixin.qq.com
	HeartbeatInterval    time.Duration
	ReconnectBaseDelay   time.Duration
	ReconnectMaxDelay    time.Duration
	MaxReconnectAttempts int // -1 for infinite
	MaxAuthFailures      int // -1 for infinite
	ReplyAckTimeout      time.Duration
	Logger               *log.Logger // nil for silent

	// ExtraAuthParams are merged into the aibot_subscribe auth frame's body
	// alongside bot_id/secret (e.g. {"scene": 1, "plug_version": "1.0.0"}).
	// Mirrors the official SDK's setCredentials(botId, botSecret, extraAuthParams).
	ExtraAuthParams map[string]interface{}
}

func (c *ClientConfig) applyDefaults() {
	if c.WsURL == "" {
		c.WsURL = DefaultWsURL
	}
	if c.HeartbeatInterval == 0 {
		c.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if c.ReconnectBaseDelay == 0 {
		c.ReconnectBaseDelay = DefaultReconnectBaseDelay
	}
	if c.ReconnectMaxDelay == 0 {
		c.ReconnectMaxDelay = DefaultReconnectMaxDelay
	}
	if c.MaxReconnectAttempts == 0 {
		c.MaxReconnectAttempts = DefaultMaxReconnectAttempts
	}
	if c.MaxAuthFailures == 0 {
		c.MaxAuthFailures = DefaultMaxAuthFailures
	}
	if c.ReplyAckTimeout == 0 {
		c.ReplyAckTimeout = DefaultReplyAckTimeout
	}
}

// Client is the main WeCom AI Bot WebSocket client.
// It manages the connection lifecycle, message dispatch, and reply sending.
type Client struct {
	cfg  ClientConfig
	conn *websocket.Conn
	mu   sync.Mutex

	handler types.EventHandler

	// Ack tracking: req_id -> channel receiving the server's ack frame
	// (its Body/ErrCode/ErrMsg), so callers can read real server responses
	// (e.g. upload_id, media_id) instead of only knowing "acked or not".
	ackChans   map[string]chan *WsFrame
	ackChansMu sync.Mutex

	// Reply serialization: per-req_id send channel
	// Messages for the same req_id are sent sequentially.
	replyChans   map[string]chan *sendOp
	replyChansMu sync.Mutex

	// Per-connection lifecycle: cancelled/recreated on every (re)connect,
	// scoping the read/heartbeat loops for the current socket only.
	cancel    context.CancelFunc
	done      chan struct{}
	connected bool

	// Client-level lifecycle: spans reconnects, cancelled only by Disconnect().
	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc
	manualClose     bool

	reconnectAttempts   int
	authFailureAttempts int
}

// NewClient creates a new WeCom AI Bot client.
func NewClient(cfg ClientConfig) *Client {
	cfg.applyDefaults()
	return &Client{
		cfg:        cfg,
		ackChans:   make(map[string]chan *WsFrame),
		replyChans: make(map[string]chan *sendOp),
	}
}

type sendOp struct {
	frame *WsFrame
	done  chan error // closed when send completes (or errors)
}

// Connect opens the WebSocket, authenticates, and starts the read loop.
// It blocks until the connection is established and authenticated, or returns
// an error. Use SetEventHandler before calling Connect to receive messages.
//
// After a successful Connect, an unexpected drop (network error, missed
// heartbeats) is retried automatically with exponential backoff — see
// ClientConfig.ReconnectBaseDelay/ReconnectMaxDelay/MaxReconnectAttempts/
// MaxAuthFailures. Call Disconnect to stop reconnecting and close for good.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	c.mu.Unlock()

	c.lifecycleCtx, c.lifecycleCancel = context.WithCancel(context.Background())
	c.manualClose = false
	c.reconnectAttempts = 0
	c.authFailureAttempts = 0

	if err := c.connectOnce(ctx); err != nil {
		c.lifecycleCancel()
		return err
	}
	return nil
}

// connectOnce dials, authenticates, and starts the read/heartbeat loops for a
// single connection attempt. Used by both Connect and the reconnect loop.
func (c *Client) connectOnce(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, c.cfg.WsURL, http.Header{})
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	loopCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.done = make(chan struct{})
	c.mu.Unlock()

	// Clear any read deadline set during auth, and enable WS-level ping handling.
	// Without this, the deadline from readFrameTimeout persists and kills readLoop.
	c.conn.SetReadDeadline(time.Time{})
	c.conn.SetPongHandler(func(appData string) error {
		return nil
	})

	// Authenticate (writeFrame acquires c.mu internally)
	if err := c.authenticate(loopCtx); err != nil {
		c.teardownConn()
		return err
	}

	// Start background goroutines
	go c.readLoop(loopCtx)
	go c.heartbeatLoop(loopCtx)

	return nil
}

// Disconnect gracefully closes the WebSocket connection and stops any
// pending/future reconnect attempts.
func (c *Client) Disconnect() {
	c.mu.Lock()
	c.manualClose = true
	c.mu.Unlock()

	if c.lifecycleCancel != nil {
		c.lifecycleCancel()
	}

	c.teardownConn()
}

// teardownConn closes the current socket and per-connection loops without
// touching the client-level lifecycle or manualClose flag, so it's safe to
// call both from a deliberate Disconnect and from internal drop handling
// (which needs the socket closed but reconnection to still proceed).
func (c *Client) teardownConn() {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
	}

	if c.conn != nil {
		// Send close frame
		c.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
		c.conn = nil
	}

	c.connected = false
	c.mu.Unlock()

	c.ackChansMu.Lock()
	c.ackChans = make(map[string]chan *WsFrame)
	c.ackChansMu.Unlock()
	c.replyChansMu.Lock()
	c.replyChans = make(map[string]chan *sendOp)
	c.replyChansMu.Unlock()
}

// IsConnected reports whether the client is currently connected.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// SetEventHandler sets the handler for incoming messages and events.
func (c *Client) SetEventHandler(h types.EventHandler) {
	c.handler = h
}

// SendReply sends a reply frame to an incoming message and returns the
// server's ack frame (its Body may carry a response payload depending on cmd).
// reqID must be the req_id from the incoming callback frame.
// It blocks until the server acknowledges or the ack timeout expires.
func (c *Client) SendReply(ctx context.Context, reqID string, body interface{}) (*WsFrame, error) {
	frame := &WsFrame{
		Cmd:     CmdResponse,
		Headers: WsFrameHeaders{ReqID: reqID},
		Body:    body,
	}
	return c.sendAndWaitAck(ctx, frame, reqID)
}

// SendWelcome sends a welcome message. Must be called within 5s of enter_chat event.
func (c *Client) SendWelcome(ctx context.Context, reqID string, body interface{}) (*WsFrame, error) {
	frame := &WsFrame{
		Cmd:     CmdResponseWelcome,
		Headers: WsFrameHeaders{ReqID: reqID},
		Body:    body,
	}
	return c.sendAndWaitAck(ctx, frame, reqID)
}

// SendUpdateCard updates a template card. Must be called within 5s of card event.
func (c *Client) SendUpdateCard(ctx context.Context, reqID string, body interface{}) (*WsFrame, error) {
	frame := &WsFrame{
		Cmd:     CmdResponseUpdate,
		Headers: WsFrameHeaders{ReqID: reqID},
		Body:    body,
	}
	return c.sendAndWaitAck(ctx, frame, reqID)
}

// SendProactive sends a proactive message without an incoming callback.
func (c *Client) SendProactive(ctx context.Context, body interface{}) (*WsFrame, error) {
	reqID := generateReqID(CmdSendMsg)
	frame := &WsFrame{
		Cmd:     CmdSendMsg,
		Headers: WsFrameHeaders{ReqID: reqID},
		Body:    body,
	}
	return c.sendAndWaitAck(ctx, frame, reqID)
}

// SendRaw sends a raw frame (used by upload flow) and returns the server's
// ack frame so callers can read response fields (e.g. upload_id, media_id).
func (c *Client) SendRaw(ctx context.Context, frame *WsFrame) (*WsFrame, error) {
	return c.sendAndWaitAck(ctx, frame, frame.Headers.ReqID)
}

// ---------------------------------------------------------------------------
// Internal: authentication
// ---------------------------------------------------------------------------

func (c *Client) authenticate(ctx context.Context) error {
	body := map[string]interface{}{
		"bot_id": c.cfg.BotID,
		"secret": c.cfg.Secret,
	}
	for k, v := range c.cfg.ExtraAuthParams {
		body[k] = v
	}

	frame := &WsFrame{
		Cmd:     CmdSubscribe,
		Headers: WsFrameHeaders{ReqID: generateReqID(CmdSubscribe)},
		Body:    body,
	}

	if err := c.writeFrame(frame); err != nil {
		return fmt.Errorf("send subscribe: %w", err)
	}

	// Wait for auth response
	ack, err := c.readFrameTimeout(ctx, c.cfg.ReplyAckTimeout)
	if err != nil {
		return fmt.Errorf("read auth response: %w", err)
	}

	if ack.ErrCode != 0 {
		return fmt.Errorf("%w: %s (errcode=%d)", errAuthFailed, ack.ErrMsg, ack.ErrCode)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Internal: read loop
// ---------------------------------------------------------------------------

func (c *Client) readLoop(ctx context.Context) {
	defer close(c.done)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			c.handleReadError(ctx, err)
			return
		}

		var frame WsFrame
		if err := json.Unmarshal(raw, &frame); err != nil {
			c.cfg.log("read frame error: %v", err)
			continue
		}

		c.dispatchFrame(ctx, &frame)
	}
}

func (c *Client) handleReadError(ctx context.Context, err error) {
	if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
		c.cfg.log("connection lost: %v", err)
	} else {
		c.cfg.log("connection closed")
	}

	c.teardownConn()

	// Notify handler
	if c.handler != nil {
		c.handler.OnEvent(ctx, &types.Event{
			EventType: "disconnected",
			Timestamp: time.Now(),
			Payload:   map[string]interface{}{"reason": err.Error()},
		})
	}

	c.mu.Lock()
	manual := c.manualClose
	c.mu.Unlock()
	if !manual {
		go c.scheduleReconnect(false)
	}
}

// ---------------------------------------------------------------------------
// Internal: frame dispatch
// ---------------------------------------------------------------------------

func (c *Client) dispatchFrame(ctx context.Context, frame *WsFrame) {
	switch {
	case frame.Cmd == CmdCallback:
		c.handleCallback(ctx, frame)
	case frame.Cmd == CmdEventCallback:
		c.handleEventCallback(ctx, frame)
	case frame.Cmd == "":
		// Ack frame (no cmd) — signal the waiting sender with the full frame.
		c.signalAck(frame)
	default:
		c.cfg.log("unknown frame cmd: %s", frame.Cmd)
	}
}

func (c *Client) handleCallback(ctx context.Context, frame *WsFrame) {
	var msg IncomingMessage
	if err := parseFrameBody(frame.Body, &msg); err != nil {
		c.cfg.log("parse message callback: %v", err)
		return
	}

	if c.handler != nil {
		chMsg := convertToChannelMessage(&msg, frame.Headers.ReqID)
		if err := c.handler.OnMessage(ctx, chMsg); err != nil {
			c.cfg.log("message handler error: %v", err)
		}
	}
}

func (c *Client) handleEventCallback(ctx context.Context, frame *WsFrame) {
	var evt IncomingEvent
	if err := parseFrameBody(frame.Body, &evt); err != nil {
		c.cfg.log("parse event callback: %v", err)
		return
	}

	// Store req_id in context for reply correlation
	payload := map[string]interface{}{
		"req_id": frame.Headers.ReqID,
	}
	if evt.Event.EventKey != "" {
		payload["event_key"] = evt.Event.EventKey
	}
	if evt.Event.TaskID != "" {
		payload["task_id"] = evt.Event.TaskID
	}

	if c.handler != nil {
		c.handler.OnEvent(ctx, &types.Event{
			EventType: evt.Event.EventType,
			AccountID: "",
			Timestamp: time.Unix(evt.CreateTime, 0),
			Payload:   payload,
		})
	}
}

// ---------------------------------------------------------------------------
// Internal: heartbeat
// ---------------------------------------------------------------------------

func (c *Client) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.HeartbeatInterval)
	defer ticker.Stop()

	missedPongs := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !c.IsConnected() {
				continue
			}

			frame := &WsFrame{
				Cmd:     CmdHeartbeat,
				Headers: WsFrameHeaders{ReqID: generateReqID(CmdHeartbeat)},
			}

			ackCh := make(chan *WsFrame, 1)
			c.ackChansMu.Lock()
			c.ackChans[frame.Headers.ReqID] = ackCh
			c.ackChansMu.Unlock()

			if err := c.writeFrame(frame); err != nil {
				c.cfg.log("heartbeat send error: %v", err)
				missedPongs++
			} else {
				select {
				case <-ackCh:
					missedPongs = 0
				case <-time.After(c.cfg.HeartbeatInterval):
					missedPongs++
				case <-ctx.Done():
					return
				}
			}

			if missedPongs >= 2 {
				c.cfg.log("too many missed heartbeats, closing connection")
				c.teardownConn()
				if c.handler != nil {
					c.handler.OnEvent(ctx, &types.Event{
						EventType: "disconnected",
						Timestamp: time.Now(),
						Payload:   map[string]interface{}{"reason": "missed heartbeats"},
					})
				}
				c.mu.Lock()
				manual := c.manualClose
				c.mu.Unlock()
				if !manual {
					go c.scheduleReconnect(false)
				}
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Internal: reconnect
// ---------------------------------------------------------------------------

// scheduleReconnect waits with exponential backoff, then retries connectOnce.
// It uses separate attempt budgets for auth failures vs. network drops,
// mirroring the official SDK's design (see ws.d.ts's scheduleReconnect doc).
// A manual Disconnect (which cancels lifecycleCtx) aborts any pending or
// future reconnect.
func (c *Client) scheduleReconnect(authFailure bool) {
	c.mu.Lock()
	manual := c.manualClose
	lifecycleCtx := c.lifecycleCtx
	c.mu.Unlock()
	if manual || lifecycleCtx == nil {
		return
	}

	var attempt, maxAttempts int
	c.mu.Lock()
	if authFailure {
		c.authFailureAttempts++
		attempt = c.authFailureAttempts
	} else {
		c.reconnectAttempts++
		attempt = c.reconnectAttempts
	}
	c.mu.Unlock()
	if authFailure {
		maxAttempts = c.cfg.MaxAuthFailures
	} else {
		maxAttempts = c.cfg.MaxReconnectAttempts
	}

	if maxAttempts >= 0 && attempt > maxAttempts {
		c.cfg.log("wecom: giving up reconnecting after %d attempts (authFailure=%v)", attempt-1, authFailure)
		if c.handler != nil {
			c.handler.OnEvent(lifecycleCtx, &types.Event{
				EventType: "reconnect_failed",
				Timestamp: time.Now(),
				Payload:   map[string]interface{}{"attempts": attempt - 1, "auth_failure": authFailure},
			})
		}
		return
	}

	delay := c.cfg.ReconnectBaseDelay * time.Duration(int64(1)<<uint(attempt-1))
	if delay <= 0 || delay > c.cfg.ReconnectMaxDelay {
		delay = c.cfg.ReconnectMaxDelay
	}

	c.cfg.log("wecom: reconnecting in %s (attempt %d, authFailure=%v)", delay, attempt, authFailure)
	if c.handler != nil {
		c.handler.OnEvent(lifecycleCtx, &types.Event{
			EventType: "reconnecting",
			Timestamp: time.Now(),
			Payload:   map[string]interface{}{"attempt": attempt, "delay_ms": delay.Milliseconds()},
		})
	}

	timer := time.NewTimer(delay)
	select {
	case <-lifecycleCtx.Done():
		timer.Stop()
		return
	case <-timer.C:
	}

	if err := c.connectOnce(lifecycleCtx); err != nil {
		c.cfg.log("wecom: reconnect attempt %d failed: %v", attempt, err)
		c.scheduleReconnect(errors.Is(err, errAuthFailed))
		return
	}

	c.mu.Lock()
	c.reconnectAttempts = 0
	c.authFailureAttempts = 0
	c.mu.Unlock()
	if c.handler != nil {
		c.handler.OnEvent(lifecycleCtx, &types.Event{
			EventType: "reconnected",
			Timestamp: time.Now(),
		})
	}
}

// ---------------------------------------------------------------------------
// Internal: send with ack
// ---------------------------------------------------------------------------

// sendAndWaitAck sends frame and waits for the server's ack, returning it.
// A non-zero ack.ErrCode is surfaced as an error (with the ack frame still
// returned, so the caller can inspect ErrMsg/ErrCode themselves if needed).
func (c *Client) sendAndWaitAck(ctx context.Context, frame *WsFrame, reqID string) (*WsFrame, error) {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return nil, fmt.Errorf("not connected")
	}
	c.mu.Unlock()

	// Register ack channel
	ackCh := make(chan *WsFrame, 1)
	c.ackChansMu.Lock()
	c.ackChans[reqID] = ackCh
	c.ackChansMu.Unlock()

	defer func() {
		c.ackChansMu.Lock()
		delete(c.ackChans, reqID)
		c.ackChansMu.Unlock()
	}()

	if err := c.writeFrame(frame); err != nil {
		return nil, err
	}

	// Wait for ack or timeout
	select {
	case ack := <-ackCh:
		if ack.ErrCode != 0 {
			errmsg := ack.ErrMsg
			if errmsg == "" {
				errmsg = "(none)"
			}
			return ack, fmt.Errorf("wecom ack failed: errcode=%d errmsg=%s", ack.ErrCode, errmsg)
		}
		return ack, nil
	case <-time.After(c.cfg.ReplyAckTimeout):
		return nil, fmt.Errorf("reply ack timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) signalAck(frame *WsFrame) {
	reqID := frame.Headers.ReqID
	c.ackChansMu.Lock()
	ch, ok := c.ackChans[reqID]
	delete(c.ackChans, reqID)
	c.ackChansMu.Unlock()

	if ok {
		ch <- frame
	}
}

// ---------------------------------------------------------------------------
// Internal: low-level write
// ---------------------------------------------------------------------------

func (c *Client) writeFrame(frame *WsFrame) error {
	data, err := encodeFrame(frame)
	if err != nil {
		return fmt.Errorf("encode frame: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("connection is nil")
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) readFrameTimeout(ctx context.Context, timeout time.Duration) (*WsFrame, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("connection is nil")
	}

	deadline := time.Now().Add(timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	c.conn.SetReadDeadline(deadline)
	defer c.conn.SetReadDeadline(time.Time{}) // reset after use

	_, raw, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var frame WsFrame
	if err := json.Unmarshal(raw, &frame); err != nil {
		return nil, fmt.Errorf("unmarshal frame: %w", err)
	}

	return &frame, nil
}

// ---------------------------------------------------------------------------
// Internal: logging helper
// ---------------------------------------------------------------------------

func (c *ClientConfig) log(format string, args ...interface{}) {
	if c.Logger != nil {
		c.Logger.Printf(format, args...)
	}
}
