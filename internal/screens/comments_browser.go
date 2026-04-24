package screens

import "strings"

func commentsOpenURL(opener func(string) error, url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to open"
	}
	if opener == nil {
		return "Could not open browser: no opener configured"
	}
	if err := opener(url); err != nil {
		return "Could not open browser: " + err.Error()
	}
	return "Opening in browser..."
}

func commentsCopyURL(copier func(string) error, url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to copy"
	}
	if copier == nil {
		return "Could not copy URL: no clipboard configured"
	}
	if err := copier(url); err != nil {
		return "Could not copy URL: " + err.Error()
	}
	return "Copied URL to clipboard"
}
