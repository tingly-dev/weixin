// Package wecom provides the WeCom AI Bot implementation.
package wecom

import (
	"context"
	"log"

	"github.com/tingly-dev/weixin/bot"
	"github.com/tingly-dev/weixin/types"
)

// WecomConfig holds WeCom-specific configuration.
type WecomConfig struct {
	// Logger for logging output (nil for silent)
	Logger *log.Logger
}

// WecomBot is the WeCom AI Bot.
// One bot manages one WebSocket client connection.
type WecomBot struct {
	*bot.BaseBot
	config       *WecomConfig
	client       *Client // WebSocket client
	running      bool
	eventHandler types.EventHandler // Stored event handler to apply after Connect
}

// NewWecomBot creates a new WeCom bot.
func NewWecomBot(config *WecomConfig) *WecomBot {
	if config == nil {
		config = &WecomConfig{}
	}

	b := &WecomBot{
		config: config,
	}

	// Create base bot with metadata
	meta := &types.Meta{
		Label:          "WeCom",
		SelectionLabel: "WeCom (Enterprise WeChat)",
		DetailLabel:    "WeCom AI Bot",
		Blurb:          "WeCom Enterprise WeChat AI Bot integration",
		DocsPath:       "/docs/wecom",
		SystemImage:    "building.2.fill",
		Version:        "1.0.0",
	}

	capabilities := &types.Capabilities{
		ChatTypes: []types.ChatType{types.ChatTypeDirect, types.ChatTypeGroup},
		Text:      true,
		Media:     true,
		Streaming: true,
	}

	b.BaseBot = bot.NewBaseBot(meta, capabilities)

	return b
}

// Config returns the bot configuration.
func (b *WecomBot) Config() *WecomConfig {
	return b.config
}

// Client returns the WebSocket client.
func (b *WecomBot) Client() *Client {
	return b.client
}

// SetEventHandler sets the event handler for incoming messages.
// If the client is already connected, the handler is set immediately.
// Otherwise, it is stored and applied when Connect() is called.
func (b *WecomBot) SetEventHandler(handler types.EventHandler) {
	b.eventHandler = handler
	if b.client != nil {
		b.client.SetEventHandler(handler)
	}
}

// Connect connects the WebSocket to WeCom.
func (b *WecomBot) Connect(ctx context.Context, botID, secret string) error {
	if b.client != nil && b.client.IsConnected() {
		return nil // Already connected
	}

	// Create client
	b.client = NewClient(ClientConfig{
		BotID:  botID,
		Secret: secret,
		Logger: b.config.Logger,
	})

	// Apply stored event handler if any was registered
	if b.eventHandler != nil {
		b.client.SetEventHandler(b.eventHandler)
	}

	// Connect
	if err := b.client.Connect(ctx); err != nil {
		return err
	}

	b.running = true
	return nil
}

// Disconnect disconnects the WebSocket.
func (b *WecomBot) Disconnect() error {
	if b.client != nil {
		b.client.Disconnect()
	}
	b.running = false
	b.client = nil // Clear client to allow reconnection
	return nil
}

// IsConnected returns whether the bot is connected.
func (b *WecomBot) IsConnected() bool {
	return b.client != nil && b.client.IsConnected()
}

// Send sends a message.
func (b *WecomBot) Send(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	if b.client == nil || !b.client.IsConnected() {
		return nil, &types.Error{
			Type:    types.ErrorAccountNotFound,
			Message: "bot not connected",
		}
	}

	// Use the client's send methods
	if msg.ContextToken != "" {
		return b.sendReply(ctx, msg)
	}
	return b.sendProactive(ctx, msg)
}

func (b *WecomBot) sendReply(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	body := map[string]interface{}{
		"msgtype": MsgTypeStream,
		"stream": map[string]interface{}{
			"id":      generateReqID("stream"),
			"finish":  true,
			"content": msg.Text,
		},
	}
	if err := b.client.SendReply(ctx, msg.ContextToken, body); err != nil {
		return &types.OutboundResult{OK: false, Error: err.Error()}, err
	}
	return &types.OutboundResult{OK: true}, nil
}

func (b *WecomBot) sendProactive(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	body := map[string]interface{}{
		"chatid":  msg.To,
		"msgtype": MsgTypeMarkdown,
		"markdown": map[string]interface{}{
			"content": msg.Text,
		},
	}
	if err := b.client.SendProactive(ctx, body); err != nil {
		return &types.OutboundResult{OK: false, Error: err.Error()}, err
	}
	return &types.OutboundResult{OK: true}, nil
}

// SendStream sends a streaming text chunk.
func (b *WecomBot) SendStream(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	if b.client == nil || !b.client.IsConnected() {
		return nil, &types.Error{
			Type:    types.ErrorAccountNotFound,
			Message: "bot not connected",
		}
	}

	if msg.ContextToken == "" {
		return nil, &types.Error{
			Type:    types.ErrorNotSupported,
			Message: "streaming requires ContextToken (req_id) from incoming message",
		}
	}

	body := map[string]interface{}{
		"msgtype": MsgTypeStream,
		"stream": map[string]interface{}{
			"id":      msg.StreamID,
			"finish":  msg.StreamFinish,
			"content": msg.Text,
		},
	}
	if msg.StreamID == "" {
		body["stream"].(map[string]interface{})["id"] = generateReqID("stream")
	}

	if err := b.client.SendReply(ctx, msg.ContextToken, body); err != nil {
		return &types.OutboundResult{OK: false, Error: err.Error()}, err
	}
	return &types.OutboundResult{OK: true, ChannelMessageID: msg.StreamID}, nil
}

// SendMedia sends a media message.
func (b *WecomBot) SendMedia(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	if b.client == nil || !b.client.IsConnected() {
		return nil, &types.Error{
			Type:    types.ErrorAccountNotFound,
			Message: "bot not connected",
		}
	}

	mediaID, ok := msg.Metadata["wecom_media_id"].(string)
	if !ok || mediaID == "" {
		return nil, &types.Error{
			Type:    types.ErrorNotSupported,
			Message: "media_id required in Metadata[\"wecom_media_id\"]",
		}
	}

	mediaType := detectMediaType(msg.ContentType)

	if msg.ContextToken != "" {
		body := buildMediaBody(mediaType, mediaID, msg)
		if err := b.client.SendReply(ctx, msg.ContextToken, body); err != nil {
			return &types.OutboundResult{OK: false, Error: err.Error()}, err
		}
	} else {
		body := map[string]interface{}{
			"chatid":  msg.To,
			"msgtype": mediaType,
		}
		addMediaToBody(body, mediaType, mediaID, msg)
		if err := b.client.SendProactive(ctx, body); err != nil {
			return &types.OutboundResult{OK: false, Error: err.Error()}, err
		}
	}

	return &types.OutboundResult{OK: true}, nil
}

// Helper functions for media sending

func detectMediaType(contentType string) string {
	switch contentType {
	case "image", "image/png", "image/jpeg", "image/gif":
		return MsgTypeImage
	case "video", "video/mp4":
		return MsgTypeVideo
	case "audio", "audio/silk":
		return MsgTypeVoice
	default:
		return MsgTypeFile
	}
}

func buildMediaBody(mediaType, mediaID string, msg *types.OutboundMessage) map[string]interface{} {
	body := map[string]interface{}{
		"msgtype": mediaType,
	}
	addMediaToBody(body, mediaType, mediaID, msg)
	return body
}

func addMediaToBody(body map[string]interface{}, mediaType, mediaID string, msg *types.OutboundMessage) {
	switch mediaType {
	case MsgTypeVideo:
		body[mediaType] = map[string]interface{}{
			"media_id": mediaID,
			"title":    msg.FileName,
		}
	default:
		body[mediaType] = map[string]interface{}{
			"media_id": mediaID,
		}
	}
}
