package screens

import (
	"html"
	"regexp"
	"strings"

	"github.com/elpdev/hackernews/internal/hn"
)

var commentLinkRE = regexp.MustCompile(`(?is)<a\s+[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
var commentTagRE = regexp.MustCompile(`(?is)</?[a-z][^>]*>`)

func commentBodyMarkdown(item hn.Item) string {
	if item.Deleted || item.Dead {
		return "*[deleted]*"
	}
	return commentHTMLToMarkdown(item.Text)
}

func commentHTMLToMarkdown(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	t := text
	t = strings.ReplaceAll(t, "<p>", "\n\n")
	t = strings.ReplaceAll(t, "</p>", "")
	t = strings.ReplaceAll(t, "<i>", "*")
	t = strings.ReplaceAll(t, "</i>", "*")
	t = strings.ReplaceAll(t, "<pre><code>", "\n\n```\n")
	t = strings.ReplaceAll(t, "</code></pre>", "\n```\n\n")
	t = strings.ReplaceAll(t, "<code>", "`")
	t = strings.ReplaceAll(t, "</code>", "`")
	t = commentLinkRE.ReplaceAllString(t, "[$2]($1)")
	t = commentTagRE.ReplaceAllString(t, "")
	return html.UnescapeString(t)
}
