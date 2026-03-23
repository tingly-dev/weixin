// Package adapters provides adapter implementations for the WeChat channel.
package adapters

import (
	"context"
	"fmt"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/channel"
	"github.com/tingly-dev/weixin/message"
)

// ActionsAdapter handles message actions for weixin.
type ActionsAdapter struct {
	plugin weixin.PluginInterface
}

// NewActionsAdapter creates a new actions adapter.
func NewActionsAdapter(plugin weixin.PluginInterface) *ActionsAdapter {
	return &ActionsAdapter{plugin: plugin}
}

// Send sends a text message to weixin.
func (a *ActionsAdapter) Send(ctx context.Context, msg *channel.OutboundMessage) (*channel.OutboundResult, error) {
	// Get account
	account, err := a.plugin.Accounts().Get(msg.AccountID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	if !account.Enabled || !account.Configured {
		return nil, &channel.ChannelError{
			Type:    channel.ErrorAccountNotFound,
			Message: "account not enabled or configured",
			Channel: channel.ChannelIDWeChat,
		}
	}

	// Create API client
	client := api.NewClient(account.BaseURL, account.BotToken)

	// Convert message
	toUserID := msg.To
	contextToken := msg.ContextToken
	items := message.ConvertOutboundMessageToList(msg)

	// Send message
	if err := client.SendMessage(ctx, toUserID, contextToken, items); err != nil {
		return &channel.OutboundResult{
			OK:    false,
			Error: err.Error(),
		}, err
	}

	return &channel.OutboundResult{
		OK:        true,
		MessageID: "wc-" + account.ID,
	}, nil
}

// SendMedia sends a media message to weixin.
func (a *ActionsAdapter) SendMedia(ctx context.Context, msg *channel.OutboundMessage) (*channel.OutboundResult, error) {
	// For WeChat, media and text messages use the same endpoint
	// The media upload should happen before calling this, and the
	// MediaURL should contain the CDN reference
	return a.Send(ctx, msg)
}

// React is not supported by weixin.
func (a *ActionsAdapter) React(ctx context.Context, reaction *channel.Reaction) (*channel.OutboundResult, error) {
	return nil, &channel.ChannelError{
		Type:    channel.ErrorNotSupported,
		Message: "reactions not supported by WeChat",
		Channel: channel.ChannelIDWeChat,
	}
}

// Edit is not supported by weixin.
func (a *ActionsAdapter) Edit(ctx context.Context, messageID string, text string) (*channel.OutboundResult, error) {
	return nil, &channel.ChannelError{
		Type:    channel.ErrorNotSupported,
		Message: "message editing not supported by WeChat",
		Channel: channel.ChannelIDWeChat,
	}
}

// Unsend is not supported by weixin.
func (a *ActionsAdapter) Unsend(ctx context.Context, messageID string) (*channel.OutboundResult, error) {
	return nil, &channel.ChannelError{
		Type:    channel.ErrorNotSupported,
		Message: "message deletion not supported by WeChat",
		Channel: channel.ChannelIDWeChat,
	}
}
