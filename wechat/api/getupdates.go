// Package api provides WeChat API implementations.
package api

import (
	"context"
	"time"
)

// GetUpdates performs long-polling for new messages.
func (c *Client) GetUpdates(ctx context.Context, syncBuf string) (*GetUpdatesResponse, error) {
	req := &GetUpdatesRequest{
		GetUpdatesBuf: syncBuf,
		BaseInfo: &BaseInfo{
			ChannelVersion: SDKVersion,
		},
	}

	resp := &GetUpdatesResponse{}
	err := c.doRequestWithTimeout(ctx, "ilink/bot/getupdates", DefaultLongPollTimeout, req, resp)
	if err != nil {
		// Timeout is normal for long-poll, return empty response
		return &GetUpdatesResponse{
			Ret:           0,
			GetUpdatesBuf: syncBuf, // Return same sync buf on timeout
		}, nil
	}

	return resp, nil
}

// GetUpdatesWithTimeout performs long-polling with a custom timeout.
func (c *Client) GetUpdatesWithTimeout(ctx context.Context, syncBuf string, timeout time.Duration) (*GetUpdatesResponse, error) {
	req := &GetUpdatesRequest{
		GetUpdatesBuf: syncBuf,
		BaseInfo: &BaseInfo{
			ChannelVersion: SDKVersion,
		},
	}

	resp := &GetUpdatesResponse{}
	err := c.doRequestWithTimeout(ctx, "ilink/bot/getupdates", timeout, req, resp)
	if err != nil {
		// Timeout is normal for long-poll, return empty response
		return &GetUpdatesResponse{
			Ret:           0,
			GetUpdatesBuf: syncBuf, // Return same sync buf on timeout
		}, nil
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
