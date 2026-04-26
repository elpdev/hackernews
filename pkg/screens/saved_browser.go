package screens

import "strings"

func (s Saved) openArticleURL(url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to open"
	}
	if s.opener == nil {
		return "Could not open browser: no opener configured"
	}
	if err := s.opener(url); err != nil {
		return "Could not open browser: " + err.Error()
	}
	return "Opening in browser..."
}

func (s Saved) copyArticleURL(url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to copy"
	}
	if s.copier == nil {
		return "Could not copy URL: no clipboard configured"
	}
	if err := s.copier(url); err != nil {
		return "Could not copy URL: " + err.Error()
	}
	return "Copied URL to clipboard"
}
