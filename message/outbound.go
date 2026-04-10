// Package message provides conversion between WeChat and SDK message formats.
package message

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/tingly-dev/weixin/message/media"
	"github.com/tingly-dev/weixin/types"
	"github.com/tingly-dev/weixin/wechat/api"
)

// ConvertToOutboundMessage converts an OutboundMessage to WeChat message format for sending.
// This is a separate function to handle any additional processing needed for outbound messages.
func ConvertToOutboundMessage(msg *types.OutboundMessage) (toUserID, contextToken string, items []api.MessageItem) {
	toUserID = msg.To
	contextToken = msg.ContextToken
	items = ConvertOutboundMessageToList(msg)
	return
}

// BuildTextItem creates a text MessageItem.
// Converts markdown to plain text before creating the item.
func BuildTextItem(text string) api.MessageItem {
	plainText := ToPlainText(text)
	return api.MessageItem{
		Type: api.MessageItemTypeText,
		TextItem: &api.TextItem{
			Text: plainText,
		},
	}
}

// ConvertOutboundMessageToList converts an OutboundMessage to WeChat MessageItem list.
// Applies markdown stripping to text content.
func ConvertOutboundMessageToList(msg *types.OutboundMessage) []api.MessageItem {
	var items []api.MessageItem

	// Add text item if present (convert markdown to plain text)
	if msg.Text != "" {
		plainText := ToPlainText(msg.Text)
		items = append(items, api.MessageItem{
			Type: api.MessageItemTypeText,
			TextItem: &api.TextItem{
				Text: plainText,
			},
		})
	}

	// Add media item if present
	if msg.MediaURL != "" || len(msg.MediaData) > 0 {
		switch msg.ContentType {
		case "image":
			items = append(items, api.MessageItem{
				Type: api.MessageItemTypeImage,
				ImageItem: &api.ImageItem{
					URL: msg.MediaURL,
				},
			})
		case "video":
			items = append(items, api.MessageItem{
				Type:      api.MessageItemTypeVideo,
				VideoItem: &api.VideoItem{},
			})
		case "audio", "voice":
			items = append(items, api.MessageItem{
				Type:      api.MessageItemTypeVoice,
				VoiceItem: &api.VoiceItem{},
			})
		default:
			// Treat as file
			items = append(items, api.MessageItem{
				Type: api.MessageItemTypeFile,
				FileItem: &api.FileItem{
					FileName: msg.FileName,
				},
			})
		}
	}

	return items
}

// BuildImageItem creates an image MessageItem with CDN media reference.
func BuildImageItem(encryptQueryParam, aesKey string) api.MessageItem {
	return api.MessageItem{
		Type: api.MessageItemTypeImage,
		ImageItem: &api.ImageItem{
			Media: &api.CDNMedia{
				EncryptQueryParam: encryptQueryParam,
				AESKey:            aesKey,
			},
		},
	}
}

// BuildVideoItem creates a video MessageItem with CDN media references.
func BuildVideoItem(encryptQueryParam, thumbEncryptParam, aesKey string) api.MessageItem {
	return api.MessageItem{
		Type: api.MessageItemTypeVideo,
		VideoItem: &api.VideoItem{
			Media: &api.CDNMedia{
				EncryptQueryParam: encryptQueryParam,
				AESKey:            aesKey,
			},
			ThumbMedia: &api.CDNMedia{
				EncryptQueryParam: thumbEncryptParam,
				AESKey:            aesKey,
			},
		},
	}
}

// BuildFileItem creates a file MessageItem with CDN media reference.
func BuildFileItem(encryptQueryParam, aesKey, fileName string) api.MessageItem {
	return api.MessageItem{
		Type: api.MessageItemTypeFile,
		FileItem: &api.FileItem{
			Media: &api.CDNMedia{
				EncryptQueryParam: encryptQueryParam,
				AESKey:            aesKey,
			},
			FileName: fileName,
		},
	}
}

// BuildVoiceItem creates a voice MessageItem with CDN media reference.
func BuildVoiceItem(encryptQueryParam, aesKey string) api.MessageItem {
	return api.MessageItem{
		Type: api.MessageItemTypeVoice,
		VoiceItem: &api.VoiceItem{
			Media: &api.CDNMedia{
				EncryptQueryParam: encryptQueryParam,
				AESKey:            aesKey,
			},
		},
	}
}

// IsTextOnly checks if the outbound message contains only text.
func IsTextOnly(msg *types.OutboundMessage) bool {
	return msg.Text != "" && msg.MediaURL == "" && len(msg.MediaData) == 0
}

// HasMedia checks if the outbound message contains media.
func HasMedia(msg *types.OutboundMessage) bool {
	return msg.MediaURL != "" || len(msg.MediaData) > 0
}

// GetMediaType returns the media type of the outbound message.
func GetMediaType(msg *types.OutboundMessage) int {
	switch msg.ContentType {
	case "image":
		return types.UploadMediaTypeImage
	case "video":
		return types.UploadMediaTypeVideo
	case "audio", "voice":
		return types.UploadMediaTypeVoice
	default:
		return types.UploadMediaTypeFile
	}
}

// aesKeyToBase64 encodes a raw AES key as base64 of its hex string,
// matching the reference implementation: Buffer.from(aeskey_hex).toString("base64").
func aesKeyToBase64(rawKey []byte) string {
	return base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(rawKey)))
}

// BuildImageItemFromUpload creates an image MessageItem from uploaded file info.
func BuildImageItemFromUpload(uploaded *media.UploadedFileInfo, midSize int64) api.MessageItem {
	return api.MessageItem{
		Type: api.MessageItemTypeImage,
		ImageItem: &api.ImageItem{
			Media: &api.CDNMedia{
				EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
				AESKey:            aesKeyToBase64(uploaded.AESKey),
				EncryptType:       1,
			},
			MidSize: midSize,
		},
	}
}

// BuildVideoItemFromUpload creates a video MessageItem from uploaded file info.
func BuildVideoItemFromUpload(uploaded *media.UploadedFileInfo, videoSize int64) api.MessageItem {
	return api.MessageItem{
		Type: api.MessageItemTypeVideo,
		VideoItem: &api.VideoItem{
			Media: &api.CDNMedia{
				EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
				AESKey:            aesKeyToBase64(uploaded.AESKey),
				EncryptType:       1,
			},
			VideoSize: videoSize,
		},
	}
}

// BuildFileItemFromUpload creates a file MessageItem from uploaded file info.
func BuildFileItemFromUpload(uploaded *media.UploadedFileInfo, fileName string, fileLen int64) api.MessageItem {
	return api.MessageItem{
		Type: api.MessageItemTypeFile,
		FileItem: &api.FileItem{
			Media: &api.CDNMedia{
				EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
				AESKey:            aesKeyToBase64(uploaded.AESKey),
				EncryptType:       1,
			},
			FileName: fileName,
			Len:      fmt.Sprintf("%d", fileLen),
		},
	}
}
