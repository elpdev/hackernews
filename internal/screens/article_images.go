package screens

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/internal/articles"
)

var (
	markdownImagePattern   = regexp.MustCompile(`!\[[^\]\n]*\]\(([^)\s]+)(?:\s+[^)]*)?\)`)
	articleImageRefPattern = regexp.MustCompile(`\bImage\s+(\d+):`)
)

func articleBodyImageURLs(article articles.Article) []string {
	urls := articleImageURLs(article)
	if resolveArticleImageURL(article) != "" && len(urls) > 0 {
		return urls[1:]
	}
	return urls
}

func articleImageURLs(article articles.Article) []string {
	matches := markdownImagePattern.FindAllStringSubmatch(article.Markdown, -1)
	urls := make([]string, 0, len(matches)+1)
	seen := make(map[string]bool, len(matches)+1)
	if imageURL := resolveArticleImageURL(article); imageURL != "" {
		urls = append(urls, imageURL)
		seen[imageURL] = true
	}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		imageURL := resolveImageURL(match[1], article.URL)
		if imageURL == "" || seen[imageURL] {
			continue
		}
		seen[imageURL] = true
		urls = append(urls, imageURL)
	}
	return urls
}

func resolveImageURL(imageURL, baseURL string) string {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return ""
	}
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		return parsed.String()
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		if strings.HasPrefix(imageURL, "//") {
			return "https:" + imageURL
		}
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return base.ResolveReference(parsed).String()
}

func articleBodyImageURLForCursor(article articles.Article, lines []string, cursor int) string {
	urls := articleImageURLs(article)
	if len(urls) == 0 || len(lines) == 0 {
		return ""
	}
	cursor = clampIndex(cursor, len(lines))
	if url := articleBodyImageURLForLine(urls, lines[cursor]); url != "" {
		return url
	}
	for offset := 1; offset < len(lines); offset++ {
		prev := cursor - offset
		if prev >= 0 {
			if url := articleBodyImageURLForLine(urls, lines[prev]); url != "" {
				return url
			}
		}
		next := cursor + offset
		if next < len(lines) {
			if url := articleBodyImageURLForLine(urls, lines[next]); url != "" {
				return url
			}
		}
	}
	return ""
}

func articleBodyImageURLForLine(urls []string, line string) string {
	match := articleImageRefPattern.FindStringSubmatch(ansi.Strip(line))
	if len(match) < 2 {
		return ""
	}
	n, err := strconv.Atoi(match[1])
	if err != nil || n < 1 || n > len(urls) {
		return ""
	}
	return urls[n-1]
}
