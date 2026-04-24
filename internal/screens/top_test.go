package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/internal/articles"
	"github.com/elpdev/hackernews/internal/hn"
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

func TestRenderedArticleLinesSeparatesMetadataFromBody(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	article := articles.Article{
		Author:   "alice",
		URL:      "https://example.com/story",
		Markdown: "Article body.",
	}

	lines := renderedArticleLines(4, 80, article, articleImage{})
	metaIndex := lineIndex(lines, "by alice")
	bodyIndex := lineIndex(lines, "Article body.")
	if metaIndex < 0 || bodyIndex < 0 {
		t.Fatalf("expected metadata and body in %q", strings.Join(lines, "\n"))
	}
	if bodyIndex != metaIndex+2 {
		t.Fatalf("expected one empty line between metadata and body; got meta=%d body=%d in %q", metaIndex, bodyIndex, strings.Join(lines, "\n"))
	}
	if strings.TrimSpace(ansi.Strip(lines[metaIndex+1])) != "" {
		t.Fatalf("expected empty separator line, got %q", ansi.Strip(lines[metaIndex+1]))
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

func TestListViewSeparatesStories(t *testing.T) {
	top := NewTop()
	top.storyIDs = []int{1, 2}
	top.stories = []hn.Item{
		{Title: "First story", URL: "https://example.com/first", Score: 10, By: "alice", Descendants: 3},
		{Title: "Second story", URL: "https://example.com/second", Score: 20, By: "bob", Descendants: 4},
	}

	lines := strings.Split(ansi.Strip(top.listView(80, 12)), "\n")
	firstMeta := lineIndex(lines, "10 points by alice")
	secondTitle := lineIndex(lines, "Second story")
	if firstMeta < 0 || secondTitle < 0 {
		t.Fatalf("expected both stories in %q", strings.Join(lines, "\n"))
	}
	if secondTitle != firstMeta+2 {
		t.Fatalf("expected one blank line between stories; got meta=%d second=%d in %q", firstMeta, secondTitle, strings.Join(lines, "\n"))
	}
	if strings.TrimSpace(lines[firstMeta+1]) != "" {
		t.Fatalf("expected blank story separator, got %q", lines[firstMeta+1])
	}
}

func TestFilteredStoriesMatchesTitleAuthorAndDomain(t *testing.T) {
	top := topWithStories()

	top.searchQuery = "alice"
	if got := filteredStoryTitles(top); strings.Join(got, ",") != "Go story" {
		t.Fatalf("expected author match, got %v", got)
	}

	top.searchQuery = "rust"
	if got := filteredStoryTitles(top); strings.Join(got, ",") != "Rust story" {
		t.Fatalf("expected title match, got %v", got)
	}

	top.searchQuery = "example"
	if got := filteredStoryTitles(top); strings.Join(got, ",") != "Go story" {
		t.Fatalf("expected domain match, got %v", got)
	}
}

func TestListViewShowsFilteredResults(t *testing.T) {
	top := topWithStories()
	top.searchQuery = "go"

	view := ansi.Strip(top.listView(80, 12))
	if !strings.Contains(view, "Go story") {
		t.Fatalf("expected matching story in %q", view)
	}
	if strings.Contains(view, "Rust story") {
		t.Fatalf("did not expect non-matching story in %q", view)
	}
	if !strings.Contains(view, "1 matches on page") {
		t.Fatalf("expected filtered count in %q", view)
	}
}

func TestSearchKeysEditQueryAndPreserveFilterOnEscape(t *testing.T) {
	top := topWithStories()
	top.searching = true

	top = updateTopWithKey(t, top, tea.Key{Text: "q", Code: 'q'})
	if top.searchQuery != "q" {
		t.Fatalf("expected q to be captured in search query, got %q", top.searchQuery)
	}

	top = updateTopWithKey(t, top, tea.Key{Code: tea.KeyBackspace})
	if top.searchQuery != "" {
		t.Fatalf("expected backspace to remove query, got %q", top.searchQuery)
	}

	top = updateTopWithKey(t, top, tea.Key{Text: "g", Code: 'g'})
	top = updateTopWithKey(t, top, tea.Key{Code: tea.KeyEscape})
	if top.searching {
		t.Fatal("expected esc to leave search mode")
	}
	if top.searchQuery != "g" {
		t.Fatalf("expected esc to preserve filter, got %q", top.searchQuery)
	}

	top = updateTopWithKey(t, top, tea.Key{Code: 'u', Mod: tea.ModCtrl})
	if top.searchQuery != "" {
		t.Fatalf("expected ctrl+u to clear filter, got %q", top.searchQuery)
	}
}

func TestEnterReadsSelectedFilteredStory(t *testing.T) {
	top := topWithStories()
	top.searchQuery = "rust"
	top.articles[2] = articles.Article{Title: "Rust story"}

	top = updateTopWithKey(t, top, tea.Key{Code: tea.KeyEnter})
	if top.readID != 2 {
		t.Fatalf("expected filtered selection to read story 2, got %d", top.readID)
	}
}

func topWithStories() Top {
	top := NewTop()
	top.loading = ""
	top.storyIDs = []int{1, 2, 3}
	top.stories = []hn.Item{
		{ID: 1, Title: "Go story", URL: "https://example.com/go", Score: 10, By: "alice", Descendants: 3},
		{ID: 2, Title: "Rust story", URL: "https://rust-lang.org/news", Score: 20, By: "bob", Descendants: 4},
		{ID: 3, Title: "SQLite tips", URL: "https://sqlite.org/tips", Score: 30, By: "carol", Descendants: 5},
	}
	return top
}

func filteredStoryTitles(top Top) []string {
	matches := top.filteredStories()
	titles := make([]string, 0, len(matches))
	for _, match := range matches {
		titles = append(titles, match.story.Title)
	}
	return titles
}

func updateTopWithKey(t *testing.T, top Top, key tea.Key) Top {
	t.Helper()
	updated, _ := top.Update(tea.KeyPressMsg(key))
	return updated.(Top)
}

func lineIndex(lines []string, needle string) int {
	for i, line := range lines {
		if strings.Contains(ansi.Strip(line), needle) {
			return i
		}
	}
	return -1
}
