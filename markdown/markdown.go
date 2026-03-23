// Package markdown provides markdown to plain text conversion.
package markdown

import (
	"regexp"
	"strings"
)

var (
	// Code blocks: ```...```
	codeBlockRegex = regexp.MustCompile("```[^\n]*\n?([\\s\\S]*?)```")

	// Inline images: ![alt](url)
	inlineImageRegex = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)

	// Links: [text](url)
	inlineLinkRegex = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)

	// Table separator rows: |---|---|
	tableSeparatorRegex = regexp.MustCompile(`^\|[\s:|-]+\|$`)

	// Table rows: | ... |
	tableRowRegex = regexp.MustCompile(`^\|(.+)\|$`)
)

// ToPlainText converts markdown-formatted text to plain text.
// Preserves newlines; strips markdown syntax.
func ToPlainText(text string) string {
	result := text

	// Code blocks: strip fences, keep code content
	result = codeBlockRegex.ReplaceAllString(result, "$1")

	// Images: remove entirely
	result = inlineImageRegex.ReplaceAllString(result, "")

	// Links: keep display text only
	result = inlineLinkRegex.ReplaceAllString(result, "$1")

	// Tables: remove separator rows, then strip leading/trailing pipes and convert inner pipes to spaces
	result = tableSeparatorRegex.ReplaceAllString(result, "")
	result = tableRowRegex.ReplaceAllStringFunc(result, func(match string) string {
		inner := strings.Trim(match, "|")
		cells := strings.Split(inner, "|")
		// Trim whitespace and join with spaces
		var trimmed []string
		for _, cell := range cells {
			trimmed = append(trimmed, strings.TrimSpace(cell))
		}
		return strings.Join(trimmed, "  ")
	})

	// Bold: **text** or __text__
	result = strings.ReplaceAll(result, "**", "")
	result = strings.ReplaceAll(result, "__", "")

	// Italic: *text* or _text_
	// This is a simple replacement - may remove too many underscores in practice
	result = strings.ReplaceAll(result, "*", "")
	result = strings.ReplaceAll(result, "_", "")

	// Strikethrough: ~~text~~
	result = strings.ReplaceAll(result, "~~", "")

	// Headers: #, ##, ###, etc.
	headerRegex := regexp.MustCompile(`^#+\s+`)
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		lines[i] = headerRegex.ReplaceAllString(line, "")
	}
	result = strings.Join(lines, "\n")

	// Blockquotes: > at start of line
	blockquoteRegex := regexp.MustCompile(`^>\s+`)
	lines = strings.Split(result, "\n")
	for i, line := range lines {
		lines[i] = blockquoteRegex.ReplaceAllString(line, "")
	}
	result = strings.Join(lines, "\n")

	// Horizontal rules: --- or ***
	hrRegex := regexp.MustCompile(`^[-*_]{3,}\s*$`)
	result = hrRegex.ReplaceAllString(result, "")

	// Code spans: `code`
	result = strings.ReplaceAll(result, "`", "")

	// Clean up extra whitespace
	result = strings.TrimSpace(result)

	return result
}