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

// LongPollAdapter handles long-polling message synchronization for weixin.
type LongPollAdapter struct {
	plugin weixin.PluginInterface
}

// NewLongPollAdapter creates a new long-poll adapter.
func NewLongPollAdapter(plugin weixin.PluginInterface) *LongPollAdapter {
	return &LongPollAdapter{plugin: plugin}
}

// SessionExpiredErrCode is the error code for session expiration.
const SessionExpiredErrCode = -14

// GetUpdates fetches new messages using long-polling.
func (a *LongPollAdapter) GetUpdates(ctx context.Context, req *channel.GetUpdatesRequest) (*channel.GetUpdatesResult, error) {
	// Get account
	account, err := a.plugin.Accounts().Get(req.AccountID)
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

	// Call getUpdates with timeout
	resp, err := client.GetUpdates(ctx, req.SyncBuf)
	if err != nil {
		return nil, fmt.Errorf("get updates: %w", err)
	}

	// Check for session expiration
	if resp.ErrCode == SessionExpiredErrCode {
		return &channel.GetUpdatesResult{
			ErrCode: int(resp.ErrCode),
			ErrMsg:  resp.ErrMsg,
		}, nil
	}

	// Convert WeixinMessage to channel.Message
	messages := make([]*channel.Message, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		// Only process USER messages (ignore BOT messages)
		if msg.MessageType == weixin.MessageTypeUser {
			channelMsg := message.ConvertInboundMessage(&msg, req.AccountID)
			if channelMsg != nil {
				messages = append(messages, channelMsg)
			}
		}
	}

	return &channel.GetUpdatesResult{
		Messages:           messages,
		SyncBuf:            resp.GetUpdatesBuf,
		LongPollingTimeout: resp.LongPollingTimeoutMs,
		ErrCode:            0,
	}, nil
}
