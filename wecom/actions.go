package wecom

import (
	"context"
	"fmt"

	"github.com/tingly-dev/weixin/types"
	"github.com/tingly-dev/weixin/wechat"
)

// ActionsAdapter implements ActionsAdapter for WeCom AI Bot.
type ActionsAdapter struct {
	gateway *GatewayAdapter
}

// NewActionsAdapter creates a new WeCom actions adapter.
func NewActionsAdapter(gateway *GatewayAdapter) *ActionsAdapter {
	return &ActionsAdapter{gateway: gateway}
}

// Send sends a text message.
// If ContextToken is set, it sends as a reply (passive).
// If ContextToken is empty, it sends as a proactive message.
func (a *ActionsAdapter) Send(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	client := a.gateway.GetClient(msg.AccountID)
	if client == nil || !client.IsConnected() {
		return nil, fmt.Errorf("account %s not connected", msg.AccountID)
	}

	// Check for template card in metadata
	if card, ok := msg.Metadata["wecom_card"]; ok {
		return a.sendCard(ctx, client, msg, card)
	}

	if msg.ContextToken != "" {
		return a.sendReply(ctx, client, msg)
	}
	return a.sendProactive(ctx, client, msg)
}

// SendStream sends a streaming text chunk.
func (a *ActionsAdapter) SendStream(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	client := a.gateway.GetClient(msg.AccountID)
	if client == nil || !client.IsConnected() {
		return nil, fmt.Errorf("account %s not connected", msg.AccountID)
	}

	if msg.ContextToken == "" {
		return nil, fmt.Errorf("streaming requires ContextToken (req_id) from incoming message")
	}

	// Check for stream+card combination
	if card, ok := msg.Metadata["wecom_card"]; ok {
		body := map[string]interface{}{
			"msgtype":       MsgTypeStreamWithCard,
			"stream":        buildStreamBody(msg),
			"template_card": card,
		}
		if err := client.SendReply(ctx, msg.ContextToken, body); err != nil {
			return &types.OutboundResult{OK: false, Error: err.Error()}, err
		}
		return &types.OutboundResult{OK: true}, nil
	}

	body := map[string]interface{}{
		"msgtype": MsgTypeStream,
		"stream":  buildStreamBody(msg),
	}
	if err := client.SendReply(ctx, msg.ContextToken, body); err != nil {
		return &types.OutboundResult{OK: false, Error: err.Error()}, err
	}
	return &types.OutboundResult{OK: true, ChannelMessageID: msg.StreamID}, nil
}

// SendMedia sends a media message.
func (a *ActionsAdapter) SendMedia(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	client := a.gateway.GetClient(msg.AccountID)
	if client == nil || !client.IsConnected() {
		return nil, fmt.Errorf("account %s not connected", msg.AccountID)
	}

	mediaID, ok := msg.Metadata["wecom_media_id"].(string)
	if !ok || mediaID == "" {
		return nil, fmt.Errorf("media_id required in Metadata[\"wecom_media_id\"]")
	}

	mediaType := detectMediaType(msg.ContentType)

	if msg.ContextToken != "" {
		body := buildMediaBody(mediaType, mediaID, msg)
		if err := client.SendReply(ctx, msg.ContextToken, body); err != nil {
			return &types.OutboundResult{OK: false, Error: err.Error()}, err
		}
	} else {
		body := map[string]interface{}{
			"chatid":  msg.To,
			"msgtype": mediaType,
		}
		addMediaToBody(body, mediaType, mediaID, msg)
		if err := client.SendProactive(ctx, body); err != nil {
			return &types.OutboundResult{OK: false, Error: err.Error()}, err
		}
	}

	return &types.OutboundResult{OK: true}, nil
}

// React is not supported by WeCom AI Bot.
func (a *ActionsAdapter) React(ctx context.Context, reaction *types.Reaction) (*types.OutboundResult, error) {
	return nil, &wechat.Error{
		Type:    wechat.ErrorNotSupported,
		Message: "reactions not supported by WeCom AI Bot",
	}
}

// Edit is not supported by WeCom AI Bot.
func (a *ActionsAdapter) Edit(ctx context.Context, messageID string, text string) (*types.OutboundResult, error) {
	return nil, &wechat.Error{
		Type:    wechat.ErrorNotSupported,
		Message: "message editing not supported by WeCom AI Bot",
	}
}

// Unsend is not supported by WeCom AI Bot.
func (a *ActionsAdapter) Unsend(ctx context.Context, messageID string) (*types.OutboundResult, error) {
	return nil, &wechat.Error{
		Type:    wechat.ErrorNotSupported,
		Message: "message deletion not supported by WeCom AI Bot",
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (a *ActionsAdapter) sendReply(ctx context.Context, client *Client, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	body := map[string]interface{}{
		"msgtype": MsgTypeStream,
		"stream": map[string]interface{}{
			"id":      generateReqID("stream"),
			"finish":  true,
			"content": msg.Text,
		},
	}
	if err := client.SendReply(ctx, msg.ContextToken, body); err != nil {
		return &types.OutboundResult{OK: false, Error: err.Error()}, err
	}
	return &types.OutboundResult{OK: true}, nil
}

func (a *ActionsAdapter) sendProactive(ctx context.Context, client *Client, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	body := map[string]interface{}{
		"chatid":  msg.To,
		"msgtype": MsgTypeMarkdown,
		"markdown": map[string]interface{}{
			"content": msg.Text,
		},
	}
	if err := client.SendProactive(ctx, body); err != nil {
		return &types.OutboundResult{OK: false, Error: err.Error()}, err
	}
	return &types.OutboundResult{OK: true}, nil
}

func (a *ActionsAdapter) sendCard(ctx context.Context, client *Client, msg *types.OutboundMessage, card interface{}) (*types.OutboundResult, error) {
	if msg.ContextToken != "" {
		body := map[string]interface{}{
			"msgtype":       MsgTypeTemplateCard,
			"template_card": card,
		}
		if err := client.SendReply(ctx, msg.ContextToken, body); err != nil {
			return &types.OutboundResult{OK: false, Error: err.Error()}, err
		}
	} else {
		body := map[string]interface{}{
			"chatid":        msg.To,
			"msgtype":       MsgTypeTemplateCard,
			"template_card": card,
		}
		if err := client.SendProactive(ctx, body); err != nil {
			return &types.OutboundResult{OK: false, Error: err.Error()}, err
		}
	}
	return &types.OutboundResult{OK: true}, nil
}

func buildStreamBody(msg *types.OutboundMessage) map[string]interface{} {
	body := map[string]interface{}{
		"id":      msg.StreamID,
		"finish":  msg.StreamFinish,
		"content": msg.Text,
	}
	if msg.StreamID == "" {
		body["id"] = generateReqID("stream")
	}
	return body
}

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
