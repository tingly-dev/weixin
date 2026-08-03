// Package api provides WeChat API implementations.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
)

// generateClientID generates a unique client ID.
func generateClientID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "openclaw-weixin-" + hex.EncodeToString(b)[:16]
	}
	return "openclaw-weixin-" + hex.EncodeToString(b)[:16]
}

// buildSendWrapper builds the WeixinMessageWrapper for an outbound message.
func buildSendWrapper(toUserID, contextToken string, items []MessageItem) *WeixinMessageWrapper {
	// contextToken is optional for block-streaming: the first chunk may lack it,
	// and subsequent chunks receive a reply context_token from the server.
	if contextToken == "" {
		log.Printf("[weixin] contextToken missing for message to %s, sending without context", toUserID)
	}
	return &WeixinMessageWrapper{
		FromUserID:   "", // Bot ID is handled by server
		ToUserID:     toUserID,
		ClientID:     generateClientID(),
		MessageType:  MessageTypeBot,
		MessageState: MessageStateFinish,
		ContextToken: contextToken,
		ItemList:     items,
	}
}

// SendMessage sends a message to weixin.
//
// Since openclaw-weixin v2.4.5 the response is parsed and a non-zero ret is
// returned as an error instead of fire-and-forget, so callers can detect
// delivery failures.
func (c *Client) SendMessage(ctx context.Context, toUserID, contextToken string, items []MessageItem) error {
	req := &SendMessageRequest{
		Msg:      buildSendWrapper(toUserID, contextToken, items),
		BaseInfo: c.BuildBaseInfo(),
	}

	var resp SendMessageResponse
	if err := c.doRequest(ctx, "ilink/bot/sendmessage", req, &resp); err != nil {
		return err
	}
	if resp.Ret != 0 {
		errmsg := resp.ErrMsg
		if errmsg == "" {
			errmsg = "(none)"
		}
		return fmt.Errorf("sendMessage failed: ret=%d errmsg=%s", resp.Ret, errmsg)
	}
	return nil
}

// SendTextMessage sends a text message.
func (c *Client) SendTextMessage(ctx context.Context, toUserID, contextToken, text string) error {
	items := []MessageItem{
		{
			Type: MessageItemTypeText,
			TextItem: &TextItem{
				Text: text,
			},
		},
	}
	return c.SendMessage(ctx, toUserID, contextToken, items)
}
