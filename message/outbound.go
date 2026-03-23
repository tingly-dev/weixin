// Package message provides conversion between WeChat and channel message formats.
package message

import (
	"encoding/base64"
	"fmt"

	"github.com/tingly-dev/weixin"
	"github.com/tingly-dev/weixin/channel"
	"github.com/tingly-dev/weixin/markdown"
	"github.com/tingly-dev/weixin/media"
)

// ConvertToOutboundMessage converts a channel.OutboundMessage to WeChat message format for sending.
// This is a separate function to handle any additional processing needed for outbound messages.
func ConvertToOutboundMessage(msg *channel.OutboundMessage) (toUserID, contextToken string, items []weixin.MessageItem) {
	toUserID = msg.To
	contextToken = msg.ContextToken
	items = ConvertOutboundMessageToList(msg)
	return
}

// BuildTextItem creates a text MessageItem.
// Converts markdown to plain text before creating the item.
func BuildTextItem(text string) weixin.MessageItem {
	plainText := markdown.ToPlainText(text)
	return weixin.MessageItem{
		Type: weixin.MessageItemTypeText,
		TextItem: &weixin.TextItem{
			Text: plainText,
		},
	}
}

// ConvertOutboundMessageToList converts a channel.OutboundMessage to WeChat MessageItem list.
// Applies markdown stripping to text content.
func ConvertOutboundMessageToList(msg *channel.OutboundMessage) []weixin.MessageItem {
	var items []weixin.MessageItem

	// Add text item if present (convert markdown to plain text)
	if msg.Text != "" {
		plainText := markdown.ToPlainText(msg.Text)
		items = append(items, weixin.MessageItem{
			Type: weixin.MessageItemTypeText,
			TextItem: &weixin.TextItem{
				Text: plainText,
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

// BuildImageItem creates an image MessageItem with CDN media reference.
func BuildImageItem(encryptQueryParam, aesKey string) weixin.MessageItem {
	return weixin.MessageItem{
		Type: weixin.MessageItemTypeImage,
		ImageItem: &weixin.ImageItem{
			Media: &weixin.CDNMedia{
				EncryptQueryParam: encryptQueryParam,
				AESKey:            aesKey,
			},
		},
	}
}

// BuildVideoItem creates a video MessageItem with CDN media references.
func BuildVideoItem(encryptQueryParam, thumbEncryptParam, aesKey string) weixin.MessageItem {
	return weixin.MessageItem{
		Type: weixin.MessageItemTypeVideo,
		VideoItem: &weixin.VideoItem{
			Media: &weixin.CDNMedia{
				EncryptQueryParam: encryptQueryParam,
				AESKey:            aesKey,
			},
			ThumbMedia: &weixin.CDNMedia{
				EncryptQueryParam: thumbEncryptParam,
				AESKey:            aesKey,
			},
		},
	}
}

// BuildFileItem creates a file MessageItem with CDN media reference.
func BuildFileItem(encryptQueryParam, aesKey, fileName string) weixin.MessageItem {
	return weixin.MessageItem{
		Type: weixin.MessageItemTypeFile,
		FileItem: &weixin.FileItem{
			Media: &weixin.CDNMedia{
				EncryptQueryParam: encryptQueryParam,
				AESKey:            aesKey,
			},
			FileName: fileName,
		},
	}
}

// BuildVoiceItem creates a voice MessageItem with CDN media reference.
func BuildVoiceItem(encryptQueryParam, aesKey string) weixin.MessageItem {
	return weixin.MessageItem{
		Type: weixin.MessageItemTypeVoice,
		VoiceItem: &weixin.VoiceItem{
			Media: &weixin.CDNMedia{
				EncryptQueryParam: encryptQueryParam,
				AESKey:            aesKey,
			},
		},
	}
}

// IsTextOnly checks if the outbound message contains only text.
func IsTextOnly(msg *channel.OutboundMessage) bool {
	return msg.Text != "" && msg.MediaURL == "" && len(msg.MediaData) == 0
}

// HasMedia checks if the outbound message contains media.
func HasMedia(msg *channel.OutboundMessage) bool {
	return msg.MediaURL != "" || len(msg.MediaData) > 0
}

// GetMediaType returns the media type of the outbound message.
func GetMediaType(msg *channel.OutboundMessage) int {
	switch msg.ContentType {
	case "image":
		return weixin.UploadMediaTypeImage
	case "video":
		return weixin.UploadMediaTypeVideo
	case "audio", "voice":
		return weixin.UploadMediaTypeVoice
	default:
		return weixin.UploadMediaTypeFile
	}
}

// BuildImageItemFromUpload creates an image MessageItem from uploaded file info.
func BuildImageItemFromUpload(uploaded *media.UploadedFileInfo, midSize int64) weixin.MessageItem {
	return weixin.MessageItem{
		Type: weixin.MessageItemTypeImage,
		ImageItem: &weixin.ImageItem{
			Media: &weixin.CDNMedia{
				EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
				AESKey:            base64.StdEncoding.EncodeToString(uploaded.AESKey),
				EncryptType:       1,
			},
			MidSize: midSize,
		},
	}
}

// BuildVideoItemFromUpload creates a video MessageItem from uploaded file info.
func BuildVideoItemFromUpload(uploaded *media.UploadedFileInfo, videoSize int64) weixin.MessageItem {
	return weixin.MessageItem{
		Type: weixin.MessageItemTypeVideo,
		VideoItem: &weixin.VideoItem{
			Media: &weixin.CDNMedia{
				EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
				AESKey:            base64.StdEncoding.EncodeToString(uploaded.AESKey),
				EncryptType:       1,
			},
			VideoSize: videoSize,
		},
	}
}

// BuildFileItemFromUpload creates a file MessageItem from uploaded file info.
func BuildFileItemFromUpload(uploaded *media.UploadedFileInfo, fileName string, fileLen int64) weixin.MessageItem {
	return weixin.MessageItem{
		Type: weixin.MessageItemTypeFile,
		FileItem: &weixin.FileItem{
			Media: &weixin.CDNMedia{
				EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
				AESKey:            base64.StdEncoding.EncodeToString(uploaded.AESKey),
				EncryptType:       1,
			},
			FileName: fileName,
			Len:      fmt.Sprintf("%d", fileLen),
		},
	}
}
