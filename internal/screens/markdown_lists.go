package screens

import "strings"

func repairLooseListItems(markdown string) string {
	lines := strings.Split(markdown, "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !isMarkdownListItem(line) {
			out = append(out, line)
			continue
		}

		item := line
		for i+2 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
			next := strings.TrimSpace(lines[i+2])
			if next == "" || startsMarkdownBlock(next) {
				break
			}
			item = strings.TrimRight(item, " ") + " " + next
			i += 2
		}
		out = append(out, item)
	}
	return strings.Join(out, "\n")
}

func isMarkdownListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
		return true
	}
	for i, r := range trimmed {
		if r >= '0' && r <= '9' {
			continue
		}
		return i > 0 && (r == '.' || r == ')') && len(trimmed) > i+1 && trimmed[i+1] == ' '
	}
	return false
}

func startsMarkdownBlock(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, ">") ||
		strings.HasPrefix(trimmed, "```") ||
		strings.HasPrefix(trimmed, "~~~") ||
		strings.HasPrefix(trimmed, "---") ||
		strings.HasPrefix(trimmed, "***") ||
		strings.HasPrefix(trimmed, "___") ||
		strings.HasPrefix(trimmed, "| ") ||
		isMarkdownListItem(trimmed)
}
