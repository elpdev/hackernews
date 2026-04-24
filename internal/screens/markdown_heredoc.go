package screens

import "strings"

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
