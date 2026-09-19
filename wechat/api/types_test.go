package api

import (
	"encoding/json"
	"testing"
)

// TestWeixinMessage_LosslessUint64ID verifies that message_id/msg_id/svr_id
// round-trip through encoding/json without precision loss, since they are
// uint64 on the wire. The upstream JS plugin has to string-quote these
// fields before JSON.parse to avoid float64 precision loss above 2^53; Go's
// encoding/json needs no such workaround when the field type is uint64.
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

	if got := msg.MessageID; got != 18446744073709551615 {
		t.Fatalf("MessageID = %d, want max uint64", got)
	}
	if len(msg.ItemList) != 1 {
		t.Fatalf("expected 1 item, got %d", len(msg.ItemList))
	}
	item := msg.ItemList[0]
	if got := item.MsgID; got != 18446744073709551615 {
		t.Fatalf("MsgID = %d, want max uint64", got)
	}
	if item.RefMsg == nil {
		t.Fatal("expected RefMsg to be set")
	}
	if got := item.RefMsg.SvrID; got != 18446744073709551615 {
		t.Fatalf("RefMsg.SvrID = %d, want max uint64", got)
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
