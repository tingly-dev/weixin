package wecom

import (
	"encoding/json"
	"testing"
)

func TestConvertToChannelMessage_Quote_DoesNotFakeReplyToID(t *testing.T) {
	msg := &IncomingMessage{
		MsgID:   "msg-1",
		From:    MsgFrom{UserID: "user-1"},
		MsgType: MsgTypeText,
		Text:    &TextContent{Content: "sure"},
		Quote: &MsgQuote{
			MsgType: MsgTypeText,
			Text:    &TextContent{Content: "want to grab lunch?"},
		},
	}

	got := convertToChannelMessage(msg, "req-1")

	// WeCom's quote protocol carries no ID for the quoted message; setting
	// ReplyToID to the current message's own ID would be actively misleading.
	if got.ReplyToID != "" {
		t.Fatalf("ReplyToID = %q, want empty (WeCom quotes have no id)", got.ReplyToID)
	}
	if got.Metadata["reply_to_is_quote"] != true {
		t.Fatalf("reply_to_is_quote = %v, want true", got.Metadata["reply_to_is_quote"])
	}
	quote, ok := got.Metadata["quote"].(map[string]interface{})
	if !ok {
		t.Fatalf("quote metadata missing or wrong type: %v", got.Metadata["quote"])
	}
	if quote["text"] != "want to grab lunch?" {
		t.Fatalf("quote text = %v, want inline quoted text", quote["text"])
	}
}

func TestConvertToChannelMessage_NoQuote_LeavesReplyFieldsEmpty(t *testing.T) {
	msg := &IncomingMessage{
		MsgID:   "msg-2",
		From:    MsgFrom{UserID: "user-1"},
		MsgType: MsgTypeText,
		Text:    &TextContent{Content: "hello"},
	}

	got := convertToChannelMessage(msg, "req-2")
	if got.ReplyToID != "" {
		t.Fatalf("ReplyToID = %q, want empty", got.ReplyToID)
	}
	if _, ok := got.Metadata["reply_to_is_quote"]; ok {
		t.Fatalf("reply_to_is_quote should be absent, got %v", got.Metadata["reply_to_is_quote"])
	}
}

func TestMsgQuote_VideoField_RoundTrips(t *testing.T) {
	// Regression test: MsgQuote previously had no Video field, so a quoted
	// video message's url/aeskey were silently dropped by JSON unmarshal.
	raw := []byte(`{
		"msgtype": "video",
		"video": {"url": "https://example.test/v.mp4", "aeskey": "abc123"}
	}`)

	var quote MsgQuote
	if err := json.Unmarshal(raw, &quote); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if quote.Video == nil {
		t.Fatal("expected Video to be populated")
	}
	if quote.Video.URL != "https://example.test/v.mp4" || quote.Video.AESKey != "abc123" {
		t.Fatalf("Video = %+v, want populated url/aeskey", quote.Video)
	}
}
