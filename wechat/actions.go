package wechat

import (
	"context"
	"fmt"

	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/message"
	"github.com/tingly-dev/weixin/types"
)

// ActionsAdapter handles message actions for weixin.
type ActionsAdapter struct {
	bot *WechatBot
}

// NewActionsAdapter creates a new actions adapter.
func NewActionsAdapter(bot *WechatBot) *ActionsAdapter {
	return &ActionsAdapter{bot: bot}
}

// Send sends a text message to weixin.
func (a *ActionsAdapter) Send(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	// Get account
	account, err := a.bot.Accounts().Get(msg.AccountID)
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
		return &types.OutboundResult{
			OK:    false,
			Error: err.Error(),
		}, err
	}

	return &types.OutboundResult{
		OK:        true,
		MessageID: "wc-" + account.ID,
	}, nil
}

// SendMedia sends a media message to weixin.
func (a *ActionsAdapter) SendMedia(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	// For WeChat, media and text messages use the same endpoint
	return a.Send(ctx, msg)
}

// SendStream is not supported by weixin.
func (a *ActionsAdapter) SendStream(ctx context.Context, msg *types.OutboundMessage) (*types.OutboundResult, error) {
	return nil, &Error{
		Type:    ErrorNotSupported,
		Message: "streaming not supported by WeChat ilink protocol",
	}
}

// React is not supported by weixin.
func (a *ActionsAdapter) React(ctx context.Context, reaction *types.Reaction) (*types.OutboundResult, error) {
	return nil, &Error{
		Type:    ErrorNotSupported,
		Message: "reactions not supported by WeChat",
	}
}

// Edit is not supported by weixin.
func (a *ActionsAdapter) Edit(ctx context.Context, messageID string, text string) (*types.OutboundResult, error) {
	return nil, &Error{
		Type:    ErrorNotSupported,
		Message: "message editing not supported by WeChat",
	}
}

// Unsend is not supported by weixin.
func (a *ActionsAdapter) Unsend(ctx context.Context, messageID string) (*types.OutboundResult, error) {
	return nil, &Error{
		Type:    ErrorNotSupported,
		Message: "message deletion not supported by WeChat",
	}
}
