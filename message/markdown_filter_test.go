package message

import (
	"strings"
	"testing"
)

func TestApply_PreservesMarkdown(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // expected exact output
	}{
		{
			name: "bold kept",
			in:   "**bold**",
			want: "**bold**",
		},
		{
			name: "italic non-CJK kept",
			in:   "*italic*",
			want: "*italic*",
		},
		{
			name: "inline code kept",
			in:   "use `code` here",
			want: "use `code` here",
		},
		{
			name: "code fence kept verbatim",
			in:   "```go\nfmt.Println(\"hi\")\n```",
			want: "```go\nfmt.Println(\"hi\")\n```",
		},
		{
			name: "table kept",
			in:   "| a | b |\n|---|---|\n| 1 | 2 |",
			want: "| a | b |\n|---|---|\n| 1 | 2 |",
		},
		{
			name: "horizontal rule kept",
			in:   "above\n---\nbelow",
			want: "above\n---\nbelow",
		},
		{
			name: "H1..H4 markers kept",
			in:   "# H1\n## H2\n### H3\n#### H4",
			want: "# H1\n## H2\n### H3\n#### H4",
		},
		{
			name: "link kept as-is",
			in:   "[text](https://example.com)",
			want: "[text](https://example.com)",
		},
		{
			name: "strikethrough kept",
			in:   "~~deleted~~",
			want: "~~deleted~~",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Apply(tc.in)
			if got != tc.want {
				t.Fatalf("Apply(%q)\n got: %q\nwant: %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestApply_StripsBadMarkdown(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		contains string // output must contain this
		omits    string // output must NOT contain this
	}{
		{
			name:     "CJK italic markers stripped",
			in:       "*中文斜体*",
			contains: "中文斜体",
			omits:    "*",
		},
		{
			name:     "H5 markers stripped, text kept",
			in:       "##### Heading Five",
			contains: "Heading Five",
			omits:    "#####",
		},
		{
			name:     "H6 markers stripped, text kept",
			in:       "###### Heading Six",
			contains: "Heading Six",
			omits:    "######",
		},
		{
			name:     "image removed entirely",
			in:       "before ![alt](https://x/y.png) after",
			contains: "before",
			omits:    "![alt](https://x/y.png)",
		},
		// Note: blockquote ">" and leading indentation are passed through
		// verbatim by upstream openclaw-weixin, so we match that behavior.
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Apply(tc.in)
			if tc.contains != "" && !strings.Contains(got, tc.contains) {
				t.Fatalf("output missing %q\n got: %q", tc.contains, got)
			}
			if tc.omits != "" && strings.Contains(got, tc.omits) {
				t.Fatalf("output should not contain %q\n got: %q", tc.omits, got)
			}
		})
	}
}

func TestApply_FullShowcase(t *testing.T) {
	// A realistic mixed document: the filtered output should still contain
	// rich markdown (not be flattened to plain text).
	in := `# Title
## Subtitle

**bold** and *italic* and ` + "`code`" + `

> a quote

1. one
2. two

| A | B |
|---|---|
| 1 | 2 |

` + "```go" + `
fmt.Println("x")
` + "```" + `

![img](https://x/y.png)

##### Deep heading
`

	out := Apply(in)
	for _, want := range []string{"# Title", "**bold**", "*italic*", "`code`", "1. one", "| A | B |", "```go"} {
		if !strings.Contains(out, want) {
			t.Errorf("filtered output missing %q\noutput:\n%s", want, out)
		}
	}
	// Blockquote is passed through verbatim by upstream (content + marker).
	if !strings.Contains(out, "> a quote") {
		t.Errorf("blockquote should be passed through verbatim\noutput:\n%s", out)
	}
	// H5 markers stripped, text kept.
	if strings.Contains(out, "##### ") {
		t.Errorf("H5 marker should be stripped\noutput:\n%s", out)
	}
	if !strings.Contains(out, "Deepheading") && !strings.Contains(out, "Deep heading") {
		t.Errorf("H5 text should be kept\noutput:\n%s", out)
	}
	// Image removed.
	if strings.Contains(out, "![img]") {
		t.Errorf("image should be removed\noutput:\n%s", out)
	}
}

func TestStreaming_FeedFlushRoundTrip(t *testing.T) {
	// Feeding the input one rune/byte at a time must yield the same result
	// as Apply on the whole string.
	in := "**bold** and `code`\n> quote\n![img](u)"
	whole := Apply(in)

	var f StreamingMarkdownFilter
	var got strings.Builder
	for i := 0; i < len(in); i++ {
		got.WriteString(f.Feed(string(in[i])))
	}
	got.WriteString(f.Flush())

	if got.String() != whole {
		t.Fatalf("streaming != whole\n streaming: %q\n     whole: %q", got.String(), whole)
	}
}

func TestToPlainText_DeprecatedAlias(t *testing.T) {
	// The deprecated alias must delegate to Apply.
	if ToPlainText("**x**") != Apply("**x**") {
		t.Fatal("ToPlainText should equal Apply")
	}
}
