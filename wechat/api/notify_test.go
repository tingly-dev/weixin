package api

import (
	"strings"
	"testing"
)

func TestSanitizeBotAgent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", DefaultBotAgent},
		{"whitespace", "   ", DefaultBotAgent},
		{"single product", "MyApp/1.0.0", "MyApp/1.0.0"},
		{"product with comment", "MyApp/1.0.0 (linux)", "MyApp/1.0.0 (linux)"},
		{"two products", "MyApp/1.0.0 Helper/2.3", "MyApp/1.0.0 Helper/2.3"},
		{"reject bad token", "MyApp/1.0.0 not_a_product", "MyApp/1.0.0"},
		{"reject all tokens", "garbage tokens here", DefaultBotAgent},
		{"comment with spaces", "MyApp/1.0.0 (build 42 linux)", "MyApp/1.0.0 (build 42 linux)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeBotAgent(tc.in)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSanitizeBotAgentTruncates(t *testing.T) {
	long := strings.Repeat("Token/1.0 ", 40) // 400 chars
	got := SanitizeBotAgent(long)
	if len(got) > botAgentMaxLen {
		t.Fatalf("len %d exceeds cap %d", len(got), botAgentMaxLen)
	}
	if got == "" || got == DefaultBotAgent {
		t.Fatalf("expected truncated value, got %q", got)
	}
}

func TestClientBuildBaseInfo(t *testing.T) {
	c := NewClient("https://example.test", "tok")
	c.SetBotAgent("MyApp/1.0.0")
	bi := c.BuildBaseInfo()
	if bi.ChannelVersion != SDKVersion {
		t.Fatalf("channel_version = %q, want %q", bi.ChannelVersion, SDKVersion)
	}
	if bi.BotAgent != "MyApp/1.0.0" {
		t.Fatalf("bot_agent = %q, want MyApp/1.0.0", bi.BotAgent)
	}
}
