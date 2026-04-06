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
	"strconv"
	"time"
)

const (
	// DefaultLongPollTimeout is the default timeout for long-polling requests.
	DefaultLongPollTimeout = 35 * time.Second
	// DefaultAPITimeout is the default timeout for regular API requests.
	DefaultAPITimeout = 15 * time.Second
	// DefaultConfigTimeout is the default timeout for config requests.
	DefaultConfigTimeout = 10 * time.Second
	// SDKVersion is the version reported in iLink-App-ClientVersion header.
	// Encoded as 0x00MMNNPP uint32.
	SDKVersion = "2.1.1"
	// ilinkAppID is the app ID sent in iLink-App-Id header.
	ilinkAppID = "bot"
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
		uinBytes = []byte{0, 0, 0, 0}
	}
	uin := base64.StdEncoding.EncodeToString(uinBytes)

	headers := map[string]string{
		"Content-Type":            "application/json",
		"AuthorizationType":       "ilink_bot_token",
		"X-WECHAT-UIN":            uin,
		"iLink-App-Id":            ilinkAppID,
		"iLink-App-ClientVersion": strconv.FormatUint(uint64(buildClientVersion(SDKVersion)), 10),
	}

	if c.botToken != "" {
		headers["Authorization"] = "Bearer " + c.botToken
	}

	return headers
}

// buildClientVersion encodes a version string "M.N.P" as uint32 0x00MMNNPP.
func buildClientVersion(version string) uint32 {
	major, minor, patch := uint32(0), uint32(0), uint32(0)
	n, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch)
	if err != nil || n < 3 {
		fmt.Printf("[weixin] WARNING: invalid SDK version %q (parsed %d/3 components): %v\n", version, n, err)
	}
	if n < 1 {
		return 0
	}
	return (major << 16) | (minor << 8) | patch
}

// doRequest performs an HTTP POST request.
func (c *Client) doRequest(ctx context.Context, endpoint string, reqBody, respBody interface{}) error {
	return c.doRequestWithClient(ctx, endpoint, c.httpClient, reqBody, respBody)
}

// doRequestWithTimeout performs an HTTP POST request with a custom timeout.
// If the context deadline is exceeded, returns nil (normal for long-poll).
func (c *Client) doRequestWithTimeout(ctx context.Context, endpoint string, timeout time.Duration, reqBody, respBody interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: timeout}

	err := c.doRequestWithClient(ctx, endpoint, client, reqBody, respBody)
	if err != nil && ctx.Err() == context.DeadlineExceeded {
		return nil // Timeout is normal for long-poll
	}
	return err
}

// doRequestWithClient performs an HTTP POST request using the given HTTP client.
func (c *Client) doRequestWithClient(ctx context.Context, endpoint string, client *http.Client, reqBody, respBody interface{}) error {
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/" + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	headers := c.buildHeaders(reqData)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: %d %s: %s", resp.StatusCode, resp.Status, string(respData))
	}

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
