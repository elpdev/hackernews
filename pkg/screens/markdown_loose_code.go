package screens

import "strings"

func fenceLooseArticleCode(markdown string) string {
	lines := strings.Split(markdown, "\n")
	langs := looseCodeLanguages(lines)
	out := make([]string, 0, len(lines)+len(langs)*2)
	inCode := false
	codeLang := "text"
	codeLines := make([]string, 0)
	for _, line := range lines {
		if isLooseCodeStart(line) {
			if inCode {
				out = appendLooseCodeBlock(out, codeLang, codeLines)
				out = append(out, "")
			}
			codeLang = "text"
			if len(langs) > 0 {
				codeLang = langs[0]
				langs = langs[1:]
			} else if inferred := inferLooseCodeLanguage(line); inferred != "" {
				codeLang = inferred
			}
			codeLines = append(codeLines[:0], cleanLooseCodeStart(line))
			inCode = true
			continue
		}
		if inCode && strings.HasPrefix(strings.TrimSpace(line), "##") {
			out = appendLooseCodeBlock(out, codeLang, codeLines)
			codeLines = codeLines[:0]
			inCode = false
		}
		if inCode {
			codeLines = append(codeLines, line)
			continue
		}
		out = append(out, line)
	}
	if inCode {
		out = appendLooseCodeBlock(out, codeLang, codeLines)
	}
	return strings.Join(out, "\n")
}

func appendLooseCodeBlock(out []string, lang string, lines []string) []string {
	out = append(out, "```"+lang)
	out = append(out, indentLooseCode(lang, compactLooseCodeLines(lines))...)
	return append(out, "```")
}

func compactLooseCodeLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = cleanLooseCodeStart(line)
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func isLooseCodeStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") || !strings.HasPrefix(trimmed, "`") {
		return false
	}
	return inferLooseCodeLanguage(trimmed) != ""
}

func inferLooseCodeLanguage(line string) string {
	code := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "`"))
	switch {
	case strings.HasPrefix(code, "curl "), strings.HasPrefix(code, "-H "), strings.HasPrefix(code, "-d "):
		return "bash"
	case strings.HasPrefix(code, "#"), strings.HasPrefix(code, "import "), strings.HasPrefix(code, "from "):
		return "python"
	case strings.HasPrefix(code, "//"), strings.HasPrefix(code, "const "), strings.HasPrefix(code, "let "), strings.HasPrefix(code, "async function"):
		return "javascript"
	case strings.HasPrefix(code, "require "), strings.HasPrefix(code, "module "), strings.HasPrefix(code, "def "), strings.HasPrefix(code, "puts "):
		return "ruby"
	case strings.HasPrefix(code, "class ") && !strings.Contains(code, "{"):
		return "ruby"
	case strings.HasPrefix(code, "package main"), strings.HasPrefix(code, "func "):
		return "go"
	case strings.HasPrefix(code, "use "), strings.HasPrefix(code, "fn "), strings.HasPrefix(code, "impl "):
		return "rust"
	default:
		return ""
	}
}

func cleanLooseCodeStart(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimLeft(line, "`")
	line = strings.TrimRight(line, "`")
	return strings.ReplaceAll(line, "`", "")
}
