package cli

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
)

const (
	blockIndentWidth             = 2
	wideEntryWidth               = 96
	mediumEntryWidth             = 60
	wideEntryLabelCap            = 34
	mediumEntryLabelCap          = 28
	minimumEntryDescriptionWidth = 24
	entryDescriptionIndentWidth  = 4
	singleByteMaxRune            = '\u00ff'
)

func render(ctx context.Context, document document, profile profile) (string, error) {
	profile = normalizeProfile(profile)
	var out strings.Builder
	for _, block := range document.Blocks {
		if err := ctx.Err(); err != nil {
			return "", err //nolint:wrapcheck // Render preserves exact context cancellation identity.
		}
		renderBlock(&out, block, profile)
	}
	return out.String(), nil
}

func renderBlock(out *strings.Builder, block block, profile profile) {
	switch block.Kind {
	case blockParagraph:
		writeWrapped(out, renderSpans(block.Spans, profile), profile.Width, "", "")
	case blockSection:
		ensureSectionBreak(out)
		out.WriteString(styleText(escapeTerminalInlineControls(block.Title), spanHeading, profile))
		out.WriteString(":\n")
	case blockUsageLines:
		for _, line := range block.Lines {
			text := styleText(escapeTerminalInlineControls(line), spanLiteral, profile)
			writeWrapped(out, text, max(1, profile.Width-blockIndentWidth), "  ", "  ")
		}
	case blockEntryList:
		renderEntries(out, block.Entries, profile)
	case blockBulletList:
		for _, item := range block.Items {
			writeWrapped(out, escapeTerminalControls(item), max(1, profile.Width-blockIndentWidth), "- ", "  ")
		}
	case blockDiagnosticGroup:
		if block.Title != "" {
			out.WriteString(styleText(escapeTerminalInlineControls(block.Title), spanError, profile))
			out.WriteString(":\n")
		}
		for _, item := range block.Items {
			writeWrapped(out, escapeTerminalControls(item), max(1, profile.Width-blockIndentWidth), "  ", "  ")
		}
	case blockHint:
		writeWrapped(out, renderSpans(block.Spans, profile), profile.Width, "", "")
	case blockLiteralLine:
		for _, line := range block.Lines {
			out.WriteString(styleText(escapeTerminalInlineControls(line), spanLiteral, profile))
			out.WriteByte('\n')
		}
	}
}

func ensureSectionBreak(out *strings.Builder) {
	if out.Len() == 0 {
		return
	}
	text := out.String()
	if !strings.HasSuffix(text, "\n") {
		out.WriteByte('\n')
	}
	if !strings.HasSuffix(out.String(), "\n\n") {
		out.WriteByte('\n')
	}
}

func renderEntries(out *strings.Builder, entries []presentationEntry, profile profile) {
	maxLabel := 0
	for _, entry := range entries {
		maxLabel = max(maxLabel, displayWidth(escapeTerminalInlineControls(entry.Label)))
	}
	leftCap := 0
	switch {
	case profile.Width >= wideEntryWidth:
		leftCap = wideEntryLabelCap
	case profile.Width >= mediumEntryWidth:
		leftCap = mediumEntryLabelCap
	}
	column := leftCap != 0 && maxLabel <= leftCap && profile.Width-maxLabel-entryDescriptionIndentWidth >= minimumEntryDescriptionWidth
	for _, entry := range entries {
		plainLabel := escapeTerminalInlineControls(entry.Label)
		label := styleText(plainLabel, spanLiteral, profile)
		description := escapeTerminalControls(entry.Description)
		if !column || description == "" {
			writeWrapped(out, label, max(1, profile.Width-blockIndentWidth), "  ", "  ")
			if description != "" {
				writeWrapped(out, description, max(1, profile.Width-entryDescriptionIndentWidth), "    ", "    ")
			}
			continue
		}
		padding := maxLabel - displayWidth(plainLabel)
		prefix := "  " + label + strings.Repeat(" ", padding) + "  "
		continuation := strings.Repeat(" ", maxLabel+entryDescriptionIndentWidth)
		paragraphs := strings.Split(description, "\n")
		writeWrapped(out, paragraphs[0], max(1, profile.Width-maxLabel-entryDescriptionIndentWidth), prefix, continuation)
		for _, paragraph := range paragraphs[1:] {
			writeWrapped(out, paragraph, max(1, profile.Width-maxLabel-entryDescriptionIndentWidth), continuation, continuation)
		}
	}
}

func renderSpans(spans []presentationSpan, profile profile) string {
	var out strings.Builder
	for _, span := range spans {
		out.WriteString(styleText(escapeTerminalControls(span.Text), span.Role, profile))
	}
	return out.String()
}

func displayWidth(text string) int {
	if !strings.Contains(text, "\x1b[") {
		return runewidth.StringWidth(text)
	}
	var plain strings.Builder
	for i := 0; i < len(text); {
		if text[i] == 0x1b && i+1 < len(text) && text[i+1] == '[' {
			i += 2
			for i < len(text) {
				b := text[i]
				i++
				if b >= 0x40 && b <= 0x7e {
					break
				}
			}
			continue
		}
		plain.WriteByte(text[i])
		i++
	}
	return runewidth.StringWidth(plain.String())
}

func styleText(text string, role spanRole, profile profile) string {
	if !profile.Styled || text == "" {
		return text
	}
	var code string
	switch role {
	case spanHeading:
		code = "1"
	case spanError:
		code = "31;1"
	case spanWarning:
		code = "33;1"
	case spanLiteral:
		code = "36"
	default:
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func escapeTerminalControls(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	var out strings.Builder
	for _, r := range text {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) && (r < 0x80 || r > 0x9f) {
			out.WriteRune(r)
			continue
		}
		if r <= singleByteMaxRune {
			fmt.Fprintf(&out, "\\x%02x", r)
		} else {
			fmt.Fprintf(&out, "\\u%04x", r)
		}
	}
	return out.String()
}

func escapeTerminalInlineControls(text string) string {
	var out strings.Builder
	for _, r := range text {
		if !unicode.IsControl(r) && (r < 0x80 || r > 0x9f) {
			out.WriteRune(r)
			continue
		}
		if r <= singleByteMaxRune {
			fmt.Fprintf(&out, "\\x%02x", r)
		} else {
			fmt.Fprintf(&out, "\\u%04x", r)
		}
	}
	return out.String()
}

func writeWrapped(out *strings.Builder, text string, width int, firstPrefix, continuationPrefix string) {
	if width < 1 {
		width = 1
	}
	paragraphs := strings.Split(text, "\n")
	for paragraphIndex, paragraph := range paragraphs {
		if paragraphIndex > 0 && paragraph == "" {
			out.WriteByte('\n')
			continue
		}
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out.WriteString(firstPrefix)
			out.WriteByte('\n')
			continue
		}
		prefix := firstPrefix
		lineWidth := 0
		out.WriteString(prefix)
		for index, word := range words {
			wordWidth := displayWidth(word)
			separator := 0
			if index > 0 && lineWidth > 0 {
				separator = 1
			}
			if lineWidth > 0 && lineWidth+separator+wordWidth > width {
				out.WriteByte('\n')
				out.WriteString(continuationPrefix)
				lineWidth = 0
				separator = 0
			}
			if separator != 0 {
				out.WriteByte(' ')
				lineWidth++
			}
			out.WriteString(word)
			lineWidth += wordWidth
		}
		out.WriteByte('\n')
	}
}
