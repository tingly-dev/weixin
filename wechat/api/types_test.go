package api

import (
	"encoding/json"
	"testing"
)

// TestWeixinMessage_LosslessUint64ID verifies that message_id/msg_id/svr_id
// round-trip through encoding/json without precision loss. These fields may
// arrive either as raw uint64 JSON numbers or as prefixed strings (e.g.
// "v1:<digits>"), mirroring the upstream plugin's LOSSLESS_ID_FIELDS handling.
func TestWeixinMessage_LosslessUint64ID(t *testing.T) {
	const maxUint64 = "18446744073709551615" // 2^64 - 1

	raw := []byte(`{
		"message_id": ` + maxUint64 + `,
		"item_list": [{
			"msg_id": ` + maxUint64 + `,
			"ref_msg": {"svr_id": ` + maxUint64 + `}
		}]
	}`)

	var msg WeixinMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got := msg.MessageID; got != LosslessID(maxUint64) {
		t.Fatalf("MessageID = %s, want max uint64", got)
	}
	if len(msg.ItemList) != 1 {
		t.Fatalf("expected 1 item, got %d", len(msg.ItemList))
	}
	item := msg.ItemList[0]
	if got := item.MsgID; got != LosslessID(maxUint64) {
		t.Fatalf("MsgID = %s, want max uint64", got)
	}
	if item.RefMsg == nil {
		t.Fatal("expected RefMsg to be set")
	}
	if got := item.RefMsg.SvrID; got != LosslessID(maxUint64) {
		t.Fatalf("RefMsg.SvrID = %s, want max uint64", got)
	}
}

// TestWeixinMessage_StringPrefixedIDs verifies the newer wire encoding where
// msg_id arrives as a prefixed string like "v1:15492892230605430853"
// (observed live on ilinkai.weixin.qq.com; official types.ts declares
// msg_id?: string). A uint64 field would fail the whole GetUpdates unmarshal
// and silently drop inbound messages.
func TestWeixinMessage_StringPrefixedIDs(t *testing.T) {
	raw := []byte(`{
		"message_id": 7507094999942111240,
		"item_list": [{
			"type": 1,
			"msg_id": "v1:15492892230605430853",
			"text_item": {"text": "hi"}
		}]
	}`)

	var msg WeixinMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got := msg.MessageID; got != "7507094999942111240" {
		t.Fatalf("MessageID = %s, want 7507094999942111240", got)
	}
	if got := msg.ItemList[0].MsgID; got != "v1:15492892230605430853" {
		t.Fatalf("MsgID = %s, want v1:15492892230605430853", got)
	}
}

func TestRefMessage_PartialTextRoundTrip(t *testing.T) {
	raw := []byte(`{
		"ref_msg": {
			"svr_id": 12345,
			"partial_text": {
				"start": "hello",
				"end": "world",
				"startindex": 0,
				"endindex": 1,
				"quotemd5": "abc123"
			}
		}
	}`)

	var item MessageItem
	if err := json.Unmarshal(raw, &item); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if item.RefMsg == nil || item.RefMsg.PartialText == nil {
		t.Fatal("expected ref_msg.partial_text to be populated")
	}
	pt := item.RefMsg.PartialText
	if pt.Start != "hello" || pt.End != "world" || pt.QuoteMD5 != "abc123" {
		t.Fatalf("partial_text not round-tripped correctly: %+v", pt)
	}
}
