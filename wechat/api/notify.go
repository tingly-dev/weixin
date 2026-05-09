// Package api provides WeChat API implementations.
package api

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	// botAgentMaxLen caps the sanitized bot_agent string length in bytes.
	botAgentMaxLen = 256
)

var (
	botAgentProductRe = regexp.MustCompile(`^[A-Za-z0-9_.\-]{1,32}/[A-Za-z0-9_.+\-]{1,32}$`)
	botAgentCommentRe = regexp.MustCompile(`^[\x20-\x27\x2A-\x7E]{1,64}$`)
)

// SanitizeBotAgent normalizes a user-supplied BotAgent string into a
// wire-safe BaseInfo.bot_agent value.
//
// Grammar (UA-style):
//
//	bot_agent = product *( SP product )
//	product   = name "/" version [ SP "(" comment ")" ]
//	name      = 1*32( ALPHA / DIGIT / "_" / "." / "-" )
//	version   = 1*32( ALPHA / DIGIT / "_" / "." / "+" / "-" )
//	comment   = 1*64( printable ASCII minus "(" ")" )
//
// Tokens that fail to parse are dropped. Returns DefaultBotAgent on empty
// input, when every token is rejected, or when the cap forces truncation
// down to nothing.
func SanitizeBotAgent(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultBotAgent
	}

	rawTokens := strings.Fields(trimmed)
	tokens := make([]string, 0, len(rawTokens))
	for i := 0; i < len(rawTokens); i++ {
		tok := rawTokens[i]
		if strings.HasPrefix(tok, "(") && !strings.HasSuffix(tok, ")") {
			acc := tok
			for i+1 < len(rawTokens) && !strings.HasSuffix(acc, ")") {
				i++
				acc += " " + rawTokens[i]
			}
			tokens = append(tokens, acc)
		} else {
			tokens = append(tokens, tok)
		}
	}

	accepted := make([]string, 0, len(tokens))
	pending := ""
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "(") && strings.HasSuffix(tok, ")") {
			inner := tok[1 : len(tok)-1]
			if pending != "" && botAgentCommentRe.MatchString(inner) {
				accepted = append(accepted, pending+" ("+inner+")")
				pending = ""
			} else if pending != "" {
				accepted = append(accepted, pending)
				pending = ""
			}
			continue
		}
		if pending != "" {
			accepted = append(accepted, pending)
			pending = ""
		}
		if botAgentProductRe.MatchString(tok) {
			pending = tok
		}
	}
	if pending != "" {
		accepted = append(accepted, pending)
	}

	if len(accepted) == 0 {
		return DefaultBotAgent
	}

	joined := strings.Join(accepted, " ")
	if utf8.RuneCountInString(joined) <= botAgentMaxLen && len(joined) <= botAgentMaxLen {
		return joined
	}

	truncated := make([]string, 0, len(accepted))
	used := 0
	for _, t := range accepted {
		add := len(t)
		if len(truncated) > 0 {
			add++
		}
		if used+add > botAgentMaxLen {
			break
		}
		truncated = append(truncated, t)
		used += add
	}
	if len(truncated) == 0 {
		return DefaultBotAgent
	}
	return strings.Join(truncated, " ")
}

// NotifyStart announces this channel client is starting.
//
// Endpoint: ilink/bot/msg/notifystart.
func (c *Client) NotifyStart(ctx context.Context) (*NotifyStartResponse, error) {
	req := &NotifyStartRequest{BaseInfo: c.BuildBaseInfo()}
	resp := &NotifyStartResponse{}
	if err := c.doRequestWithTimeout(ctx, "ilink/bot/msg/notifystart", DefaultConfigTimeout, req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// NotifyStop announces this channel client is stopping (gateway shutdown).
//
// Endpoint: ilink/bot/msg/notifystop.
func (c *Client) NotifyStop(ctx context.Context) (*NotifyStopResponse, error) {
	req := &NotifyStopRequest{BaseInfo: c.BuildBaseInfo()}
	resp := &NotifyStopResponse{}
	if err := c.doRequestWithTimeout(ctx, "ilink/bot/msg/notifystop", DefaultConfigTimeout, req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
