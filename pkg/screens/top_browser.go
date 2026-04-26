package screens

import (
	"strings"
)

func (t Top) openArticleURL(url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to open"
	}
	if t.opener == nil {
		return "Could not open browser: no opener configured"
	}
	if err := t.opener(url); err != nil {
		return "Could not open browser: " + err.Error()
	}
	return "Opening in browser..."
}

func (t Top) copyArticleURL(url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to copy"
	}
	if t.copier == nil {
		return "Could not copy URL: no clipboard configured"
	}
	if err := t.copier(url); err != nil {
		return "Could not copy URL: " + err.Error()
	}
	return "Copied URL to clipboard"
}
