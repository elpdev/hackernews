package screens

import "strings"

func labelUnlabeledCodeFences(markdown string) string {
	lines := strings.Split(markdown, "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if !isUnlabeledCodeFence(trimmed) {
			out = append(out, line)
			continue
		}

		fenceMarker := trimmed[:3]
		end := codeFenceEnd(lines, i+1, fenceMarker)
		if end == -1 {
			out = append(out, line)
			continue
		}
		out = append(out, normalizeUnlabeledCodeFence(fenceMarker, lines[i+1:end])...)
		i = end
	}
	return strings.Join(out, "\n")
}

func isUnlabeledCodeFence(line string) bool {
	return line == "```" || line == "~~~"
}

func codeFenceEnd(lines []string, start int, marker string) int {
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), marker) {
			return i
		}
	}
	return -1
}

func normalizeUnlabeledCodeFence(marker string, lines []string) []string {
	if blocks := splitShellHeredocFence(marker, lines); len(blocks) > 0 {
		return blocks
	}
	if lang := inferCodeBlockLanguage(lines); lang != "" {
		return fencedCodeBlock(marker, lang, lines)
	}
	return fencedCodeBlock(marker, "", lines)
}

func fencedCodeBlock(marker, lang string, lines []string) []string {
	out := make([]string, 0, len(lines)+2)
	out = append(out, marker+lang)
	out = append(out, indentLooseCode(lang, lines)...)
	return append(out, marker)
}
