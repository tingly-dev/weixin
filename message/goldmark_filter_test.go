package message

import (
	"strings"
	"testing"
)

// These tests validate ApplyGoldmark against the same expectations as the
// state-machine-based Apply (markdown_filter_test.go), so both engines are
// checked against one spec. A parity test additionally asserts the two agree
// on a representative corpus (modulo trailing whitespace, and excluding the
// known setext-heading divergence where goldmark parses "above\n---" as an H2).

func TestApplyGoldmark_PreservesMarkdown(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bold kept", "**bold**", "**bold**"},
		{"italic non-CJK kept", "*italic*", "*italic*"},
		{"inline code kept", "use `code` here", "use `code` here"},
		{"code fence kept verbatim", "```go\nfmt.Println(\"hi\")\n```", "```go\nfmt.Println(\"hi\")\n```"},
		{"table kept", "| a | b |\n|---|---|\n| 1 | 2 |", "| a | b |\n|---|---|\n| 1 | 2 |"},
		{"link kept as-is", "[text](https://example.com)", "[text](https://example.com)"},
		{"strikethrough kept", "~~deleted~~", "~~deleted~~"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := trimSpace(ApplyGoldmark(tc.in))
			if got != tc.want {
				t.Fatalf("ApplyGoldmark(%q)\n got: %q\nwant: %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestApplyGoldmark_StripsBadMarkdown(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		contains string
		omits    string
	}{
		{"CJK italic markers stripped", "*中文斜体*", "中文斜体", "*"},
		{"H5 markers stripped", "##### Heading Five", "Heading Five", "#####"},
		{"H6 markers stripped", "###### Heading Six", "Heading Six", "######"},
		{"image removed entirely", "before ![alt](https://x/y.png) after", "before", "![alt](https://x/y.png)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ApplyGoldmark(tc.in)
			if tc.contains != "" && !strings.Contains(got, tc.contains) {
				t.Fatalf("output missing %q\n got: %q", tc.contains, got)
			}
			if tc.omits != "" && strings.Contains(got, tc.omits) {
				t.Fatalf("output should not contain %q\n got: %q", tc.omits, got)
			}
		})
	}
}

// TestApplyGoldmark_Parity asserts ApplyGoldmark matches Apply across a
// corpus, modulo cosmetic differences that are goldmark being MORE correct
// than the state machine:
//   - goldmark canonicalizes "_" emphasis to "*" (we normalize both to "*").
//   - goldmark nests headings inside blockquotes correctly ("> # Title"),
//     where the state machine leaves the marker outside. These mixed
//     blockquote/heading cases are excluded from parity and documented.
// Trailing whitespace is normalized (goldmark emits blank lines between
// blocks; the state machine does not). Setext headings ("above\n---") are
// also excluded — goldmark parses the underlined text as an H2.
func TestApplyGoldmark_Parity(t *testing.T) {
	corpus := []string{
		"**bold**",
		"*italic*",
		"***bold-italic***",
		"plain text only",
		"use `inline code` here",
		"```go\nfmt.Println(\"hi\")\n```",
		"| a | b |\n|---|---|\n| 1 | 2 |",
		"# H1",
		"## H2\n### H3\n#### H4",
		"##### H5 stripped",
		"###### H6 stripped",
		"[text](https://example.com)",
		"~~deleted~~",
		"*中文斜体*",
		"**中文加粗**",
		"before ![alt](https://x/y.png) after",
		"> a quote",
		"- one\n- two",
		"1. first\n2. second",
	}
	for _, in := range corpus {
		g := normEmph(trimSpace(ApplyGoldmark(in)))
		a := normEmph(trimSpace(Apply(in)))
		if g != a {
			t.Errorf("parity mismatch for %q\n goldmark: %q\n    apply: %q", in, g, a)
		}
	}
}

// trimSpace collapses cosmetic blank-line/whitespace differences so the two
// engines can be compared on content.
func trimSpace(s string) string {
	s = strings.TrimSpace(s)
	for strings.Contains(s, "\n\n") {
		s = strings.ReplaceAll(s, "\n\n", "\n")
	}
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

// normEmph canonicalizes underscore emphasis to asterisk emphasis, since
// goldmark renders "__x__" as "**x**" while the state machine keeps "__x__".
func normEmph(s string) string {
	return strings.NewReplacer("__", "**", "_", "*").Replace(s)
}
