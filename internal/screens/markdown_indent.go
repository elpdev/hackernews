package screens

import "strings"

func indentLooseCode(lang string, lines []string) []string {
	switch lang {
	case "python", "javascript", "ruby", "go", "rust", "java", "cpp", "csharp", "php", "swift", "kotlin", "scala":
		return indentLooseBlockCode(lines)
	case "bash":
		return indentLooseBashCode(lines)
	default:
		return lines
	}
}

func indentLooseBlockCode(lines []string) []string {
	out := make([]string, 0, len(lines))
	indent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, "")
			continue
		}
		if startsBlockDedent(trimmed) {
			indent = maxScreen(0, indent-1)
		}
		out = append(out, strings.Repeat("  ", indent)+trimmed)
		indent = maxScreen(0, indent+blockIndentDelta(trimmed))
		if startsBlockContinuation(trimmed) {
			indent++
		}
	}
	return out
}

func startsBlockDedent(line string) bool {
	return startsWithClosingDelimiter(line) || line == "end" || line == "else" || strings.HasPrefix(line, "elsif ") || strings.HasPrefix(line, "elif ") || strings.HasPrefix(line, "when ") || strings.HasPrefix(line, "catch ") || strings.HasPrefix(line, "rescue") || line == "ensure" || strings.HasPrefix(line, "finally") || (strings.HasPrefix(line, "case ") && strings.HasSuffix(line, ":"))
}

func startsBlockContinuation(line string) bool {
	if line == "else" || strings.HasPrefix(line, "elsif ") || strings.HasPrefix(line, "elif ") || strings.HasPrefix(line, "when ") || strings.HasPrefix(line, "case ") || strings.HasPrefix(line, "catch ") || strings.HasPrefix(line, "rescue") || line == "ensure" || strings.HasPrefix(line, "finally") {
		return true
	}
	return strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "class ") || strings.HasPrefix(line, "module ") ||
		strings.HasPrefix(line, "if ") || strings.HasPrefix(line, "unless ") || strings.HasPrefix(line, "case ") ||
		strings.HasPrefix(line, "while ") || strings.HasPrefix(line, "until ") || strings.HasPrefix(line, "for ") ||
		strings.HasPrefix(line, "begin") || strings.HasSuffix(line, " do") || strings.HasSuffix(line, ":")
}

func blockIndentDelta(line string) int {
	delta := looseIndentDelta(line)
	if delta > 0 {
		return 1
	}
	if delta < 0 {
		return -1
	}
	return 0
}

func indentLooseBashCode(lines []string) []string {
	out := make([]string, 0, len(lines))
	inJSON := false
	jsonIndent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inJSON {
			content, quoted := strings.CutSuffix(trimmed, "'")
			if startsWithClosingDelimiter(content) {
				jsonIndent = maxScreen(0, jsonIndent-1)
			}
			if quoted {
				out = append(out, "  "+strings.Repeat("  ", jsonIndent)+content+"'")
				inJSON = false
				continue
			}
			out = append(out, "  "+strings.Repeat("  ", jsonIndent)+content)
			jsonIndent = maxScreen(0, jsonIndent+looseIndentDelta(content))
			continue
		}
		if strings.HasPrefix(trimmed, "-d '{") && strings.TrimSpace(strings.TrimPrefix(trimmed, "-d '")) == "{" {
			out = append(out, "  "+trimmed)
			inJSON = true
			jsonIndent = 1
			continue
		}
		if strings.HasPrefix(trimmed, "-H ") || strings.HasPrefix(trimmed, "-d ") {
			trimmed = "  " + trimmed
		}
		out = append(out, trimmed)
	}
	return out
}

func indentBracketedLooseCode(lines []string) []string {
	out := make([]string, 0, len(lines))
	indent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if startsWithClosingDelimiter(trimmed) {
			indent = maxScreen(0, indent-1)
		}
		out = append(out, strings.Repeat("  ", indent)+trimmed)
		indent = maxScreen(0, indent+looseIndentDelta(trimmed))
	}
	return out
}

func startsWithClosingDelimiter(line string) bool {
	return strings.HasPrefix(line, ")") || strings.HasPrefix(line, "]") || strings.HasPrefix(line, "}")
}

func looseIndentDelta(line string) int {
	delta := 0
	for _, r := range line {
		switch r {
		case '(', '[', '{':
			delta++
		case ')', ']', '}':
			delta--
		}
	}
	if delta > 1 {
		return 1
	}
	if delta < -1 {
		return -1
	}
	return delta
}
