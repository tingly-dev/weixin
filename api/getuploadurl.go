// Package api provides WeChat API implementations.
package api

import (
	"context"
)

// GetUploadURL gets a pre-signed CDN upload URL.
func (c *Client) GetUploadURL(ctx context.Context, req *GetUploadURLRequest) (*GetUploadURLResponse, error) {
	if req.BaseInfo == nil {
		req.BaseInfo = &BaseInfo{}
	}
	req.BaseInfo.ChannelVersion = "1.0.0"

	resp := &GetUploadURLResponse{}
	err := c.doRequest(ctx, "ilink/bot/getuploadurl", req, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetConfig gets account configuration including typing ticket.
func (c *Client) GetConfig(ctx context.Context, ilinkUserID, contextToken string) (*GetConfigResponse, error) {
	req := &GetConfigRequest{
		IlinkUserID:  ilinkUserID,
		ContextToken: contextToken,
		BaseInfo: &BaseInfo{
			ChannelVersion: "1.0.0",
		},
	}

	resp := &GetConfigResponse{}
	err := c.doRequestWithTimeout(ctx, "ilink/bot/getconfig", DefaultConfigTimeout, req, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// SendTyping sends or cancels a typing indicator.
func (c *Client) SendTyping(ctx context.Context, ilinkUserID, typingTicket string, status int) error {
	req := &SendTypingRequest{
		IlinkUserID:  ilinkUserID,
		TypingTicket: typingTicket,
		Status:       status,
		BaseInfo: &BaseInfo{
			ChannelVersion: "1.0.0",
		},
	}

	return c.doRequestWithTimeout(ctx, "ilink/bot/sendtyping", DefaultConfigTimeout, req, nil)
}
