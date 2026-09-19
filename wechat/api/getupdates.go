// Package api provides WeChat API implementations.
package api

import (
	"context"
	"time"
)

// GetUpdates performs long-polling for new messages.
func (c *Client) GetUpdates(ctx context.Context, syncBuf string) (*GetUpdatesResponse, error) {
	return c.GetUpdatesWithTimeout(ctx, syncBuf, DefaultLongPollTimeout)
}

// GetUpdatesWithTimeout performs long-polling with a custom timeout.
//
// A long-poll timeout is normal and yields an empty response (handled inside
// doRequestWithTimeout, which returns nil on deadline exceeded with respBody
// left empty); any other error — network failure, HTTP error, JSON parse
// failure — is propagated so callers can log/retry instead of silently
// dropping messages.
func (c *Client) GetUpdatesWithTimeout(ctx context.Context, syncBuf string, timeout time.Duration) (*GetUpdatesResponse, error) {
	req := &GetUpdatesRequest{
		GetUpdatesBuf: syncBuf,
		BaseInfo:      c.BuildBaseInfo(),
	}

	resp := &GetUpdatesResponse{}
	if err := c.doRequestWithTimeout(ctx, "ilink/bot/getupdates", timeout, req, resp); err != nil {
		return nil, err
	}

	// Timed-out long-poll: respBody untouched, echo the caller's sync buf.
	if resp.GetUpdatesBuf == "" {
		resp.GetUpdatesBuf = syncBuf
	}

	return resp, nil
}

// GetUpdatesWithRetry performs GetUpdates with retry logic for transient failures.
func (c *Client) GetUpdatesWithRetry(ctx context.Context, syncBuf string, maxRetries int) (*GetUpdatesResponse, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := c.GetUpdates(ctx, syncBuf)
		if err != nil {
			lastErr = err
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}
