package message

// ToPlainText filters markdown for WeChat.
//
// Deprecated: WeChat renders markdown natively; use Apply (or a
// StreamingMarkdownFilter) instead, which preserves the constructs WeChat
// renders well and only strips the subset that renders poorly. This alias
// is retained so existing callers keep compiling.
func ToPlainText(text string) string {
	return Apply(text)
}
