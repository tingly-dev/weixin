// Package api provides WeChat API implementations.
package api

import (
	"context"
	"fmt"
)

// GetUploadURL gets a pre-signed CDN upload URL.
func (c *Client) GetUploadURL(ctx context.Context, req *GetUploadURLRequest) (*GetUploadURLResponse, error) {
	if req.BaseInfo == nil {
		req.BaseInfo = &BaseInfo{}
	}
	req.BaseInfo.ChannelVersion = SDKVersion

	resp := &GetUploadURLResponse{}
	err := c.doRequest(ctx, "ilink/bot/getuploadurl", req, resp)
	if err != nil {
		return nil, err
	}

	if resp.Ret != 0 || resp.UploadParam == "" && resp.UploadFullURL == "" {
		msg := resp.ErrMsg
		if msg == "" {
			msg = "no upload_param or upload_full_url returned"
		}
		return nil, fmt.Errorf("getUploadURL failed (ret=%d): %s", resp.Ret, msg)
	}

	return resp, nil
}

// GetConfig gets account configuration including typing ticket.
func (c *Client) GetConfig(ctx context.Context, ilinkUserID, contextToken string) (*GetConfigResponse, error) {
	req := &GetConfigRequest{
		IlinkUserID:  ilinkUserID,
		ContextToken: contextToken,
		BaseInfo: &BaseInfo{
			ChannelVersion: SDKVersion,
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
			ChannelVersion: SDKVersion,
		},
	}

	return c.doRequestWithTimeout(ctx, "ilink/bot/sendtyping", DefaultConfigTimeout, req, nil)
}
