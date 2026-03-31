// Package cdn provides CDN utilities for WeChat media upload/download.
package cdn

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/tingly-dev/weixin/api"
)

// DownloadAndDecryptBuffer downloads and decrypts media from WeChat CDN.
// When fullURL is non-empty, it is used directly instead of client-side URL construction.
// aesKeyBase64 can be either:
//   - base64(raw 16 bytes) - for images
//   - base64(hex string of 16 bytes) - for file/voice/video
func DownloadAndDecryptBuffer(ctx context.Context, encryptedQueryParam, aesKeyBase64, cdnBaseURL string, fullURL ...string) ([]byte, error) {
	// Parse AES key
	aesKey, err := parseAesKey(aesKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("parse AES key: %w", err)
	}

	// Build download URL: prefer server-returned full URL
	downloadURL := resolveDownloadURL(cdnBaseURL, encryptedQueryParam, fullURL)

	// Download encrypted data
	encrypted, err := fetchCdnBytes(ctx, downloadURL)
	if err != nil {
		return nil, fmt.Errorf("fetch CDN bytes: %w", err)
	}

	// Decrypt
	plaintext, err := api.DecryptAesEcb(encrypted, aesKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// DownloadPlainBuffer downloads plain (unencrypted) bytes from CDN.
func DownloadPlainBuffer(ctx context.Context, encryptedQueryParam, cdnBaseURL string, fullURL ...string) ([]byte, error) {
	downloadURL := resolveDownloadURL(cdnBaseURL, encryptedQueryParam, fullURL)
	return fetchCdnBytes(ctx, downloadURL)
}

// resolveDownloadURL returns the download URL, preferring server-returned full URL.
func resolveDownloadURL(cdnBaseURL, encryptedQueryParam string, fullURL []string) string {
	if len(fullURL) > 0 && fullURL[0] != "" {
		return fullURL[0]
	}
	return BuildDownloadURL(encryptedQueryParam, cdnBaseURL)
}

// fetchCdnBytes downloads raw bytes from CDN.
func fetchCdnBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CDN download failed: %d %s: %s", resp.StatusCode, resp.Status, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return data, nil
}

// parseAesKey parses AES key from base64.
// Handles two formats:
//   - base64(raw 16 bytes) -> direct use
//   - base64(hex string of 16 bytes) -> decode base64, then parse hex
func parseAesKey(aesKeyBase64 string) ([]byte, error) {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(aesKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	// Case 1: Decoded is exactly 16 bytes (raw key)
	if len(decoded) == 16 {
		return decoded, nil
	}

	// Case 2: Decoded is 32 bytes (hex-encoded key)
	if len(decoded) == 32 {
		// Check if it's valid hex
		decodedStr := string(decoded)
		keyBytes, err := hex.DecodeString(decodedStr)
		if err == nil && len(keyBytes) == 16 {
			return keyBytes, nil
		}
	}

	return nil, fmt.Errorf("aes_key must decode to 16 raw bytes or 32-char hex string, got %d bytes", len(decoded))
}
