// Package api provides WeChat API implementations.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// SendMessage sends a message to weixin.
func (c *Client) SendMessage(ctx context.Context, toUserID, contextToken string, items []MessageItem) error {
	// contextToken is optional for block-streaming: the first chunk may lack it,
	// and subsequent chunks receive a reply context_token from the server.
	if contextToken == "" {
		log.Printf("[weixin] contextToken missing for message to %s, sending without context", toUserID)
	}
	req := &SendMessageRequest{
		Msg: &WeixinMessageWrapper{
			FromUserID:   "", // Bot ID is handled by server
			ToUserID:     toUserID,
			ClientID:     generateClientID(),
			MessageType:  MessageTypeBot,
			MessageState: MessageStateFinish,
			ContextToken: contextToken,
			ItemList:     items,
		},
		BaseInfo: &BaseInfo{
			ChannelVersion: SDKVersion,
		},
	}

	return c.doRequest(ctx, "ilink/bot/sendmessage", req, nil)
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
