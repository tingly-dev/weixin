// Package api provides WeChat API implementations.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// GetBotQRCode fetches a QR code for login.
func (c *Client) GetBotQRCode(ctx context.Context, botType string) (*QRCodeResponse, error) {
	if botType == "" {
		botType = "3" // Default bot type
	}

	// Build URL with query params (GET request)
	u, err := url.Parse(c.baseURL + "/ilink/bot/get_bot_qrcode")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	query := u.Query()
	query.Set("bot_type", botType)
	u.RawQuery = query.Encode()

	fmt.Printf("GetBotQRCode URL: %s\n", u.String())

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers (no Authorization for QR code request)
	headers := c.buildHeaders([]byte{})
	for k, v := range headers {
		// Skip Authorization header for QR code login
		if k == "Authorization" {
			continue
		}
		req.Header.Set(k, v)
		fmt.Printf("Header: %s = %s\n", k, v)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Log response for debugging
	fmt.Printf("GetBotQRCode response: %s\n", string(body))

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d %s: %s", resp.StatusCode, resp.Status, string(body))
	}

	// Unmarshal response
	var result QRCodeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// GetQRStatus polls the QR code status.
func (c *Client) GetQRStatus(ctx context.Context, qrcode string) (*QRStatusResponse, error) {
	// Build URL with query params
	u, err := url.Parse(c.baseURL + "/ilink/bot/get_qrcode_status")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	query := u.Query()
	query.Set("qrcode", qrcode)
	u.RawQuery = query.Encode()

	// Create request with longer timeout for long-poll
	client := &http.Client{Timeout: DefaultLongPollTimeout}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	headers := c.buildHeaders([]byte{})
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	// Add required header for QR status polling
	req.Header.Set("iLink-App-ClientVersion", "1")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		// Timeout is normal, return "wait" status
		if ctx.Err() == context.DeadlineExceeded {
			return &QRStatusResponse{Status: "wait"}, nil
		}
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d %s: %s", resp.StatusCode, resp.Status, string(body))
	}

	// Unmarshal response
	var result QRStatusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
