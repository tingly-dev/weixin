package wecom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tingly-dev/weixin/channel"
)

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

	handler channel.EventHandler

	// Ack tracking: req_id -> channel to signal ack received
	ackChans   map[string]chan struct{}
	ackChansMu sync.Mutex

	// Reply serialization: per-req_id send channel
	// Messages for the same req_id are sent sequentially.
	replyChans   map[string]chan *sendOp
	replyChansMu sync.Mutex

	// Lifecycle
	cancel    context.CancelFunc
	done      chan struct{}
	connected bool
}

// NewClient creates a new WeCom AI Bot client.
func NewClient(cfg ClientConfig) *Client {
	cfg.applyDefaults()
	return &Client{
		cfg:        cfg,
		ackChans:   make(map[string]chan struct{}),
		replyChans: make(map[string]chan *sendOp),
	}
}

type sendOp struct {
	frame *WsFrame
	done  chan error // closed when send completes (or errors)
}

// Connect opens the WebSocket, authenticates, and starts the read loop.
// It blocks until the connection is established and authenticated, or returns an error.
// Use SetEventHandler before calling Connect to receive messages.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	c.mu.Unlock()

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
	ctx, cancel := context.WithCancel(context.Background())
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
	if err := c.authenticate(ctx); err != nil {
		c.Disconnect()
		return err
	}

	// Start background goroutines
	go c.readLoop(ctx)
	go c.heartbeatLoop(ctx)

	return nil
}

// Disconnect gracefully closes the WebSocket connection.
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.ackChansMu.Lock()
	c.ackChans = make(map[string]chan struct{})
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
func (c *Client) SetEventHandler(h channel.EventHandler) {
	c.handler = h
}

// SendReply sends a reply frame to an incoming message.
// reqID must be the req_id from the incoming callback frame.
// It blocks until the server acknowledges or the ack timeout expires.
func (c *Client) SendReply(ctx context.Context, reqID string, body interface{}) error {
	frame := &WsFrame{
		Cmd:     CmdResponse,
		Headers: WsFrameHeaders{ReqID: reqID},
		Body:    body,
	}
	return c.sendAndWaitAck(ctx, frame, reqID)
}

// SendWelcome sends a welcome message. Must be called within 5s of enter_chat event.
func (c *Client) SendWelcome(ctx context.Context, reqID string, body interface{}) error {
	frame := &WsFrame{
		Cmd:     CmdResponseWelcome,
		Headers: WsFrameHeaders{ReqID: reqID},
		Body:    body,
	}
	return c.sendAndWaitAck(ctx, frame, reqID)
}

// SendUpdateCard updates a template card. Must be called within 5s of card event.
func (c *Client) SendUpdateCard(ctx context.Context, reqID string, body interface{}) error {
	frame := &WsFrame{
		Cmd:     CmdResponseUpdate,
		Headers: WsFrameHeaders{ReqID: reqID},
		Body:    body,
	}
	return c.sendAndWaitAck(ctx, frame, reqID)
}

// SendProactive sends a proactive message without an incoming callback.
func (c *Client) SendProactive(ctx context.Context, body interface{}) error {
	reqID := generateReqID(CmdSendMsg)
	frame := &WsFrame{
		Cmd:     CmdSendMsg,
		Headers: WsFrameHeaders{ReqID: reqID},
		Body:    body,
	}
	return c.sendAndWaitAck(ctx, frame, reqID)
}

// SendRaw sends a raw frame (used by upload flow).
func (c *Client) SendRaw(ctx context.Context, frame *WsFrame) error {
	return c.sendAndWaitAck(ctx, frame, frame.Headers.ReqID)
}

// ---------------------------------------------------------------------------
// Internal: authentication
// ---------------------------------------------------------------------------

func (c *Client) authenticate(ctx context.Context) error {
	frame := &WsFrame{
		Cmd:     CmdSubscribe,
		Headers: WsFrameHeaders{ReqID: generateReqID(CmdSubscribe)},
		Body: map[string]interface{}{
			"bot_id": c.cfg.BotID,
			"secret": c.cfg.Secret,
		},
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
		return fmt.Errorf("auth failed: %s (errcode=%d)", ack.ErrMsg, ack.ErrCode)
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

	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	// Notify handler
	if c.handler != nil {
		c.handler.OnEvent(ctx, &channel.Event{
			EventType: "disconnected",
			Timestamp: time.Now(),
			Payload:   map[string]interface{}{"reason": err.Error()},
		})
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
		// Ack frame (no cmd) — signal the waiting sender
		c.signalAck(frame.Headers.ReqID)
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
		c.handler.OnEvent(ctx, &channel.Event{
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

			ackCh := make(chan struct{})
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
				c.Disconnect()
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Internal: send with ack
// ---------------------------------------------------------------------------

func (c *Client) sendAndWaitAck(ctx context.Context, frame *WsFrame, reqID string) error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return fmt.Errorf("not connected")
	}
	c.mu.Unlock()

	// Register ack channel
	ackCh := make(chan struct{})
	c.ackChansMu.Lock()
	c.ackChans[reqID] = ackCh
	c.ackChansMu.Unlock()

	defer func() {
		c.ackChansMu.Lock()
		delete(c.ackChans, reqID)
		c.ackChansMu.Unlock()
	}()

	if err := c.writeFrame(frame); err != nil {
		return err
	}

	// Wait for ack or timeout
	select {
	case <-ackCh:
		return nil
	case <-time.After(c.cfg.ReplyAckTimeout):
		return fmt.Errorf("reply ack timeout")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) signalAck(reqID string) {
	c.ackChansMu.Lock()
	ch, ok := c.ackChans[reqID]
	delete(c.ackChans, reqID)
	c.ackChansMu.Unlock()

	if ok {
		close(ch)
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

// ---------------------------------------------------------------------------
// Internal: io.Reader for readFrame
// ---------------------------------------------------------------------------

// wsReader wraps a websocket.Conn as an io.Reader for frame reading.
type wsReader struct {
	conn *websocket.Conn
}

func (r *wsReader) Read(p []byte) (n int, err error) {
	_, msg, err := r.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	copy(p, msg)
	return len(msg), io.EOF
}
