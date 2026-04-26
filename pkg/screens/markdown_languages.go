package screens

import (
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
)

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
