package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a Client pointed at a test server whose handler is h.
func newTestClient(t *testing.T, h http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "test-token")
}

func TestSendMessage_ZeroRet_Succeeds(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ret":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	err := c.SendMessage(context.Background(), "user-1", SendOptions{ContextToken: "ctx", RunID: "run-1"},
		[]MessageItem{{Type: MessageItemTypeText, TextItem: &TextItem{Text: "hi"}}})
	if err != nil {
		t.Fatalf("expected nil error for ret=0, got %v", err)
	}
	if !strings.HasSuffix(gotPath, "ilink/bot/sendmessage") {
		t.Fatalf("unexpected path %q", gotPath)
	}
}

func TestSendMessage_NonZeroRet_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ret":-14,"errmsg":"session timeout"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	err := c.SendMessage(context.Background(), "user-1", SendOptions{}, []MessageItem{})
	if err == nil {
		t.Fatal("expected error for non-zero ret, got nil")
	}
	// Must surface both ret and errmsg so failures are diagnosable.
	if !strings.Contains(err.Error(), "ret=-14") {
		t.Fatalf("error %q missing ret=-14", err.Error())
	}
	if !strings.Contains(err.Error(), "session timeout") {
		t.Fatalf("error %q missing errmsg", err.Error())
	}
}

func TestSendMessage_NonZeroRet_NoErrMsg_Fallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ret":-1}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	err := c.SendMessage(context.Background(), "user-1", SendOptions{}, []MessageItem{})
	if err == nil || !strings.Contains(err.Error(), "(none)") {
		t.Fatalf("expected (none) fallback, got %v", err)
	}
}

func TestSendMessage_RunIDAndContextToken_OnWire(t *testing.T) {
	var captured SendMessageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_, _ = w.Write([]byte(`{"ret":0}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	err := c.SendMessage(context.Background(), "user-1",
		SendOptions{ContextToken: "ctx-token", RunID: "run-abc"},
		[]MessageItem{{Type: MessageItemTypeText, TextItem: &TextItem{Text: "hi"}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Msg == nil {
		t.Fatal("missing msg in request")
	}
	if captured.Msg.RunID != "run-abc" {
		t.Fatalf("run_id = %q, want run-abc", captured.Msg.RunID)
	}
	if captured.Msg.ContextToken != "ctx-token" {
		t.Fatalf("context_token = %q, want ctx-token", captured.Msg.ContextToken)
	}
}

func TestSendMessage_OmitsEmptyRunID(t *testing.T) {
	raw, err := json.Marshal(WeixinMessageWrapper{
		FromUserID:   "bot",
		ToUserID:     "user",
		ClientID:     "c",
		MessageType:  MessageTypeBot,
		MessageState: MessageStateFinish,
		ContextToken: "ctx",
		ItemList:     []MessageItem{{Type: MessageItemTypeText, TextItem: &TextItem{Text: "hi"}}},
		// RunID intentionally unset
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "run_id") {
		t.Fatalf("run_id should be omitted when empty, got %s", string(raw))
	}
}

func TestSendMessageItem_ToolCallStartRoundTrip(t *testing.T) {
	item := MessageItem{
		Type:             MessageItemTypeToolCallStart,
		ToolCallStartItem: &ToolCallStartItem{ToolName: "search", ToolCallID: "call-1"},
	}
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(raw)
	for _, want := range []string{`"type":11`, `"tool_call_start_item"`, `"tool_name":"search"`, `"tool_call_id":"call-1"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("marshaled JSON missing %q\n got: %s", want, got)
		}
	}

	// Round-trip back.
	var back MessageItem
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Type != MessageItemTypeToolCallStart {
		t.Fatalf("type = %d, want %d", back.Type, MessageItemTypeToolCallStart)
	}
	if back.ToolCallStartItem == nil || back.ToolCallStartItem.ToolName != "search" {
		t.Fatalf("tool_call_start_item not round-tripped: %+v", back.ToolCallStartItem)
	}
}

func TestSendMessageItem_ToolCallResultRoundTrip(t *testing.T) {
	item := MessageItem{
		Type:              MessageItemTypeToolCallResult,
		ToolCallResultItem: &ToolCallResultItem{ToolName: "search", ToolCallID: "call-1", Status: "completed"},
	}
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(raw)
	for _, want := range []string{`"type":12`, `"tool_call_result_item"`, `"status":"completed"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("marshaled JSON missing %q\n got: %s", want, got)
		}
	}
}

func TestSendMessageItem_DelegatesAndValidates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ret":-1,"errmsg":"bad"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	id, err := c.SendMessageItem(context.Background(), "user-1", SendOptions{},
		MessageItem{Type: MessageItemTypeToolCallStart, ToolCallStartItem: &ToolCallStartItem{}})
	if err == nil {
		t.Fatal("expected error from non-zero ret")
	}
	if id != "" {
		t.Fatalf("expected empty id, got %q", id)
	}
}

func TestClassifyFetchError_TableDriven(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want FetchErrorType
	}{
		{
			name: "nil",
			err:  nil,
			want: "", // ClassifyFetchError(nil) == nil
		},
		{
			name: "context deadline",
			err:  context.DeadlineExceeded,
			want: FetchErrTimeout,
		},
		{
			name: "context canceled (external abort)",
			err:  context.Canceled,
			want: FetchErrTimeout,
		},
		{
			name: "dns error",
			err:  &net.DNSError{Err: "no such host", Name: "example.test", IsNotFound: true},
			want: FetchErrDNS,
		},
		{
			name: "tcp connection refused",
			err:  &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			want: FetchErrTCP,
		},
		{
			name: "tls handshake error",
			err:  &net.OpError{Op: "remote error", Net: "tcp", Err: errors.New("tls: handshake failure")},
			want: FetchErrTLS,
		},
		{
			name: "opaque tls string",
			err:  errors.New("x509: certificate signed by unknown authority"),
			want: FetchErrTLS,
		},
		{
			name: "opaque tcp string",
			err:  errors.New("dial tcp: connect: connection refused"),
			want: FetchErrTCP,
		},
		{
			name: "unknown",
			err:  errors.New("something completely different"),
			want: FetchErrUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyFetchError(tc.err)
			if tc.err == nil {
				if got != nil {
					t.Fatalf("expected nil for nil err, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil FetchError")
			}
			if got.Type != tc.want {
				t.Fatalf("type = %s, want %s (err=%v)", got.Type, tc.want, tc.err)
			}
			// Unwrap must expose the original for errors.Is/As.
			if !errors.Is(got, tc.err) && got.Err != tc.err {
				// For wrapped errors like context.DeadlineExceeded, errors.Is should hold;
				// otherwise the raw pointer must match.
				if !errors.Is(got, tc.err) {
					t.Fatalf("Unwrap broke errors.Is: %v", got)
				}
			}
		})
	}
}

func TestQRStatus_Constants(t *testing.T) {
	if QRStatusBindedRedirect != "binded_redirect" {
		t.Fatalf("QRStatusBindedRedirect = %q", QRStatusBindedRedirect)
	}
	if QRStatusConfirmed != "confirmed" {
		t.Fatalf("QRStatusConfirmed = %q", QRStatusConfirmed)
	}
	if MessageItemTypeToolCallStart != 11 || MessageItemTypeToolCallResult != 12 {
		t.Fatalf("tool call type ids: start=%d result=%d, want 11/12",
			MessageItemTypeToolCallStart, MessageItemTypeToolCallResult)
	}
}
