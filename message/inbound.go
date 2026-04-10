// Package message provides conversion between WeChat and SDK message formats.
package message

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/tingly-dev/weixin/types"
	"github.com/tingly-dev/weixin/wechat/api"
)

// ConvertInboundMessage converts a WeixinMessage to an SDK Message.
// cdnBaseURL is used to populate CDN fields in attachments so callers can download media.
func ConvertInboundMessage(msg *api.WeixinMessage, accountID, cdnBaseURL string) *types.Message {
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
				a := types.Attachment{
					ContentType: "image",
					URL:         item.ImageItem.URL,
				}
				if item.ImageItem.Media != nil {
					a.EncryptQueryParam = item.ImageItem.Media.EncryptQueryParam
					a.AESKey = item.ImageItem.Media.AESKey
					a.CDNBaseURL = cdnBaseURL
					if item.ImageItem.Media.FullURL != "" {
						a.URL = item.ImageItem.Media.FullURL
					}
				}
				// Image top-level aeskey field is hex-encoded; convert to base64 to match
			// media.aes_key format expected by parseAesKey in the CDN download layer.
			if a.AESKey == "" && item.ImageItem.AESKey != "" {
				if b, err := hex.DecodeString(item.ImageItem.AESKey); err == nil {
					a.AESKey = base64.StdEncoding.EncodeToString(b)
				}
			}
				attachments = append(attachments, a)
			}

		case api.MessageItemTypeVoice:
			if item.VoiceItem != nil {
				a := types.Attachment{
					ContentType: "audio",
					FileName:    fmt.Sprintf("voice_%d.silk", msg.CreateTimeMs),
				}
				if item.VoiceItem.Media != nil {
					a.EncryptQueryParam = item.VoiceItem.Media.EncryptQueryParam
					a.AESKey = item.VoiceItem.Media.AESKey
					a.CDNBaseURL = cdnBaseURL
				}
				attachments = append(attachments, a)
			}

		case api.MessageItemTypeFile:
			if item.FileItem != nil {
				a := types.Attachment{
					ContentType: "file",
					FileName:    item.FileItem.FileName,
				}
				if item.FileItem.Media != nil {
					a.EncryptQueryParam = item.FileItem.Media.EncryptQueryParam
					a.AESKey = item.FileItem.Media.AESKey
					a.CDNBaseURL = cdnBaseURL
				}
				attachments = append(attachments, a)
			}

		case api.MessageItemTypeVideo:
			if item.VideoItem != nil {
				a := types.Attachment{
					ContentType: "video",
				}
				if item.VideoItem.Media != nil {
					a.EncryptQueryParam = item.VideoItem.Media.EncryptQueryParam
					a.AESKey = item.VideoItem.Media.AESKey
					a.CDNBaseURL = cdnBaseURL
				}
				attachments = append(attachments, a)
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
