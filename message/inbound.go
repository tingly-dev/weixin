// Package message provides conversion between WeChat and SDK message formats.
package message

import (
	"fmt"
	"time"

	"github.com/tingly-dev/weixin/types"
	"github.com/tingly-dev/weixin/wechat/api"
)

// ConvertInboundMessage converts a WeixinMessage to an SDK Message.
func ConvertInboundMessage(msg *api.WeixinMessage, accountID string) *types.Message {
	if msg == nil {
		return nil
	}

	// Extract text and attachments from item list
	var text string
	var attachments []types.Attachment

	for _, item := range msg.ItemList {
		switch item.Type {
		case api.MessageItemTypeText:
			if item.TextItem != nil {
				text = item.TextItem.Text
			}

		case api.MessageItemTypeImage:
			if item.ImageItem != nil {
				attachments = append(attachments, types.Attachment{
					URL:         item.ImageItem.URL,
					ContentType: "image",
				})
			}

		case api.MessageItemTypeVoice:
			if item.VoiceItem != nil {
				attachments = append(attachments, types.Attachment{
					ContentType: "audio",
					FileName:    fmt.Sprintf("voice_%d.silk", msg.CreateTimeMs),
				})
			}

		case api.MessageItemTypeFile:
			if item.FileItem != nil {
				attachments = append(attachments, types.Attachment{
					FileName:    item.FileItem.FileName,
					ContentType: "file",
				})
			}

		case api.MessageItemTypeVideo:
			if item.VideoItem != nil {
				attachments = append(attachments, types.Attachment{
					ContentType: "video",
				})
			}
		}
	}

	// Convert timestamp (ms to seconds)
	var timestamp time.Time
	if msg.CreateTimeMs > 0 {
		timestamp = time.Unix(msg.CreateTimeMs/1000, (msg.CreateTimeMs%1000)*1e6)
	} else {
		timestamp = time.Now()
	}

	return &types.Message{
		MessageID:    fmt.Sprintf("%d", msg.MessageID),
		AccountID:    accountID,
		ChatType:     types.ChatTypeDirect, // WeChat only supports direct messages
		Timestamp:    timestamp,
		Text:         text,
		Attachments:  attachments,
		From:         msg.ToUserID, // Bot ID (sender of this message in the system)
		SenderID:     msg.ToUserID,
		To:           msg.FromUserID, // User ID (who sent the message - this is the reply target)
		ContextToken: msg.ContextToken,
		Metadata: map[string]interface{}{
			"session_id":    msg.SessionID,
			"message_type":  msg.MessageType,
			"message_state": msg.MessageState,
			// Store original sender for reference
			"from_user_id": msg.FromUserID,
			"to_user_id":   msg.ToUserID,
		},
	}
}
