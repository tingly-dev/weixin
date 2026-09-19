package wecom

import (
	"time"

	"github.com/tingly-dev/weixin/types"
)

// convertToChannelMessage converts a WeCom IncomingMessage to an SDK Message.
func convertToChannelMessage(msg *IncomingMessage, reqID string) *types.Message {
	chMsg := &types.Message{
		MessageID:    msg.MsgID,
		ChatType:     convertChatType(msg.ChatType),
		Timestamp:    time.Unix(msg.CreateTime, 0),
		From:         msg.From.UserID,
		SenderID:     msg.From.UserID,
		ContextToken: reqID, // store req_id for reply correlation
		Metadata:     make(map[string]interface{}),
	}

	if msg.ChatID != "" {
		chMsg.Metadata["chatId"] = msg.ChatID
	}

	// Extract text content
	switch {
	case msg.Text != nil:
		chMsg.Text = msg.Text.Content
	case msg.Mixed != nil:
		for _, item := range msg.Mixed.Items {
			if item.Text != nil {
				chMsg.Text += item.Text.Content
			}
		}
	case msg.Voice != nil:
		chMsg.Text = msg.Voice.Content
	}

	// Extract quote. WeCom's quote protocol carries only inline content (see
	// MsgQuote) — there is no ID for the quoted message on the wire, so
	// ReplyToID is intentionally left unset here rather than set to this
	// message's own ID (which would misleadingly look like a real quoted-
	// message reference). Metadata["reply_to_is_quote"] mirrors the
	// convention used by the wechat package for the same "is a reply, body
	// may or may not be resolved" signal.
	if msg.Quote != nil {
		quoteText := extractQuoteText(msg.Quote)
		chMsg.Metadata["quote"] = map[string]interface{}{
			"msgType": msg.Quote.MsgType,
			"text":    quoteText,
		}
		chMsg.Metadata["reply_to_is_quote"] = true
	}

	// Extract attachments
	switch {
	case msg.Image != nil:
		chMsg.Attachments = append(chMsg.Attachments, types.Attachment{
			URL:      msg.Image.URL,
			MimeType: "image",
		})
	case msg.File != nil:
		chMsg.Attachments = append(chMsg.Attachments, types.Attachment{
			URL:      msg.File.URL,
			MimeType: "file",
		})
	case msg.Video != nil:
		chMsg.Attachments = append(chMsg.Attachments, types.Attachment{
			URL:      msg.Video.URL,
			MimeType: "video",
		})
	case msg.Mixed != nil:
		for _, item := range msg.Mixed.Items {
			if item.Image != nil {
				chMsg.Attachments = append(chMsg.Attachments, types.Attachment{
					URL:      item.Image.URL,
					MimeType: "image",
				})
			}
		}
	}

	// Store encryption keys in metadata for later download
	if msg.Image != nil && msg.Image.AESKey != "" {
		chMsg.Metadata["image_aes_key"] = msg.Image.AESKey
	}
	if msg.File != nil && msg.File.AESKey != "" {
		chMsg.Metadata["file_aes_key"] = msg.File.AESKey
	}
	if msg.Video != nil && msg.Video.AESKey != "" {
		chMsg.Metadata["video_aes_key"] = msg.Video.AESKey
	}

	return chMsg
}

func convertChatType(chatType string) types.ChatType {
	switch chatType {
	case "group":
		return types.ChatTypeGroup
	default:
		return types.ChatTypeDirect
	}
}

func extractQuoteText(quote *MsgQuote) string {
	switch {
	case quote.Text != nil:
		return quote.Text.Content
	case quote.Voice != nil:
		return quote.Voice.Content
	default:
		return ""
	}
}
