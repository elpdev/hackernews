package screens

import (
	"errors"
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

func TestListViewSpacesTitleFromStories(t *testing.T) {
	top := NewTop()
	top.loading = ""
	top.storyIDs = []int{1}
	top.stories = []hn.Item{{Title: "First story", URL: "https://example.com/first", Score: 10, By: "alice", Descendants: 3}}

	lines := strings.Split(ansi.Strip(top.listView(80, 12)), "\n")
	title := lineIndex(lines, "Top Hacker News")
	story := lineIndex(lines, "First story")
	if title < 0 || story < 0 {
		t.Fatalf("expected title and story in %q", strings.Join(lines, "\n"))
	}
	if story != title+2 {
		t.Fatalf("expected one blank line between title and first story; got title=%d story=%d", title, story)
	}
	if strings.TrimSpace(lines[title+1]) != "" {
		t.Fatalf("expected blank line after title, got %q", lines[title+1])
	}
}

func TestListViewUsesCompactStoryIndent(t *testing.T) {
	top := NewTop()
	top.loading = ""
	top.storyIDs = []int{1}
	top.stories = []hn.Item{{Title: "First story", URL: "https://example.com/first", Score: 10, By: "alice", Descendants: 3}}

	lines := strings.Split(ansi.Strip(top.listView(80, 12)), "\n")
	story := lineIndex(lines, "1. First story")
	meta := lineIndex(lines, "10 points by alice")
	if story < 0 || meta < 0 {
		t.Fatalf("expected story and metadata in %q", strings.Join(lines, "\n"))
	}
	if column := strings.Index(lines[story], "1. First story"); column != 2 {
		t.Fatalf("expected story rank at column 2, got %d in %q", column, lines[story])
	}
	if column := strings.Index(lines[meta], "10 points by alice"); column != 3 {
		t.Fatalf("expected metadata at column 3, got %d in %q", column, lines[meta])
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
	if !strings.Contains(view, "1 matches") {
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

func TestAllLoadedStoriesSpansPages(t *testing.T) {
	top := NewTop()
	top.loading = ""
	top.storyIDs = []int{11, 12, 13, 14}
	top.pages[0] = []hn.Item{
		{ID: 11, Title: "A"},
		{ID: 12, Title: "B"},
	}
	top.pages[1] = []hn.Item{
		{ID: 13, Title: "C"},
		{ID: 14, Title: "D"},
	}

	items := top.allLoadedStories()
	if len(items) != 4 {
		t.Fatalf("expected 4 loaded stories across pages, got %d", len(items))
	}
	wantTitles := []string{"A", "B", "C", "D"}
	for i, item := range items {
		if item.index != i {
			t.Fatalf("item %d has index %d, want %d", i, item.index, i)
		}
		if item.story.Title != wantTitles[i] {
			t.Fatalf("item %d has title %q, want %q", i, item.story.Title, wantTitles[i])
		}
	}
}

func TestSortedStoriesByPointsDescending(t *testing.T) {
	top := topWithStories()
	top.sortMode = sortPoints

	items := top.sortedFilteredStories()
	gotTitles := make([]string, 0, len(items))
	gotIndices := make([]int, 0, len(items))
	for _, item := range items {
		gotTitles = append(gotTitles, item.story.Title)
		gotIndices = append(gotIndices, item.index)
	}
	if strings.Join(gotTitles, ",") != "SQLite tips,Rust story,Go story" {
		t.Fatalf("expected points-desc order, got %v", gotTitles)
	}
	if gotIndices[0] != 2 || gotIndices[1] != 1 || gotIndices[2] != 0 {
		t.Fatalf("expected preserved HN-rank indices [2 1 0], got %v", gotIndices)
	}
}

func TestSortedStoriesByRecentDescending(t *testing.T) {
	top := topWithStories()
	top.sortMode = sortRecent

	items := top.sortedFilteredStories()
	gotTitles := make([]string, 0, len(items))
	for _, item := range items {
		gotTitles = append(gotTitles, item.story.Title)
	}
	if strings.Join(gotTitles, ",") != "Go story,SQLite tips,Rust story" {
		t.Fatalf("expected recent-desc order, got %v", gotTitles)
	}
}

func TestSortKeyCyclesModes(t *testing.T) {
	top := topWithStories()

	top = updateTopWithKey(t, top, tea.Key{Text: "o", Code: 'o'})
	if top.sortMode != sortRecent {
		t.Fatalf("expected sortRecent after 1st o, got %v", top.sortMode)
	}
	top = updateTopWithKey(t, top, tea.Key{Text: "o", Code: 'o'})
	if top.sortMode != sortPoints {
		t.Fatalf("expected sortPoints after 2nd o, got %v", top.sortMode)
	}
	top = updateTopWithKey(t, top, tea.Key{Text: "o", Code: 'o'})
	if top.sortMode != sortDefault {
		t.Fatalf("expected sortDefault after 3rd o, got %v", top.sortMode)
	}
}

func TestFilterSpansAllLoadedPages(t *testing.T) {
	top := NewTop()
	top.loading = ""
	top.storyIDs = []int{1, 2, 3, 4}
	top.pages[0] = []hn.Item{
		{ID: 1, Title: "Go story"},
		{ID: 2, Title: "Rust story"},
	}
	top.pages[1] = []hn.Item{
		{ID: 3, Title: "Python tricks"},
		{ID: 4, Title: "Another Rust post"},
	}
	top.page = 0
	top.stories = top.pages[0]
	top.searchQuery = "rust"

	titles := filteredStoryTitles(top)
	if len(titles) != 2 {
		t.Fatalf("expected 2 cross-page matches, got %v", titles)
	}
	if !(titles[0] == "Rust story" && titles[1] == "Another Rust post") {
		t.Fatalf("expected matches from both pages in rank order, got %v", titles)
	}
}

func TestDefaultViewStaysPageScoped(t *testing.T) {
	top := NewTop()
	top.loading = ""
	top.storyIDs = []int{1, 2, 3, 4}
	top.pages[0] = []hn.Item{
		{ID: 1, Title: "p0a"},
		{ID: 2, Title: "p0b"},
	}
	top.pages[1] = []hn.Item{
		{ID: 3, Title: "p1a"},
		{ID: 4, Title: "p1b"},
	}
	top.page = 0
	top.stories = top.pages[0]

	items := top.filteredStories()
	if len(items) != 2 {
		t.Fatalf("expected default view to be page-scoped (2 items), got %d", len(items))
	}
	if items[0].story.Title != "p0a" || items[1].story.Title != "p0b" {
		t.Fatalf("expected page-0 stories, got %v", items)
	}
	if items[0].index != 0 || items[1].index != 1 {
		t.Fatalf("expected global-rank indices [0 1], got [%d %d]", items[0].index, items[1].index)
	}
}

func TestSortCombinedWithSearchFilter(t *testing.T) {
	top := topWithStories()
	top.searchQuery = "story"
	top.sortMode = sortPoints

	items := top.sortedFilteredStories()
	gotTitles := make([]string, 0, len(items))
	for _, item := range items {
		gotTitles = append(gotTitles, item.story.Title)
	}
	if strings.Join(gotTitles, ",") != "Rust story,Go story" {
		t.Fatalf("expected filtered+sorted result, got %v", gotTitles)
	}
}

func topWithStories() Top {
	top := NewTop()
	top.loading = ""
	top.storyIDs = []int{1, 2, 3}
	stories := []hn.Item{
		{ID: 1, Title: "Go story", URL: "https://example.com/go", Score: 10, By: "alice", Descendants: 3, Time: 300},
		{ID: 2, Title: "Rust story", URL: "https://rust-lang.org/news", Score: 20, By: "bob", Descendants: 4, Time: 100},
		{ID: 3, Title: "SQLite tips", URL: "https://sqlite.org/tips", Score: 30, By: "carol", Descendants: 5, Time: 200},
	}
	top.stories = stories
	top.pages[0] = stories
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

func TestArticleLoadedMsgWithErrorStillOpensReader(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	top := NewTop()

	msg := articleLoadedMsg{
		id: 42,
		article: articles.Article{
			Title: "Broken page",
			URL:   "https://example.com/spa",
		},
		err: errors.New("trafilatura did not find readable article content"),
	}
	updated, _ := top.Update(msg)
	got := updated.(Top)

	if got.readID != 42 {
		t.Fatalf("expected reader to open on extraction error, readID=%d", got.readID)
	}
	if got.err == "" {
		t.Fatal("expected error surfaced in header")
	}
	stored, ok := got.articles[42]
	if !ok {
		t.Fatal("expected article record populated even on error")
	}
	if stored.URL != "https://example.com/spa" {
		t.Fatalf("expected article URL preserved, got %q", stored.URL)
	}
}

func TestRenderArticleFallbackBodyWhenMarkdownEmpty(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	article := articles.Article{
		Title: "Broken page",
		URL:   "https://example.com/spa",
	}

	lines := renderedArticleLines(101, 80, article, articleImage{})
	body := ansi.Strip(strings.Join(lines, "\n"))
	if !strings.Contains(body, "Couldn't extract readable content") {
		t.Fatalf("expected fallback copy in %q", body)
	}
	if !strings.Contains(body, "https://example.com/spa") {
		t.Fatalf("expected URL in fallback body, got %q", body)
	}
	if !strings.Contains(body, "o") {
		t.Fatalf("expected hint to press o, got %q", body)
	}
}

func TestRenderArticleNoFallbackWhenBodyPresent(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	article := articles.Article{
		Title:    "Story",
		URL:      "https://example.com/story",
		Markdown: "Article body.",
	}

	lines := renderedArticleLines(102, 80, article, articleImage{})
	body := ansi.Strip(strings.Join(lines, "\n"))
	if strings.Contains(body, "Couldn't extract") {
		t.Fatalf("did not expect fallback copy when body present: %q", body)
	}
}

func TestArticleOpenKeyInvokesOpener(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	top := NewTop()
	var gotURL string
	var calls int
	top.opener = func(url string) error {
		calls++
		gotURL = url
		return nil
	}
	top.articles[7] = articles.Article{URL: "https://example.com/x"}
	top.readID = 7

	top = updateTopWithKey(t, top, tea.Key{Text: "o", Code: 'o'})

	if calls != 1 {
		t.Fatalf("expected opener called once, got %d", calls)
	}
	if gotURL != "https://example.com/x" {
		t.Fatalf("expected URL forwarded to opener, got %q", gotURL)
	}
	if top.status != "Opening in browser..." {
		t.Fatalf("expected success status, got %q", top.status)
	}
}

func TestArticleOpenKeySurfacesOpenerError(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	top := NewTop()
	top.opener = func(string) error { return errors.New("xdg-open not found") }
	top.articles[8] = articles.Article{URL: "https://example.com/y"}
	top.readID = 8

	top = updateTopWithKey(t, top, tea.Key{Text: "o", Code: 'o'})

	if !strings.Contains(top.status, "xdg-open not found") {
		t.Fatalf("expected opener error in status, got %q", top.status)
	}
}

func TestArticleOpenKeyWithoutURLGivesStatus(t *testing.T) {
	articleRenderCache.lines = make(map[string][]string)
	top := NewTop()
	top.opener = func(string) error {
		t.Fatal("opener should not be called without URL")
		return nil
	}
	top.articles[9] = articles.Article{}
	top.readID = 9

	top = updateTopWithKey(t, top, tea.Key{Text: "o", Code: 'o'})

	if top.status != "No URL to open" {
		t.Fatalf("expected 'No URL to open', got %q", top.status)
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
