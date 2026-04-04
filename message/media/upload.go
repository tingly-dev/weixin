// Package media provides media handling utilities for weixin.
package media

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/tingly-dev/weixin/api"
	"github.com/tingly-dev/weixin/message/cdn"
	"github.com/tingly-dev/weixin/types"
)

// UploadedFileInfo contains information about an uploaded file.
type UploadedFileInfo struct {
	FileKey                     string
	DownloadEncryptedQueryParam string
	AESKey                      []byte // Raw 16 bytes
	FileSize                    int64  // Plaintext size
	FileSizeCiphertext          int64  // Ciphertext size
}

// DownloadRemoteMediaToTemp downloads a remote media URL to a temp file.
func DownloadRemoteMediaToTemp(ctx context.Context, url, destDir string) (string, error) {
	// Create temp directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	// Download file
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %d %s", resp.StatusCode, resp.Status)
	}

	// Determine file extension
	ext := GetExtensionFromContentTypeOrURL(resp.Header.Get("Content-Type"), url)

	// Generate temp filename
	tempFile := filepath.Join(destDir, "weixin-remote-"+generateRandomID()+ext)

	// Write to file
	f, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(tempFile)
		return "", fmt.Errorf("write file: %w", err)
	}

	return tempFile, nil
}

// UploadMediaToCDN uploads a media file to WeChat CDN with encryption.
// This is the complete pipeline: read → hash → encrypt → getUploadURL → uploadToCDN.
func UploadMediaToCDN(ctx context.Context, filePath, toUserID, baseURL, cdnBaseURL, botToken string, mediaType int) (*UploadedFileInfo, error) {
	// Read file
	plaintext, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	rawSize := int64(len(plaintext))

	// Calculate MD5
	rawMD5 := fmt.Sprintf("%x", md5.Sum(plaintext))

	// Generate AES key and filekey
	aesKey := make([]byte, 16)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("generate AES key: %w", err)
	}

	filekey := make([]byte, 16)
	if _, err := rand.Read(filekey); err != nil {
		return nil, fmt.Errorf("generate filekey: %w", err)
	}
	filekeyHex := hex.EncodeToString(filekey)

	// Calculate ciphertext size
	fileSize := int64(api.AesEcbPaddedSize(int(rawSize)))

	// Create API client
	client := api.NewClient(baseURL, botToken)

	// Get upload URL
	uploadReq := &api.GetUploadURLRequest{
		FileKey:     filekeyHex,
		MediaType:   mediaType,
		ToUserID:    toUserID,
		RawSize:     rawSize,
		RawMD5:      rawMD5,
		FileSize:    fileSize,
		AESKey:      hex.EncodeToString(aesKey),
		NoNeedThumb: true, // Skip thumbnail for now
	}

	uploadResp, err := client.GetUploadURL(ctx, uploadReq)
	if err != nil {
		return nil, fmt.Errorf("get upload URL: %w", err)
	}

	if uploadResp.UploadParam == "" && uploadResp.UploadFullURL == "" {
		return nil, fmt.Errorf("getUploadURL returned empty upload_param and upload_full_url")
	}

	// Upload to CDN (prefer server-returned full URL)
	downloadParam, err := cdn.UploadBufferToCdn(ctx, plaintext, uploadResp.UploadParam, filekeyHex, cdnBaseURL, aesKey, uploadResp.UploadFullURL)
	if err != nil {
		return nil, fmt.Errorf("upload to CDN: %w", err)
	}

	return &UploadedFileInfo{
		FileKey:                     filekeyHex,
		DownloadEncryptedQueryParam: downloadParam,
		AESKey:                      aesKey,
		FileSize:                    rawSize,
		FileSizeCiphertext:          fileSize,
	}, nil
}

// UploadImageToWeixin uploads an image file to WeChat CDN.
func UploadImageToWeixin(ctx context.Context, filePath, toUserID, baseURL, cdnBaseURL, botToken string) (*UploadedFileInfo, error) {
	return UploadMediaToCDN(ctx, filePath, toUserID, baseURL, cdnBaseURL, botToken, types.UploadMediaTypeImage)
}

// UploadVideoToWeixin uploads a video file to WeChat CDN.
func UploadVideoToWeixin(ctx context.Context, filePath, toUserID, baseURL, cdnBaseURL, botToken string) (*UploadedFileInfo, error) {
	return UploadMediaToCDN(ctx, filePath, toUserID, baseURL, cdnBaseURL, botToken, types.UploadMediaTypeVideo)
}

// UploadFileAttachmentToWeixin uploads a file attachment to WeChat CDN.
func UploadFileAttachmentToWeixin(ctx context.Context, filePath, toUserID, baseURL, cdnBaseURL, botToken string) (*UploadedFileInfo, error) {
	return UploadMediaToCDN(ctx, filePath, toUserID, baseURL, cdnBaseURL, botToken, types.UploadMediaTypeFile)
}

// generateRandomID generates a random ID for temp files.
func generateRandomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "temp-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
