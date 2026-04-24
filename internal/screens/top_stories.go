package screens

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/internal/articles"
	"github.com/elpdev/hackernews/internal/hn"
)

type sortMode int

const (
	sortDefault sortMode = iota
	sortRecent
	sortPoints
)

func (m sortMode) label() string {
	switch m {
	case sortRecent:
		return "recent"
	case sortPoints:
		return "points"
	default:
		return ""
	}
}

func (m sortMode) String() string {
	s := m.label()
	if s == "" {
		return "default"
	}
	return s
}

func sortModeFromString(value string) sortMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "recent":
		return sortRecent
	case "points":
		return sortPoints
	default:
		return sortDefault
	}
}

type storyListItem struct {
	index int
	story hn.Item
}

func (t Top) hasExtractedArticle(story hn.Item) bool {
	article, ok := t.articles[story.ID]
	if ok && strings.TrimSpace(article.Markdown) != "" {
		return true
	}
	return strings.TrimSpace(story.URL) == "" || strings.TrimSpace(story.Text) != ""
}

func (t Top) storyByID(id int) (hn.Item, bool) {
	for _, page := range t.pages {
		for _, story := range page {
			if story.ID == id {
				return story, true
			}
		}
	}
	for _, story := range t.stories {
		if story.ID == id {
			return story, true
		}
	}
	return hn.Item{}, false
}

func (t Top) articleForStory(story hn.Item) articles.Article {
	if article, ok := t.articles[story.ID]; ok {
		return article
	}
	articleURL := t.articleURLForStory(story)
	article := articles.Article{Title: story.Title, Author: story.By, URL: articleURL}
	if strings.TrimSpace(story.Text) != "" {
		article.Markdown = hnTextMarkdown(story)
	}
	return article
}

func (t Top) articleURLForID(id int) string {
	if article, ok := t.articles[id]; ok && strings.TrimSpace(article.URL) != "" {
		return article.URL
	}
	if story, ok := t.storyByID(id); ok {
		return t.articleURLForStory(story)
	}
	return ""
}

func (t Top) articleURLForStory(story hn.Item) string {
	if strings.TrimSpace(story.URL) != "" {
		return story.URL
	}
	if story.ID == 0 {
		return ""
	}
	return fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)
}

func (t Top) goToPage(page int) (Screen, tea.Cmd) {
	if stories, ok := t.pages[page]; ok {
		t.page = page
		t.stories = stories
		t.selected = 0
		t.listTop = 0
		return t, nil
	}
	t.loading = fmt.Sprintf("Loading page %d...", page+1)
	t.err = ""
	return t, t.loadStoryPage(page)
}

func (t Top) loadStoryPage(page int) tea.Cmd {
	ids := append([]int(nil), t.storyIDs...)
	screenID := t.screenID()
	return func() tea.Msg {
		start := page * topStoriesPerPage
		if start >= len(ids) {
			return storyPageLoadedMsg{screenID: screenID, page: page, stories: nil}
		}
		end := minScreen(len(ids), start+topStoriesPerPage)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		stories, err := t.client.Stories(ctx, ids[start:end])
		return storyPageLoadedMsg{screenID: screenID, page: page, stories: stories, err: err}
	}
}

func (t Top) allLoadedStories() []storyListItem {
	loaded := make(map[int]hn.Item, len(t.pages)*topStoriesPerPage)
	for _, page := range t.pages {
		for _, story := range page {
			loaded[story.ID] = story
		}
	}
	for _, story := range t.stories {
		loaded[story.ID] = story
	}
	items := make([]storyListItem, 0, len(loaded))
	for rank, id := range t.storyIDs {
		if story, ok := loaded[id]; ok {
			items = append(items, storyListItem{index: rank, story: story})
		}
	}
	return items
}

func (t Top) filteredStories() []storyListItem {
	var scope []storyListItem
	if t.searchQuery != "" || t.sortMode != sortDefault {
		scope = t.allLoadedStories()
	} else {
		scope = make([]storyListItem, 0, len(t.stories))
		base := t.page * topStoriesPerPage
		for i, story := range t.stories {
			scope = append(scope, storyListItem{index: base + i, story: story})
		}
	}
	query := strings.ToLower(strings.TrimSpace(t.searchQuery))
	if t.hideRead {
		out := scope[:0]
		for _, item := range scope {
			if !t.readIDs[item.story.ID] {
				out = append(out, item)
			}
		}
		scope = out
	}
	if query == "" {
		return scope
	}
	out := scope[:0]
	for _, item := range scope {
		if storyMatchesQuery(item.story, query) {
			out = append(out, item)
		}
	}
	return out
}

func (t Top) sortedFilteredStories() []storyListItem {
	items := t.filteredStories()
	switch t.sortMode {
	case sortRecent:
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].story.Time > items[j].story.Time
		})
	case sortPoints:
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].story.Score > items[j].story.Score
		})
	}
	return items
}

func storyMatchesQuery(story hn.Item, query string) bool {
	fields := []string{story.Title, story.By, story.URL, storyDomain(story.URL)}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func hnTextMarkdown(story hn.Item) string {
	text := html.UnescapeString(story.Text)
	text = strings.ReplaceAll(text, "<p>", "\n\n")
	text = strings.ReplaceAll(text, "<pre><code>", "\n\n```")
	text = strings.ReplaceAll(text, "</code></pre>", "```\n\n")
	return "# " + story.Title + "\n\n" + text
}

func storyDomain(raw string) string {
	if raw == "" {
		return "news.ycombinator.com"
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Hostname(), "www.")
}

func (t Top) pageCount() int {
	if len(t.storyIDs) == 0 {
		return 1
	}
	return (len(t.storyIDs) + topStoriesPerPage - 1) / topStoriesPerPage
}

func (t Top) selectedInPage() bool {
	return t.selected >= 0 && t.selected < len(t.stories)
}

func clampPage(page, length int) int {
	if length <= 0 {
		return 0
	}
	pages := (length + topStoriesPerPage - 1) / topStoriesPerPage
	if page < 0 {
		return 0
	}
	if page >= pages {
		return pages - 1
	}
	return page
}
