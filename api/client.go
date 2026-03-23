// Package api provides the WeChat API client.
package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultLongPollTimeout is the default timeout for long-polling requests.
	DefaultLongPollTimeout = 35 * time.Second
	// DefaultAPITimeout is the default timeout for regular API requests.
	DefaultAPITimeout = 15 * time.Second
	// DefaultConfigTimeout is the default timeout for config requests.
	DefaultConfigTimeout = 10 * time.Second
)

// Client is the WeChat API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	botToken   string
}

// NewClient creates a new WeChat API client.
func NewClient(baseURL, botToken string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: DefaultAPITimeout,
		},
		botToken: botToken,
	}
}

// BaseInfo represents base info sent with each request.
type BaseInfo struct {
	ChannelVersion string `json:"channel_version,omitempty"`
}

// buildHeaders creates the required headers for WeChat API.
func (c *Client) buildHeaders(body []byte) map[string]string {
	// Generate random X-WECHAT-UIN (uint32 -> decimal string -> base64)
	uinBytes := make([]byte, 4)
	if _, err := rand.Read(uinBytes); err != nil {
		// Fallback to zero if random fails
		uinBytes = []byte{0, 0, 0, 0}
	}
	uin := base64.StdEncoding.EncodeToString(uinBytes)

	headers := map[string]string{
		"Content-Type":      "application/json",
		"AuthorizationType": "ilink_bot_token",
		"X-WECHAT-UIN":      uin,
	}

	if c.botToken != "" {
		headers["Authorization"] = "Bearer " + c.botToken
	}

	return headers
}

// doRequest performs an HTTP POST request.
func (c *Client) doRequest(ctx context.Context, endpoint string, reqBody, respBody interface{}) error {
	// Marshal request body
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// Create request
	url := c.baseURL + "/" + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Set headers
	headers := c.buildHeaders(reqData)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: %d %s: %s", resp.StatusCode, resp.Status, string(respData))
	}

	// Unmarshal response
	if respBody != nil {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

// doRequestWithTimeout performs an HTTP POST request with a custom timeout.
func (c *Client) doRequestWithTimeout(ctx context.Context, endpoint string, timeout time.Duration, reqBody, respBody interface{}) error {
	// Create client with custom timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Marshal request body
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// Create request with context and timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := c.baseURL + "/" + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Set headers
	headers := c.buildHeaders(reqData)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		// Check for timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil // Timeout is normal for long-poll
		}
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: %d %s: %s", resp.StatusCode, resp.Status, string(respData))
	}

	// Unmarshal response
	if respBody != nil {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

// SetBotToken updates the bot token for the client.
func (c *Client) SetBotToken(token string) {
	c.botToken = token
}

// GetBotToken returns the current bot token.
func (c *Client) GetBotToken() string {
	return c.botToken
}
