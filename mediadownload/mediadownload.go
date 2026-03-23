// Package mediadownload provides media download and decryption functionality for WeChat.
package mediadownload

import (
	"context"
	"fmt"

	"github.com/tingly-dev/weixin/cdn"
)

// MediaDownloadOpts contains options for downloading media.
type MediaDownloadOpts struct {
	// Paths to downloaded media files (set after download)
	ImagePath string
	VideoPath string
	VoicePath string
	VoiceMime string // MIME type for voice (audio/wav or audio/silk)
	FilePath  string
	FileMime  string
}

// DownloadAndDecryptImage downloads and decrypts an image from CDN.
// aesKeyBase64 is base64-encoded AES key (either raw 16 bytes or hex string).
func DownloadAndDecryptImage(ctx context.Context, encryptedQueryParam, aesKeyBase64, cdnBaseURL string) ([]byte, error) {
	return cdn.DownloadAndDecryptBuffer(ctx, encryptedQueryParam, aesKeyBase64, cdnBaseURL)
}

// DownloadAndDecryptVoice downloads and decrypts a voice message from CDN.
// WeChat voice messages are in SILK format (audio/silk).
func DownloadAndDecryptVoice(ctx context.Context, encryptedQueryParam, aesKeyBase64, cdnBaseURL string) ([]byte, error) {
	return cdn.DownloadAndDecryptBuffer(ctx, encryptedQueryParam, aesKeyBase64, cdnBaseURL)
}

// DownloadAndDecryptFile downloads and decrypts a file attachment from CDN.
func DownloadAndDecryptFile(ctx context.Context, encryptedQueryParam, aesKeyBase64, cdnBaseURL string) ([]byte, error) {
	return cdn.DownloadAndDecryptBuffer(ctx, encryptedQueryParam, aesKeyBase64, cdnBaseURL)
}

// DownloadAndDecryptVideo downloads and decrypts a video from CDN.
func DownloadAndDecryptVideo(ctx context.Context, encryptedQueryParam, aesKeyBase64, cdnBaseURL string) ([]byte, error) {
	return cdn.DownloadAndDecryptBuffer(ctx, encryptedQueryParam, aesKeyBase64, cdnBaseURL)
}

// DownloadAndDecryptBuffer downloads and decrypts media buffer using the appropriate method.
// This is a convenience function that selects the correct download method based on media type.
func DownloadAndDecryptBuffer(ctx context.Context, mediaType, encryptedQueryParam, aesKeyBase64, cdnBaseURL string) ([]byte, error) {
	switch mediaType {
	case "image":
		return DownloadAndDecryptImage(ctx, encryptedQueryParam, aesKeyBase64, cdnBaseURL)
	case "voice":
		return DownloadAndDecryptVoice(ctx, encryptedQueryParam, aesKeyBase64, cdnBaseURL)
	case "file":
		return DownloadAndDecryptFile(ctx, encryptedQueryParam, aesKeyBase64, cdnBaseURL)
	case "video":
		return DownloadAndDecryptVideo(ctx, encryptedQueryParam, aesKeyBase64, cdnBaseURL)
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

// DownloadPlainBuffer downloads unencrypted buffer from CDN.
func DownloadPlainBuffer(ctx context.Context, encryptedQueryParam, cdnBaseURL string) ([]byte, error) {
	return cdn.DownloadPlainBuffer(ctx, encryptedQueryParam, cdnBaseURL)
}
