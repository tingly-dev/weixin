// Package media provides media handling utilities for weixin.
package media

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/message/cdn"
)

const (
	// MaxMediaBytes is the maximum allowed media file size (100MB).
	MaxMediaBytes = 100 * 1024 * 1024
)

// InboundMediaOpts contains decrypted media paths and metadata.
type InboundMediaOpts struct {
	DecryptedPicPath   string
	DecryptedVoicePath string
	VoiceMediaType     string
	DecryptedFilePath  string
	FileMediaType      string
	DecryptedVideoPath string
}

// DownloadMediaFromItem downloads and decrypts media from a single MessageItem.
// Returns populated InboundMediaOpts; fields are empty on failure or unsupported types.
func DownloadMediaFromItem(ctx context.Context, item *api.MessageItem, cdnBaseURL, destDir string) (*InboundMediaOpts, error) {
	result := &InboundMediaOpts{}

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create dest dir: %w", err)
	}

	switch item.Type {
	case api.MessageItemTypeImage:
		if err := downloadImage(ctx, item.ImageItem, cdnBaseURL, destDir, result); err != nil {
			return result, fmt.Errorf("download image: %w", err)
		}

	case api.MessageItemTypeVoice:
		if err := downloadVoice(ctx, item.VoiceItem, cdnBaseURL, destDir, result); err != nil {
			return result, fmt.Errorf("download voice: %w", err)
		}

	case api.MessageItemTypeFile:
		if err := downloadFile(ctx, item.FileItem, cdnBaseURL, destDir, result); err != nil {
			return result, fmt.Errorf("download file: %w", err)
		}

	case api.MessageItemTypeVideo:
		if err := downloadVideo(ctx, item.VideoItem, cdnBaseURL, destDir, result); err != nil {
			return result, fmt.Errorf("download video: %w", err)
		}
	}

	return result, nil
}

// downloadImage downloads and decrypts an image item.
func downloadImage(ctx context.Context, img *api.ImageItem, cdnBaseURL, destDir string, result *InboundMediaOpts) error {
	if img == nil || img.Media == nil || (img.Media.EncryptQueryParam == "" && img.Media.FullURL == "") {
		return nil // No media to download
	}

	// Determine AES key source
	var aesKeyBase64 string
	if img.AESKey != "" {
		// Use image_item.aeskey (hex format) - convert to base64
		hexBytes := []byte(img.AESKey)
		aesKeyBase64 = base64.StdEncoding.EncodeToString(hexBytes)
	} else if img.Media.AESKey != "" {
		// Use media.aes_key (already base64)
		aesKeyBase64 = img.Media.AESKey
	}

	var plaintext []byte
	var err error

	if aesKeyBase64 != "" {
		// Download and decrypt
		plaintext, err = cdn.DownloadAndDecryptBuffer(ctx, img.Media.EncryptQueryParam, aesKeyBase64, cdnBaseURL, img.Media.FullURL)
	} else {
		// Download plain (unencrypted)
		plaintext, err = cdn.DownloadPlainBuffer(ctx, img.Media.EncryptQueryParam, cdnBaseURL, img.Media.FullURL)
	}

	if err != nil {
		return err
	}

	// Save to file
	tempFile := filepath.Join(destDir, "image-"+generateRandomID()+".jpg")
	if err := os.WriteFile(tempFile, plaintext, 0644); err != nil {
		return fmt.Errorf("write image: %w", err)
	}

	result.DecryptedPicPath = tempFile
	return nil
}

// downloadVoice downloads and decrypts a voice item.
func downloadVoice(ctx context.Context, voice *api.VoiceItem, cdnBaseURL, destDir string, result *InboundMediaOpts) error {
	if voice == nil || voice.Media == nil || (voice.Media.EncryptQueryParam == "" && voice.Media.FullURL == "") || voice.Media.AESKey == "" {
		return nil
	}

	// Download and decrypt
	plaintext, err := cdn.DownloadAndDecryptBuffer(ctx, voice.Media.EncryptQueryParam, voice.Media.AESKey, cdnBaseURL, voice.Media.FullURL)
	if err != nil {
		return err
	}

	// TODO: Transcode SILK to WAV if needed
	// For now, save as raw SILK
	tempFile := filepath.Join(destDir, "voice-"+generateRandomID()+".silk")
	if err := os.WriteFile(tempFile, plaintext, 0644); err != nil {
		return fmt.Errorf("write voice: %w", err)
	}

	result.DecryptedVoicePath = tempFile
	result.VoiceMediaType = "audio/silk"
	return nil
}

// downloadFile downloads and decrypts a file item.
func downloadFile(ctx context.Context, fileItem *api.FileItem, cdnBaseURL, destDir string, result *InboundMediaOpts) error {
	if fileItem == nil || fileItem.Media == nil || (fileItem.Media.EncryptQueryParam == "" && fileItem.Media.FullURL == "") || fileItem.Media.AESKey == "" {
		return nil
	}

	// Download and decrypt
	plaintext, err := cdn.DownloadAndDecryptBuffer(ctx, fileItem.Media.EncryptQueryParam, fileItem.Media.AESKey, cdnBaseURL, fileItem.Media.FullURL)
	if err != nil {
		return err
	}

	// Determine extension from filename
	ext := filepath.Ext(fileItem.FileName)
	if ext == "" {
		ext = ".bin"
	}

	tempFile := filepath.Join(destDir, "file-"+generateRandomID()+ext)
	if err := os.WriteFile(tempFile, plaintext, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	result.DecryptedFilePath = tempFile
	result.FileMediaType = GetMIMEFromFilename(fileItem.FileName)
	return nil
}

// downloadVideo downloads and decrypts a video item.
func downloadVideo(ctx context.Context, video *api.VideoItem, cdnBaseURL, destDir string, result *InboundMediaOpts) error {
	if video == nil || video.Media == nil || (video.Media.EncryptQueryParam == "" && video.Media.FullURL == "") || video.Media.AESKey == "" {
		return nil
	}

	// Download and decrypt
	plaintext, err := cdn.DownloadAndDecryptBuffer(ctx, video.Media.EncryptQueryParam, video.Media.AESKey, cdnBaseURL, video.Media.FullURL)
	if err != nil {
		return err
	}

	tempFile := filepath.Join(destDir, "video-"+generateRandomID()+".mp4")
	if err := os.WriteFile(tempFile, plaintext, 0644); err != nil {
		return fmt.Errorf("write video: %w", err)
	}

	result.DecryptedVideoPath = tempFile
	return nil
}
