// Package message provides conversion between WeChat and channel message formats.
package message

import (
	"fmt"
	"time"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/channel"
)

// ConvertInboundMessage converts a WeixinMessage to a channel.Message.
func ConvertInboundMessage(msg *weixin.WeixinMessage, accountID string) *channel.Message {
	if msg == nil {
		return nil
	}

	// Extract text and attachments from item list
	var text string
	var attachments []channel.Attachment

	for _, item := range msg.ItemList {
		switch item.Type {
		case weixin.MessageItemTypeText:
			if item.TextItem != nil {
				text = item.TextItem.Text
			}

		case weixin.MessageItemTypeImage:
			if item.ImageItem != nil {
				attachments = append(attachments, channel.Attachment{
					URL:         item.ImageItem.URL,
					ContentType: "image",
				})
			}

		case weixin.MessageItemTypeVoice:
			if item.VoiceItem != nil {
				attachments = append(attachments, channel.Attachment{
					ContentType: "audio",
					FileName:    fmt.Sprintf("voice_%d.silk", msg.CreateTimeMs),
				})
			}

		case weixin.MessageItemTypeFile:
			if item.FileItem != nil {
				attachments = append(attachments, channel.Attachment{
					FileName:    item.FileItem.FileName,
					ContentType: "file",
				})
			}

		case weixin.MessageItemTypeVideo:
			if item.VideoItem != nil {
				attachments = append(attachments, channel.Attachment{
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

	return &channel.Message{
		MessageID:    fmt.Sprintf("%d", msg.MessageID),
		ChannelID:    channel.ChannelIDWeChat,
		AccountID:    accountID,
		ChatType:     channel.ChatTypeDirect, // WeChat only supports direct messages
		Timestamp:    timestamp,
		Text:         text,
		Attachments:  attachments,
		From:         msg.ToUserID,     // Bot ID (sender of this message in the system)
		SenderID:     msg.ToUserID,
		To:           msg.FromUserID,   // User ID (who sent the message - this is the reply target)
		ContextToken: msg.ContextToken,
		Metadata: map[string]interface{}{
			"session_id":    msg.SessionID,
			"message_type":  msg.MessageType,
			"message_state": msg.MessageState,
			// Store original sender for reference
			"from_user_id":  msg.FromUserID,
			"to_user_id":    msg.ToUserID,
		},
	}
}

// ConvertOutboundMessage converts a channel.OutboundMessage to WeChat MessageItem list.
func ConvertOutboundMessage(msg *channel.OutboundMessage) []weixin.MessageItem {
	var items []weixin.MessageItem

	// Add text item if present
	if msg.Text != "" {
		items = append(items, weixin.MessageItem{
			Type: weixin.MessageItemTypeText,
			TextItem: &weixin.TextItem{
				Text: msg.Text,
			},
		})
	}

	// Add media item if present
	if msg.MediaURL != "" || len(msg.MediaData) > 0 {
		switch msg.ContentType {
		case "image":
			items = append(items, weixin.MessageItem{
				Type: weixin.MessageItemTypeImage,
				ImageItem: &weixin.ImageItem{
					URL: msg.MediaURL,
				},
			})
		case "video":
			items = append(items, weixin.MessageItem{
				Type:      weixin.MessageItemTypeVideo,
				VideoItem: &weixin.VideoItem{},
			})
		case "audio", "voice":
			items = append(items, weixin.MessageItem{
				Type:      weixin.MessageItemTypeVoice,
				VoiceItem: &weixin.VoiceItem{},
			})
		default:
			// Treat as file
			items = append(items, weixin.MessageItem{
				Type: weixin.MessageItemTypeFile,
				FileItem: &weixin.FileItem{
					FileName: msg.FileName,
				},
			})
		}
	}

	return items
}
