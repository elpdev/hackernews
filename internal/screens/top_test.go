package screens

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/internal/articles"
)

func TestRenderedArticleLinesPlacesImageAfterTitle(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	article := articles.Article{
		Title:    "Story title",
		Author:   "alice",
		URL:      "https://example.com/story",
		Image:    "https://example.com/image.jpg",
		Markdown: "Article body.",
	}

	lines := renderedArticleLines(1, 80, article, articleImage{url: article.Image})
	titleIndex := lineIndex(lines, "Story title")
	imageIndex := lineIndex(lines, "Image: loading...")
	metaIndex := lineIndex(lines, "by alice")
	if titleIndex < 0 {
		t.Fatalf("expected rendered title in %q", strings.Join(lines, "\n"))
	}
	if imageIndex < 0 {
		t.Fatalf("expected image loading line in %q", strings.Join(lines, "\n"))
	}
	if metaIndex < 0 {
		t.Fatalf("expected rendered metadata in %q", strings.Join(lines, "\n"))
	}
	if !(titleIndex < imageIndex && imageIndex < metaIndex) {
		t.Fatalf("expected title, image, metadata order; got title=%d image=%d meta=%d", titleIndex, imageIndex, metaIndex)
	}
}

func TestRenderedArticleLinesOmitsImageWithoutArticleImage(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	article := articles.Article{
		Title:    "Story title",
		Markdown: "Article body.",
	}

	lines := renderedArticleLines(2, 80, article, articleImage{})
	if lineIndex(lines, "Image:") >= 0 {
		t.Fatalf("expected no image line in %q", strings.Join(lines, "\n"))
	}
}

func TestRenderedArticleLinesCapsWideArticleWidth(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	article := articles.Article{
		Markdown: strings.Repeat("word ", 80),
	}

	lines := renderedArticleLines(3, articleContentWidth(140), article, articleImage{})
	for _, line := range lines {
		if width := lipgloss.Width(ansi.Strip(line)); width > articleMaxWidth {
			t.Fatalf("expected line width <= %d, got %d for %q", articleMaxWidth, width, ansi.Strip(line))
		}
	}
}

func TestStartArticleImageLoadRestartsCachedLoadingImage(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	top := NewTop()
	top.images[7] = articleImage{url: "https://example.com/image.jpg"}

	updated, cmd := top.startArticleImageLoad(7, articles.Article{
		URL:   "https://example.com/story",
		Image: "https://example.com/image.jpg",
	})
	if cmd == nil {
		t.Fatal("expected image load command to restart for cached loading state")
	}
	if updated.images[7].url != "https://example.com/image.jpg" {
		t.Fatalf("unexpected image URL: %q", updated.images[7].url)
	}
}

func TestStartArticleImageLoadKeepsLoadedImage(t *testing.T) {
	top := NewTop()
	top.images[7] = articleImage{url: "https://example.com/image.jpg", bytes: []byte("image")}

	updated, cmd := top.startArticleImageLoad(7, articles.Article{
		URL:   "https://example.com/story",
		Image: "https://example.com/image.jpg",
	})
	if cmd != nil {
		t.Fatal("expected no image load command for already loaded image")
	}
	if string(updated.images[7].bytes) != "image" {
		t.Fatal("expected loaded image bytes to be preserved")
	}
}

func TestResolveArticleImageURL(t *testing.T) {
	article := articles.Article{URL: "https://example.com/news/story", Image: "/media/hero.webp"}
	if got := resolveArticleImageURL(article); got != "https://example.com/media/hero.webp" {
		t.Fatalf("unexpected resolved URL: %q", got)
	}

	article = articles.Article{URL: "https://example.com/news/story", Image: "//cdn.example.com/hero.webp"}
	if got := resolveArticleImageURL(article); got != "https://cdn.example.com/hero.webp" {
		t.Fatalf("unexpected protocol-relative URL: %q", got)
	}
}

func lineIndex(lines []string, needle string) int {
	for i, line := range lines {
		if strings.Contains(ansi.Strip(line), needle) {
			return i
		}
	}
	return -1
}
