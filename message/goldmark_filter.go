// Package message provides message conversion and session utilities.
package message

import (
	"bytes"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	mdtext "github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// ApplyGoldmark filters markdown for WeChat using a goldmark AST plus a custom
// markdown renderer. It is an opt-in alternative to Apply (which uses a
// hand-written state machine); behavior matches Apply on the common constructs.
//
// WeChat renders most markdown natively, so — like Apply — this keeps bold,
// non-CJK italic, code, code fences, tables, rules, H1-H4, links, strikethrough
// and blockquotes verbatim, and only:
//   - strips italic/bold-italic markers wrapping CJK content,
//   - strips H5/H6 markers (text kept),
//   - removes images entirely.
//
// Goldmark is whole-message (not streaming); for incremental input use the
// StreamingMarkdownFilter.
func ApplyGoldmark(text string) string {
	source := []byte(text)
	doc := goldmarkMD.Parser().Parse(mdtext.NewReader(source))

	// Two-pass: drop all image nodes first so the renderer never sees them.
	dropImages(doc)

	bw := &bufferWriter{buf: &bytes.Buffer{}}
	if err := goldmarkMD.Renderer().Render(bw, source, doc); err != nil {
		return text
	}
	return bw.buf.String()
}

// goldmarkMD is the configured markdown instance: GFM table + strikethrough
// parsing, with a custom markdown (not HTML) renderer.
var goldmarkMD = func() goldmark.Markdown {
	r := renderer.NewRenderer(renderer.WithNodeRenderers(
		util.Prioritized(&mdRenderer{}, 1),
	))
	return goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough),
		goldmark.WithRenderer(r),
	)
}()

// bufferWriter implements util.BufWriter over a *bytes.Buffer, giving the
// render funcs direct access to accumulated bytes (needed for blockquote
// line-prefixing on exit).
type bufferWriter struct{ buf *bytes.Buffer }

func (b *bufferWriter) Write(p []byte) (int, error)       { return b.buf.Write(p) }
func (b *bufferWriter) WriteByte(c byte) error            { return b.buf.WriteByte(c) }
func (b *bufferWriter) WriteRune(r rune) (int, error)     { return b.buf.WriteRune(r) }
func (b *bufferWriter) WriteString(s string) (int, error) { return b.buf.WriteString(s) }
func (b *bufferWriter) Available() int                    { return 1 << 30 }
func (b *bufferWriter) Buffered() int                     { return b.buf.Len() }
func (b *bufferWriter) Flush() error                      { return nil }

// dropImages removes every ast.Image node from the tree in a separate pass to
// avoid mutating the tree while ast.Walk iterates it.
func dropImages(n gast.Node) {
	_ = gast.Walk(n, func(node gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering && node.Kind() == gast.KindImage {
			if p := node.Parent(); p != nil {
				p.RemoveChild(p, node)
			}
			return gast.WalkSkipChildren, nil
		}
		return gast.WalkContinue, nil
	})
}

// mdRenderer emits markdown source from a goldmark AST, applying the WeChat
// filtering rules. It implements renderer.NodeRenderer.
type mdRenderer struct{}

// RegisterFuncs wires each node Kind to an enter/exit render function.
func (r *mdRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// Blocks
	reg.Register(gast.KindDocument, r.doNothing)
	reg.Register(gast.KindParagraph, r.doParagraph)
	reg.Register(gast.KindTextBlock, r.doParagraph)
	reg.Register(gast.KindHeading, r.doHeading)
	reg.Register(gast.KindThematicBreak, r.doThematicBreak)
	reg.Register(gast.KindCodeBlock, r.doIndentedCode)
	reg.Register(gast.KindFencedCodeBlock, r.doFencedCode)
	reg.Register(gast.KindBlockquote, r.doBlockquote)
	reg.Register(gast.KindList, r.doList)
	reg.Register(gast.KindListItem, r.doListItem)
	reg.Register(gast.KindHTMLBlock, r.doRawBlock)

	// Inlines
	reg.Register(gast.KindText, r.doText)
	reg.Register(gast.KindCodeSpan, r.doCodeSpan)
	reg.Register(gast.KindEmphasis, r.doEmphasis)
	reg.Register(gast.KindLink, r.doLink)
	reg.Register(gast.KindAutoLink, r.doAutoLink)
	reg.Register(gast.KindRawHTML, r.doRawInline)

	// GFM extension
	reg.Register(extast.KindStrikethrough, r.doStrikethrough)
	reg.Register(extast.KindTable, r.doTable)
	reg.Register(extast.KindTableHeader, r.doTableRow)
	reg.Register(extast.KindTableRow, r.doTableRow)
	reg.Register(extast.KindTableCell, r.doTableCell)
}

// helpers

func bufOf(w util.BufWriter) *bytes.Buffer {
	if b, ok := w.(*bufferWriter); ok {
		return b.buf
	}
	return nil
}

func ensureNewline(w util.BufWriter) gast.WalkStatus {
	if b := bufOf(w); b != nil && b.Len() > 0 && b.Bytes()[b.Len()-1] != '\n' {
		_ = w.WriteByte('\n')
	}
	return gast.WalkContinue
}

func ensureLeadingNewline(w util.BufWriter) gast.WalkStatus {
	if b := bufOf(w); b != nil && b.Len() > 0 {
		last := b.Bytes()[b.Len()-1]
		// Don't insert a newline if we're already at a line start, or right
		// after a list marker / table cell content (trailing space).
		if last != '\n' && last != ' ' {
			_ = w.WriteByte('\n')
		}
	}
	return gast.WalkContinue
}

// ---- block render funcs ----

func (r *mdRenderer) doNothing(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doParagraph(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		return ensureLeadingNewline(w), nil
	}
	if s := ensureNewline(w); s != gast.WalkContinue {
		return s, nil
	}
	_ = w.WriteByte('\n')
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doHeading(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	h := node.(*gast.Heading)
	if entering {
		if s := ensureLeadingNewline(w); s != gast.WalkContinue {
			return s, nil
		}
		if h.Level <= 4 {
			_, _ = w.WriteString(hashRepeat(h.Level))
			_ = w.WriteByte(' ')
		}
		// H5/H6: emit nothing (markers dropped, text kept by children).
		return gast.WalkContinue, nil
	}
	_ = w.WriteByte('\n')
	_ = w.WriteByte('\n')
	return gast.WalkContinue, nil
}

func hashRepeat(level int) string {
	switch level {
	case 1:
		return "#"
	case 2:
		return "##"
	case 3:
		return "###"
	default:
		return "####"
	}
}

func (r *mdRenderer) doThematicBreak(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	if s := ensureLeadingNewline(w); s != gast.WalkContinue {
		return s, nil
	}
	_, _ = w.WriteString("---\n\n")
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doIndentedCode(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	if s := ensureLeadingNewline(w); s != gast.WalkContinue {
		return s, nil
	}
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		_, _ = w.WriteString("    ")
		_, _ = w.Write(source[seg.Start:seg.Stop])
	}
	_ = w.WriteByte('\n')
	_ = w.WriteByte('\n')
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doFencedCode(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	fcb := node.(*gast.FencedCodeBlock)
	if entering {
		if s := ensureLeadingNewline(w); s != gast.WalkContinue {
			return s, nil
		}
		_, _ = w.WriteString("```")
		if lang := fcb.Language(source); len(lang) > 0 {
			_, _ = w.Write(lang)
		}
		_ = w.WriteByte('\n')
		return gast.WalkContinue, nil
	}
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		_, _ = w.Write(source[seg.Start:seg.Stop])
	}
	_, _ = w.WriteString("```\n\n")
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doBlockquote(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		return ensureLeadingNewline(w), nil
	}
	// On exit, prefix every line of the blockquote's content with "> ". The
	// blockquote's children were rendered after the enter call, so their output
	// is the tail of the buffer. Since goldmark strips `>` during parsing, we
	// reconstruct the marker here.
	b := bufOf(w)
	if b == nil {
		return gast.WalkContinue, nil
	}
	out := trimTrailingNewlines(b.String())
	b.Reset()
	for _, line := range splitLines(out) {
		if line == "" {
			_, _ = w.WriteString(">\n")
			continue
		}
		_, _ = w.WriteString("> ")
		_, _ = w.WriteString(line)
		_ = w.WriteByte('\n')
	}
	_ = w.WriteByte('\n')
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doList(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		return ensureLeadingNewline(w), nil
	}
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doListItem(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	if s := ensureLeadingNewline(w); s != gast.WalkContinue {
		return s, nil
	}
	marker := "- "
	if list, ok := node.Parent().(*gast.List); ok && list.IsOrdered() {
		// Number = list start + this item's index among its siblings.
		num := list.Start
		for sib := node.PreviousSibling(); sib != nil; sib = sib.PreviousSibling() {
			num++
		}
		marker = itoa(num) + ". "
	}
	_, _ = w.WriteString(marker)
	return gast.WalkContinue, nil
}

// itoa formats a non-negative int as a decimal string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func (r *mdRenderer) doRawBlock(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		_, _ = w.Write(source[seg.Start:seg.Stop])
	}
	return gast.WalkContinue, nil
}

// ---- inline render funcs ----

func (r *mdRenderer) doText(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	t := node.(*gast.Text)
	_, _ = w.Write(t.Value(source))
	if t.SoftLineBreak() {
		_ = w.WriteByte('\n')
	}
	if t.HardLineBreak() {
		_ = w.WriteByte('\n')
	}
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doCodeSpan(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_ = w.WriteByte('`')
	} else {
		_ = w.WriteByte('`')
	}
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doEmphasis(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	e := node.(*gast.Emphasis)
	// CJK content: strip ITALIC (Level 1) markers — WeChat renders them
	// literally. Bold (Level 2) renders fine for CJK, so keep its markers.
	if e.Level == 1 && containsCJK(textOf(node, source)) {
		return gast.WalkContinue, nil
	}
	marker := "*"
	if e.Level == 2 {
		marker = "**"
	}
	_, _ = w.WriteString(marker)
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doLink(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	l := node.(*gast.Link)
	if entering {
		_ = w.WriteByte('[')
	} else {
		_, _ = w.WriteString("](")
		_, _ = w.Write(l.Destination)
		_ = w.WriteByte(')')
	}
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doAutoLink(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	a := node.(*gast.AutoLink)
	if entering {
		_ = w.WriteByte('<')
	} else {
		_, _ = w.Write(a.URL(source))
		_ = w.WriteByte('>')
	}
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doRawInline(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	seg := node.(*gast.RawHTML)
	for i := 0; i < seg.Segments.Len(); i++ {
		s := seg.Segments.At(i)
		_, _ = w.Write(source[s.Start:s.Stop])
	}
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doStrikethrough(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("~~")
	} else {
		_, _ = w.WriteString("~~")
	}
	return gast.WalkContinue, nil
}

// ---- table render funcs ----

func (r *mdRenderer) doTable(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		return ensureLeadingNewline(w), nil
	}
	_ = w.WriteByte('\n')
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doTableRow(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_ = w.WriteByte('|')
		return gast.WalkContinue, nil
	}
	_ = w.WriteByte('\n')
	// After the header row, emit the alignment separator row.
	if node.Kind() == extast.KindTableHeader && node.Parent() != nil {
		if tbl, ok := node.Parent().(*extast.Table); ok {
			parts := make([]string, 0, len(tbl.Alignments))
			for _, a := range tbl.Alignments {
				switch a {
				case extast.AlignLeft:
					parts = append(parts, ":---")
				case extast.AlignCenter:
					parts = append(parts, ":---:")
				case extast.AlignRight:
					parts = append(parts, "---:")
				default:
					parts = append(parts, "---")
				}
			}
			_ = w.WriteByte('|')
			_, _ = w.WriteString(joinPipe(parts))
			_, _ = w.WriteString("|\n")
		}
	}
	return gast.WalkContinue, nil
}

func (r *mdRenderer) doTableCell(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		// Leading space so cells render as "| a | b |".
		_, _ = w.WriteString(" ")
	} else {
		// Trailing space before the pipe.
		_ = w.WriteByte(' ')
		_ = w.WriteByte('|')
	}
	return gast.WalkContinue, nil
}

// textOf returns the concatenated text content of a node's subtree, for CJK
// detection on emphasis nodes.
func textOf(n gast.Node, source []byte) string {
	var b []byte
	_ = gast.Walk(n, func(node gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}
		if t, ok := node.(*gast.Text); ok {
			b = append(b, t.Value(source)...)
		}
		return gast.WalkContinue, nil
	})
	return string(b)
}

func joinPipe(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "|"
		}
		out += p
	}
	return out
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func trimTrailingNewlines(s string) string {
	end := len(s)
	for end > 0 && (s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[:end]
}
