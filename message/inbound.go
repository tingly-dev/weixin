// Package message provides conversion between WeChat and SDK message formats.
package message

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
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

	result := &types.Message{
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

	applyQuoteContext(result, msg)

	return result
}

// findReferenceItem returns the first item carrying a quote/reply (since
// openclaw-weixin v2.4.9-beta.0), if any.
func findReferenceItem(msg *api.WeixinMessage) *api.MessageItem {
	for i := range msg.ItemList {
		if msg.ItemList[i].RefMsg != nil {
			return &msg.ItemList[i]
		}
	}
	return nil
}

// mediaLabel returns the bracketed display label WeChat uses for a
// content-less media item, matching the upstream reference implementation's
// getMediaLabel (keyed off Type, not whether the nested item struct happens
// to be populated).
func mediaLabel(itemType int) string {
	switch itemType {
	case api.MessageItemTypeImage:
		return "[图片]"
	case api.MessageItemTypeVideo:
		return "[视频]"
	case api.MessageItemTypeFile:
		return "[文件]"
	case api.MessageItemTypeVoice:
		return "[语音]"
	default:
		return ""
	}
}

// inlineQuoteBody builds a display string for a quoted message's inline
// content, when the server provides it. Only covers text and a short label
// for media items; this SDK doesn't need the full CDN-stitched Attachment
// conversion just to describe what was quoted.
func inlineQuoteBody(ref *api.RefMessage) string {
	var parts []string
	if title := ref.Title; title != "" {
		parts = append(parts, title)
	}
	if item := ref.MessageItem; item != nil {
		if item.TextItem != nil && item.TextItem.Text != "" {
			parts = append(parts, item.TextItem.Text)
		} else if item.VoiceItem != nil && item.VoiceItem.Text != "" {
			parts = append(parts, item.VoiceItem.Text)
		} else if label := mediaLabel(item.Type); label != "" {
			parts = append(parts, label)
		}
	}
	return strings.Join(parts, " | ")
}

// applyQuoteContext populates reply-to fields on result when msg contains a
// quote/reply (ref_msg).
//
// Older WeChat clients send the quoted content inline (ref.MessageItem /
// ref.Title); newer clients may send only ref.SvrID, omitting the content.
// This SDK does not maintain a local message-history cache, so an ID-only
// quote leaves ReplyToBody empty; Metadata["reply_to_is_quote"] distinguishes
// that case from "no reply at all". ref.PartialText (a highlighted sub-range
// of the quoted message) is preserved on the wire in api.RefMessage but is
// not resolved here for the same reason.
func applyQuoteContext(result *types.Message, msg *api.WeixinMessage) {
	item := findReferenceItem(msg)
	if item == nil {
		return
	}
	ref := item.RefMsg

	// Only set ReplyToID when a real id was found; a ref_msg with neither
	// svr_id nor a nested message_item.msg_id (e.g. an empty ref_msg, or one
	// carrying only a title) must not surface a fake "0" id.
	if ref.SvrID != 0 {
		result.ReplyToID = fmt.Sprintf("%d", ref.SvrID)
	} else if ref.MessageItem != nil && ref.MessageItem.MsgID != 0 {
		result.ReplyToID = fmt.Sprintf("%d", ref.MessageItem.MsgID)
	}
	result.ReplyToBody = inlineQuoteBody(ref)

	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["reply_to_is_quote"] = true
}
