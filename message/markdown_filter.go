// Package message provides message conversion and session utilities.
package message

import "regexp"

// StreamingMarkdownFilter is a character-level state machine that filters
// markdown for the WeChat client.
//
// WeChat renders most markdown natively (bold, italic, inline code, code
// fences, tables, horizontal rules, H1–H4), so the filter keeps those
// constructs intact and only strips the subset that WeChat renders poorly.
// This mirrors @tencent-weixin/openclaw-weixin's StreamingMarkdownFilter
// (src/messaging/markdown-filter.ts).
//
// Pass-through (kept verbatim):
//   - Code fences (```)
//   - Inline code (`)
//   - Tables (| ... |)
//   - Horizontal rules (---, ***, ___)
//   - Bold (**, __) and bold-italic (***)
//   - Italic/bold-italic wrapping non-CJK content
//
// Filtered (markers stripped, content kept):
//   - Italic/bold-italic wrapping CJK content (renders with literal markers)
//   - Headings H5/H6 (#####, ######) — unsupported
//   - Blockquote marker ">" at line start
//   - Leading whitespace (indentation renders as a code/monospace mess)
//   - Images (![alt](url)) — removed entirely
//
// Use Feed for incremental (streaming) input and Flush at end-of-stream, or
// Apply for a complete message in one call.
type StreamingMarkdownFilter struct {
	buf   string
	fence bool
	// sol: start-of-line. Defaults to true on first use; Go zero-value is
	// false, so we track "initialized" via started and set sol=true once.
	sol     bool
	started bool
	inl     *inlineAcc
}

// inlineAcc accumulates content inside an inline marker pair until the
// closing marker is found.
type inlineAcc struct {
	kind string // "image" | "bold3" | "italic" | "ubold3" | "uitalic"
	acc  string
}

// ensureStarted sets the start-of-line flag on first use.
func (f *StreamingMarkdownFilter) ensureStarted() {
	if !f.started {
		f.started = true
		f.sol = true
	}
}

// Feed appends delta to the buffer and returns as much filtered output as
// can be emitted immediately. Only the minimum characters needed for pattern
// disambiguation are held back.
func (f *StreamingMarkdownFilter) Feed(delta string) string {
	f.ensureStarted()
	f.buf += delta
	return f.pump(false)
}

// Flush drains the buffer assuming end-of-stream and returns the remaining
// filtered output.
func (f *StreamingMarkdownFilter) Flush() string {
	f.ensureStarted()
	return f.pump(true)
}

// Apply filters a complete markdown string in one pass. It is a convenience
// wrapper around Feed+Flush for non-streaming callers.
func Apply(text string) string {
	var f StreamingMarkdownFilter
	return f.Feed(text) + f.Flush()
}

func (f *StreamingMarkdownFilter) pump(eof bool) string {
	out := ""
	for f.buf != "" {
		sLen := len(f.buf)
		sSol := f.sol
		sFence := f.fence
		sInl := f.inl

		if f.fence {
			out += f.pumpFence(eof)
		} else if f.inl != nil {
			out += f.pumpInline(eof)
		} else if f.sol {
			out += f.pumpSOL(eof)
		} else {
			out += f.pumpBody(eof)
		}

		if len(f.buf) == sLen && f.sol == sSol && f.fence == sFence && f.inl == sInl {
			break
		}
	}

	if eof && f.inl != nil {
		out += inlineOpenMarker(f.inl.kind) + f.inl.acc
		f.inl = nil
	}
	return out
}

func inlineOpenMarker(kind string) string {
	switch kind {
	case "image":
		return "!["
	case "bold3":
		return "***"
	case "italic":
		return "*"
	case "ubold3":
		return "___"
	case "uitalic":
		return "_"
	}
	return ""
}

// pumpFence passes content and markers through verbatim until the closing ```.
func (f *StreamingMarkdownFilter) pumpFence(eof bool) string {
	if f.sol {
		if len(f.buf) < 3 && !eof {
			return ""
		}
		if stringsHasPrefixSafe(f.buf, "```") {
			nl := stringsIndexFrom(f.buf, "\n", 3)
			if nl != -1 {
				f.fence = false
				line := f.buf[:nl+1]
				f.buf = f.buf[nl+1:]
				f.sol = true
				return line
			}
			if eof {
				f.fence = false
				line := f.buf
				f.buf = ""
				return line
			}
			return ""
		}
		f.sol = false
	}
	nl := stringsIndexByte(f.buf, '\n')
	if nl != -1 {
		chunk := f.buf[:nl+1]
		f.buf = f.buf[nl+1:]
		f.sol = true
		return chunk
	}
	chunk := f.buf
	f.buf = ""
	return chunk
}

// pumpSOL detects and consumes line-start patterns, then transitions to body.
func (f *StreamingMarkdownFilter) pumpSOL(eof bool) string {
	b := f.buf
	if b == "" {
		return ""
	}

	// Blank line.
	if b[0] == '\n' {
		f.buf = b[1:]
		return "\n"
	}

	// Code fence open.
	if b[0] == '`' {
		if len(b) < 3 && !eof {
			return ""
		}
		if stringsHasPrefixSafe(b, "```") {
			nl := stringsIndexFrom(b, "\n", 3)
			if nl != -1 {
				f.fence = true
				line := b[:nl+1]
				f.buf = b[nl+1:]
				f.sol = true
				return line
			}
			if eof {
				f.buf = ""
				return b
			}
			return ""
		}
		f.sol = false
		return ""
	}

	// Blockquote marker — strip.
	if b[0] == '>' {
		f.sol = false
		return ""
	}

	// Headings.
	if b[0] == '#' {
		n := 0
		for n < len(b) && b[n] == '#' {
			n++
		}
		if n == len(b) && !eof {
			return ""
		}
		// H5/H6 unsupported: strip the markers, keep the text.
		if n >= 5 && n <= 6 && n < len(b) && b[n] == ' ' {
			f.buf = b[n+1:]
			f.sol = false
			return ""
		}
		// H1–H4: keep the markers verbatim.
		f.sol = false
		return ""
	}

	// Leading whitespace — strip (indentation renders as monospace mess).
	if b[0] == ' ' || b[0] == '\t' {
		if !eof && onlyWhitespaceOrTab(b) {
			return ""
		}
		f.sol = false
		return ""
	}

	// Horizontal rules and standalone -/*/_, handled below; otherwise body.
	if b[0] == '-' || b[0] == '*' || b[0] == '_' {
		ch := b[0]
		j := 0
		for j < len(b) && (b[j] == ch || b[j] == ' ') {
			j++
		}
		if j == len(b) && !eof {
			return ""
		}
		// Rule iff the run ends at EOL/EOF with >= 3 marker chars.
		if j == len(b) || b[j] == '\n' {
			count := 0
			for k := 0; k < j; k++ {
				if b[k] == ch {
					count++
				}
			}
			if count >= 3 {
				if j < len(b) {
					f.buf = b[j+1:]
					f.sol = true
					return b[:j+1]
				}
				f.buf = ""
				return b
			}
		}
		f.sol = false
		return ""
	}

	f.sol = false
	return ""
}

// pumpBody scans the line body for inline triggers, emitting safe chars eagerly.
func (f *StreamingMarkdownFilter) pumpBody(eof bool) string {
	out := ""
	i := 0
	for i < len(f.buf) {
		c := f.buf[i]
		if c == '\n' {
			out += f.buf[:i+1]
			f.buf = f.buf[i+1:]
			f.sol = true
			return out
		}
		if c == '!' && i+1 < len(f.buf) && f.buf[i+1] == '[' {
			out += f.buf[:i]
			f.buf = f.buf[i+2:]
			f.inl = &inlineAcc{kind: "image"}
			return out
		}
		if c == '~' { // strikethrough ~~ — pass through verbatim
			i++
			continue
		}
		if c == '*' {
			if i+2 < len(f.buf) && f.buf[i+1] == '*' && f.buf[i+2] == '*' {
				out += f.buf[:i]
				f.buf = f.buf[i+3:]
				f.inl = &inlineAcc{kind: "bold3"}
				return out
			}
			if i+1 < len(f.buf) && f.buf[i+1] == '*' {
				i += 2
				continue
			}
			if i+1 < len(f.buf) && f.buf[i+1] != ' ' && f.buf[i+1] != '\n' {
				out += f.buf[:i]
				f.buf = f.buf[i+1:]
				f.inl = &inlineAcc{kind: "italic"}
				return out
			}
			i++
			continue
		}
		if c == '_' {
			if i+2 < len(f.buf) && f.buf[i+1] == '_' && f.buf[i+2] == '_' {
				out += f.buf[:i]
				f.buf = f.buf[i+3:]
				f.inl = &inlineAcc{kind: "ubold3"}
				return out
			}
			if i+1 < len(f.buf) && f.buf[i+1] == '_' {
				i += 2
				continue
			}
			if i+1 < len(f.buf) && f.buf[i+1] != ' ' && f.buf[i+1] != '\n' {
				out += f.buf[:i]
				f.buf = f.buf[i+1:]
				f.inl = &inlineAcc{kind: "uitalic"}
				return out
			}
			i++
			continue
		}
		i++
	}

	// Hold back trailing chars that could be the start of a marker.
	hold := 0
	if !eof {
		switch {
		case stringsHasSuffixSafe(f.buf, "**"):
			hold = 2
		case stringsHasSuffixSafe(f.buf, "__"):
			hold = 2
		case stringsHasSuffixSafe(f.buf, "*"):
			hold = 1
		case stringsHasSuffixSafe(f.buf, "_"):
			hold = 1
		case stringsHasSuffixSafe(f.buf, "!"):
			hold = 1
		}
	}
	out += f.buf[:len(f.buf)-hold]
	if hold > 0 {
		f.buf = f.buf[len(f.buf)-hold:]
	} else {
		f.buf = ""
	}
	return out
}

// pumpInline accumulates inline content until the closing marker is found.
func (f *StreamingMarkdownFilter) pumpInline(_ bool) string {
	if f.inl == nil {
		return ""
	}
	f.inl.acc += f.buf
	f.buf = ""

	switch f.inl.kind {
	case "bold3", "ubold3":
		marker := "***"
		if f.inl.kind == "ubold3" {
			marker = "___"
		}
		idx := stringsIndex(f.inl.acc, marker)
		if idx != -1 {
			content := f.inl.acc[:idx]
			f.buf = f.inl.acc[idx+3:]
			f.inl = nil
			if containsCJK(content) {
				return content // CJK: drop markers
			}
			return marker + content + marker // non-CJK: keep markers
		}
		return ""

	case "italic", "uitalic":
		marker := byte('*')
		if f.inl.kind == "uitalic" {
			marker = '_'
		}
		for j := 0; j < len(f.inl.acc); j++ {
			ch := f.inl.acc[j]
			if ch == '\n' {
				// Unterminated at line end: restore the opening marker.
				r := string(marker) + f.inl.acc[:j+1]
				f.buf = f.inl.acc[j+1:]
				f.inl = nil
				f.sol = true
				return r
			}
			if ch == marker {
				if j+1 < len(f.inl.acc) && f.inl.acc[j+1] == marker {
					j++ // skip doubled marker
					continue
				}
				content := f.inl.acc[:j]
				f.buf = f.inl.acc[j+1:]
				f.inl = nil
				if containsCJK(content) {
					return content
				}
				return string(marker) + content + string(marker)
			}
		}
		return ""

	case "image":
		cb := stringsIndexByte(f.inl.acc, ']')
		if cb == -1 {
			return ""
		}
		if cb+1 >= len(f.inl.acc) {
			return ""
		}
		if f.inl.acc[cb+1] != '(' {
			r := "![" + f.inl.acc[:cb+1]
			f.buf = f.inl.acc[cb+1:]
			f.inl = nil
			return r
		}
		cp := stringsIndexFrom(f.inl.acc, ")", cb+2)
		if cp != -1 {
			f.buf = f.inl.acc[cp+1:]
			f.inl = nil
			return "" // drop the image entirely
		}
		return ""
	}
	return ""
}

// containsCJK reports whether text contains CJK ideographs / Hangul.
// Used to decide whether to strip italic/bold-italic markers: WeChat renders
// literal markers around CJK content, so we drop them for CJK and keep them
// otherwise.
func containsCJK(text string) bool {
	return cjkRegex.MatchString(text)
}

var cjkRegex = regexp.MustCompile(`[\x{2E80}-\x{9FFF}\x{AC00}-\x{D7AF}\x{F900}-\x{FAFF}]`)

// --- small string helpers (avoid importing strings just for these) ---

func stringsHasPrefixSafe(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func stringsHasSuffixSafe(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func stringsIndexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func stringsIndex(s, sub string) int {
	return stringsIndexFrom(s, sub, 0)
}

func stringsIndexFrom(s, sub string, from int) int {
	if from < 0 {
		from = 0
	}
	if len(sub) == 0 {
		if from <= len(s) {
			return from
		}
		return -1
	}
	for i := from; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func onlyWhitespaceOrTab(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return false
		}
	}
	return true
}
