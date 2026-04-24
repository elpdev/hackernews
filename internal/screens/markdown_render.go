package screens

import (
	"strings"

	"charm.land/glamour/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

func renderMarkdown(markdown string, width int) string {
	markdown = repairLooseListItems(markdown)
	markdown = fenceLooseArticleCode(markdown)
	markdown = labelUnlabeledCodeFences(markdown)
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(maxScreen(20, width)),
	)
	if err != nil {
		return markdown
	}
	out, err := r.Render(markdown)
	if err != nil {
		return markdown
	}
	return out
}

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

func looseCodeLanguages(lines []string) []string {
	langs := make([]string, 0, 3)
	for _, line := range lines {
		if lang := normalizeCodeLanguage(line); lang != "" {
			langs = append(langs, lang)
		}
	}
	return langs
}

func normalizeCodeLanguage(line string) string {
	lang := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-")))
	lang = strings.Trim(lang, "`:")
	return normalizeLanguageName(lang)
}

func normalizeLanguageName(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "curl", "shell", "bash", "sh", "zsh", "fish", "terminal", "console":
		return "bash"
	case "python", "py":
		return "python"
	case "nodejs", "node", "javascript", "js", "typescript", "ts":
		return "javascript"
	case "ruby", "rb", "rails":
		return "ruby"
	case "golang", "go":
		return "go"
	case "rust", "rs":
		return "rust"
	case "java":
		return "java"
	case "c", "cpp", "c++", "cc", "h", "hpp":
		return "cpp"
	case "c#", "csharp", "cs":
		return "csharp"
	case "kotlin", "kt":
		return "kotlin"
	case "yaml", "yml":
		return "yaml"
	case "php", "swift", "scala", "sql", "html", "css", "json", "xml", "toml", "dockerfile":
		return lang
	default:
		if lexer := lexers.Get(lang); lexer != nil {
			aliases := lexer.Config().Aliases
			if len(aliases) > 0 {
				return aliases[0]
			}
			return strings.ToLower(lexer.Config().Name)
		}
		return ""
	}
}

func splitShellHeredocFence(marker string, lines []string) []string {
	for i, line := range lines {
		delimiter, lang := heredocDelimiter(line)
		if delimiter == "" || lang == "" {
			continue
		}
		end := heredocEnd(lines, i+1, delimiter)
		if end == -1 {
			continue
		}

		out := make([]string, 0, len(lines)+6)
		if i > 0 {
			out = append(out, fencedCodeBlock(marker, "bash", lines[:i+1])...)
		} else {
			out = append(out, fencedCodeBlock(marker, "bash", lines[:1])...)
		}
		out = append(out, "")
		out = append(out, fencedCodeBlock(marker, lang, lines[i+1:end])...)
		if end < len(lines)-1 {
			out = append(out, "")
			out = append(out, fencedCodeBlock(marker, "bash", lines[end:])...)
		}
		return out
	}
	return nil
}

func heredocDelimiter(line string) (string, string) {
	idx := strings.Index(line, "<<")
	if idx == -1 {
		return "", ""
	}
	delimiter := strings.TrimSpace(line[idx+2:])
	delimiter = strings.TrimPrefix(delimiter, "-")
	delimiter = strings.TrimPrefix(delimiter, "~")
	delimiter = strings.Trim(delimiter, "'\"")
	if delimiter == "" {
		return "", ""
	}
	lang := normalizeLanguageName(delimiter)
	if lang == "" || lang == "text" {
		return delimiter, ""
	}
	return delimiter, lang
}

func heredocEnd(lines []string, start int, delimiter string) int {
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == delimiter {
			return i
		}
	}
	return -1
}

func inferCodeBlockLanguage(lines []string) string {
	text := strings.TrimSpace(strings.Join(lines, "\n"))
	if text == "" {
		return ""
	}
	if lang := inferCodeBlockLanguageFromSignals(lines, false); lang != "" {
		return lang
	}
	if lexer := lexers.Analyse(text); lexer != nil {
		if lang := normalizeLanguageName(lexer.Config().Name); lang != "" && lang != "text" {
			return lang
		}
		for _, alias := range lexer.Config().Aliases {
			if lang := normalizeLanguageName(alias); lang != "" && lang != "text" {
				return lang
			}
		}
	}
	return inferCodeBlockLanguageFromSignals(lines, true)
}

func inferCodeBlockLanguageFromSignals(lines []string, includeShell bool) string {
	var bashScore, rubyScore, goScore, rustScore, cScore int
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		switch {
		case strings.Contains(trimmed, "<<'RUBY'"), strings.Contains(trimmed, "<<\"RUBY\""), strings.Contains(trimmed, "<<RUBY"):
			rubyScore += 4
		case strings.HasPrefix(trimmed, "def "), strings.HasPrefix(trimmed, "class "), strings.HasPrefix(trimmed, "module "), strings.HasPrefix(trimmed, "require "), strings.HasPrefix(trimmed, "puts "):
			rubyScore += 3
		case strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, "attr_"):
			rubyScore += 2
		case strings.HasPrefix(trimmed, "package "), strings.HasPrefix(trimmed, "func "), strings.HasPrefix(trimmed, "type ") && strings.Contains(trimmed, " struct"):
			goScore += 3
		case strings.HasPrefix(trimmed, "use "), strings.HasPrefix(trimmed, "fn "), strings.HasPrefix(trimmed, "impl "), strings.HasPrefix(trimmed, "let mut "):
			rustScore += 3
		case strings.HasPrefix(trimmed, "#include"), strings.HasPrefix(trimmed, "int main"), strings.HasPrefix(trimmed, "static "):
			cScore += 3
		case includeShell && (strings.HasPrefix(lower, "make") || strings.HasPrefix(trimmed, "./") || strings.HasPrefix(lower, "sudo ") || strings.HasPrefix(lower, "cat >") || strings.HasPrefix(trimmed, "$")):
			bashScore += 2
		case includeShell && strings.HasPrefix(trimmed, "#"):
			bashScore++
		}
	}
	if rubyScore >= 4 && rubyScore >= bashScore {
		return "ruby"
	}
	if bashScore >= 2 {
		return "bash"
	}
	if rubyScore >= 3 {
		return "ruby"
	}
	if goScore >= 3 {
		return "go"
	}
	if rustScore >= 3 {
		return "rust"
	}
	if cScore >= 3 {
		return "cpp"
	}
	return ""
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
