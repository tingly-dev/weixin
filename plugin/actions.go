package plugin

import (
	"context"
	"fmt"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/message"
)

// actionsAdapter handles message actions for weixin.
type actionsAdapter struct {
	plugin *Plugin
}

// newActionsAdapter creates a new actions adapter.
func newActionsAdapter(plugin *Plugin) *actionsAdapter {
	return &actionsAdapter{plugin: plugin}
}

// Send sends a text message to weixin.
func (a *actionsAdapter) Send(ctx context.Context, msg *weixin.OutboundMessage) (*weixin.OutboundResult, error) {
	// Get account
	account, err := a.plugin.Accounts().Get(msg.AccountID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	if !account.Enabled || !account.Configured {
		return nil, &Error{
			Type:    ErrorAccountNotFound,
			Message: "account not enabled or configured",
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
		return &weixin.OutboundResult{
			OK:    false,
			Error: err.Error(),
		}, err
	}

	return &weixin.OutboundResult{
		OK:        true,
		MessageID: "wc-" + account.ID,
	}, nil
}

// SendMedia sends a media message to weixin.
func (a *actionsAdapter) SendMedia(ctx context.Context, msg *weixin.OutboundMessage) (*weixin.OutboundResult, error) {
	// For WeChat, media and text messages use the same endpoint
	return a.Send(ctx, msg)
}

// SendStream is not supported by weixin.
func (a *actionsAdapter) SendStream(ctx context.Context, msg *weixin.OutboundMessage) (*weixin.OutboundResult, error) {
	return nil, &Error{
		Type:    ErrorNotSupported,
		Message: "streaming not supported by WeChat ilink protocol",
	}
}

// React is not supported by weixin.
func (a *actionsAdapter) React(ctx context.Context, reaction *weixin.Reaction) (*weixin.OutboundResult, error) {
	return nil, &Error{
		Type:    ErrorNotSupported,
		Message: "reactions not supported by WeChat",
	}
}

// Edit is not supported by weixin.
func (a *actionsAdapter) Edit(ctx context.Context, messageID string, text string) (*weixin.OutboundResult, error) {
	return nil, &Error{
		Type:    ErrorNotSupported,
		Message: "message editing not supported by WeChat",
	}
}

// Unsend is not supported by weixin.
func (a *actionsAdapter) Unsend(ctx context.Context, messageID string) (*weixin.OutboundResult, error) {
	return nil, &Error{
		Type:    ErrorNotSupported,
		Message: "message deletion not supported by WeChat",
	}
}
