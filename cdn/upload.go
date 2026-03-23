// Package cdn provides CDN utilities for WeChat media upload/download.
package cdn

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tingly-dev/weixin/crypto"
)

const (
	// MaxUploadRetries is the maximum number of retry attempts for CDN upload.
	MaxUploadRetries = 3
	// RetryDelay is the delay between retry attempts.
	RetryDelay = 1 * time.Second
)

// UploadBufferToCdn uploads encrypted buffer to WeChat CDN with retry logic.
// Returns the download encrypted_query_param from the CDN response.
func UploadBufferToCdn(ctx context.Context, plaintext []byte, uploadParam, filekey, cdnBaseURL string, aesKey []byte) (string, error) {
	// Encrypt plaintext
	ciphertext, err := crypto.EncryptAesEcb(plaintext, aesKey)
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}

	// Build upload URL
	uploadURL := BuildUploadURL(cdnBaseURL, uploadParam, filekey)

	var lastError error
	for attempt := 1; attempt <= MaxUploadRetries; attempt++ {
		downloadParam, err := uploadToCdn(ctx, uploadURL, ciphertext)
		if err == nil {
			return downloadParam, nil
		}

		lastError = err

		// Don't retry on client errors (4xx)
		if isClientError(err) {
			return "", err
		}

		// Retry on server errors (5xx) or network errors
		if attempt < MaxUploadRetries {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(RetryDelay):
				// Continue to next attempt
			}
		}
	}

	return "", fmt.Errorf("CDN upload failed after %d attempts: %w", MaxUploadRetries, lastError)
}

// uploadToCdn performs a single upload attempt.
func uploadToCdn(ctx context.Context, uploadURL string, ciphertext []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, bytes.NewReader(ciphertext))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		errMsg := resp.Header.Get("x-error-message")
		if errMsg == "" {
			body, _ := io.ReadAll(resp.Body)
			errMsg = string(body)
		}
		return "", &ClientError{
			StatusCode: resp.StatusCode,
			Message:    errMsg,
		}
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := resp.Header.Get("x-error-message")
		if errMsg == "" {
			errMsg = fmt.Sprintf("status %d", resp.StatusCode)
		}
		return "", &ServerError{
			StatusCode: resp.StatusCode,
			Message:    errMsg,
		}
	}

	// Extract download param from response header
	downloadParam := resp.Header.Get("x-encrypted-param")
	if downloadParam == "" {
		return "", fmt.Errorf("CDN response missing x-encrypted-param header")
	}

	return downloadParam, nil
}

// ClientError represents a client-side error (4xx).
type ClientError struct {
	StatusCode int
	Message    string
}

func (e *ClientError) Error() string {
	return fmt.Sprintf("CDN client error %d: %s", e.StatusCode, e.Message)
}

// ServerError represents a server-side error (5xx).
type ServerError struct {
	StatusCode int
	Message    string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("CDN server error %d: %s", e.StatusCode, e.Message)
}

// isClientError checks if the error is a client error.
func isClientError(err error) bool {
	_, ok := err.(*ClientError)
	return ok
}
