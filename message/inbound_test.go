package message

import (
	"testing"

	"github.com/tingly-dev/weixin/wechat/api"
)

func TestConvertInboundMessage_NoQuote_LeavesReplyFieldsEmpty(t *testing.T) {
	msg := &api.WeixinMessage{
		MessageID:  1,
		FromUserID: "user-1",
		ToUserID:   "bot-1",
		ItemList: []api.MessageItem{
			{Type: api.MessageItemTypeText, TextItem: &api.TextItem{Text: "hello"}},
		},
	}

	got := ConvertInboundMessage(msg, "acct-1", "")
	if got.ReplyToID != "" {
		t.Fatalf("ReplyToID = %q, want empty", got.ReplyToID)
	}
	if got.ReplyToBody != "" {
		t.Fatalf("ReplyToBody = %q, want empty", got.ReplyToBody)
	}
	if v, ok := got.Metadata["reply_to_is_quote"]; ok {
		t.Fatalf("reply_to_is_quote should be absent, got %v", v)
	}
}

func TestConvertInboundMessage_InlineQuote_ResolvesBody(t *testing.T) {
	msg := &api.WeixinMessage{
		MessageID:  2,
		FromUserID: "user-1",
		ToUserID:   "bot-1",
		ItemList: []api.MessageItem{
			{
				Type:     api.MessageItemTypeText,
				TextItem: &api.TextItem{Text: "sure, sounds good"},
				RefMsg: &api.RefMessage{
					SvrID: 42,
					MessageItem: &api.MessageItem{
						TextItem: &api.TextItem{Text: "want to grab lunch?"},
					},
				},
			},
		},
	}

	got := ConvertInboundMessage(msg, "acct-1", "")
	if got.ReplyToID != "42" {
		t.Fatalf("ReplyToID = %q, want 42", got.ReplyToID)
	}
	if got.ReplyToBody != "want to grab lunch?" {
		t.Fatalf("ReplyToBody = %q, want inline quoted text", got.ReplyToBody)
	}
	if got.Metadata["reply_to_is_quote"] != true {
		t.Fatalf("reply_to_is_quote = %v, want true", got.Metadata["reply_to_is_quote"])
	}
}

func TestConvertInboundMessage_IDOnlyQuote_LeavesBodyUnresolved(t *testing.T) {
	msg := &api.WeixinMessage{
		MessageID:  3,
		FromUserID: "user-1",
		ToUserID:   "bot-1",
		ItemList: []api.MessageItem{
			{
				Type:     api.MessageItemTypeText,
				TextItem: &api.TextItem{Text: "yes!"},
				RefMsg: &api.RefMessage{
					SvrID: 99,
					// No inline MessageItem/Title: newer WeChat clients send
					// ID-only quotes that this SDK doesn't resolve.
				},
			},
		},
	}

	got := ConvertInboundMessage(msg, "acct-1", "")
	if got.ReplyToID != "99" {
		t.Fatalf("ReplyToID = %q, want 99", got.ReplyToID)
	}
	if got.ReplyToBody != "" {
		t.Fatalf("ReplyToBody = %q, want empty for an unresolved ID-only quote", got.ReplyToBody)
	}
	if got.Metadata["reply_to_is_quote"] != true {
		t.Fatalf("reply_to_is_quote = %v, want true", got.Metadata["reply_to_is_quote"])
	}
}

func TestConvertInboundMessage_QuoteFallsBackToNestedMsgID(t *testing.T) {
	msg := &api.WeixinMessage{
		MessageID:  4,
		FromUserID: "user-1",
		ToUserID:   "bot-1",
		ItemList: []api.MessageItem{
			{
				Type:     api.MessageItemTypeText,
				TextItem: &api.TextItem{Text: "ok"},
				RefMsg: &api.RefMessage{
					// SvrID absent (older-style quote): fall back to the
					// quoted item's own msg_id.
					MessageItem: &api.MessageItem{
						MsgID:    7,
						TextItem: &api.TextItem{Text: "original message"},
					},
				},
			},
		},
	}

	got := ConvertInboundMessage(msg, "acct-1", "")
	if got.ReplyToID != "7" {
		t.Fatalf("ReplyToID = %q, want 7 (fallback to message_item.msg_id)", got.ReplyToID)
	}
	if got.ReplyToBody != "original message" {
		t.Fatalf("ReplyToBody = %q, want inline text", got.ReplyToBody)
	}
}
