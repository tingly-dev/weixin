// Package api provides WeChat API implementations.
package api

import (
	"context"
	"github.com/tingly-dev/weixin"
)

// SendMessage sends a message to weixin.
func (c *Client) SendMessage(ctx context.Context, toUserID, contextToken string, items []weixin.MessageItem) error {
	req := &SendMessageRequest{
		Msg: &WeixinMessageWrapper{
			ToUserID:     toUserID,
			ContextToken: contextToken,
			ItemList:     items,
		},
		BaseInfo: &BaseInfo{
			ChannelVersion: "1.0.0",
		},
	}

	return c.doRequest(ctx, "ilink/bot/sendmessage", req, nil)
}

// SendTextMessage sends a text message.
func (c *Client) SendTextMessage(ctx context.Context, toUserID, contextToken, text string) error {
	items := []weixin.MessageItem{
		{
			Type: weixin.MessageItemTypeText,
			TextItem: &weixin.TextItem{
				Text: text,
			},
		},
	}
	return c.SendMessage(ctx, toUserID, contextToken, items)
}
