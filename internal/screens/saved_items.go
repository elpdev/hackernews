package screens

import (
	"fmt"
	"sort"
	"strings"

	"github.com/elpdev/hackernews/internal/saved"
)

func (s Saved) itemByID(id int) (saved.Article, bool) {
	for _, item := range s.items {
		if item.ID == id {
			return item, true
		}
	}
	return saved.Article{}, false
}

func savedTitle(item saved.Article) string {
	if item.Article.Title != "" {
		return item.Article.Title
	}
	if item.Story.Title != "" {
		return item.Story.Title
	}
	return fmt.Sprintf("HN item %d", item.ID)
}

func (s Saved) filteredItems() []savedListItem {
	items := make([]savedListItem, 0, len(s.items))
	query := strings.ToLower(strings.TrimSpace(s.searchQuery))
	for i, item := range s.items {
		if query == "" || savedMatchesQuery(item, query) {
			items = append(items, savedListItem{index: i, item: item})
		}
	}
	sortSavedItems(items, s.sortMode)
	return items
}

func sortSavedItems(items []savedListItem, mode savedSortMode) {
	switch mode {
	case savedSortStoryDate:
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].item.Story.Time > items[j].item.Story.Time
		})
	case savedSortTitle:
		sort.SliceStable(items, func(i, j int) bool {
			return strings.ToLower(savedTitle(items[i].item)) < strings.ToLower(savedTitle(items[j].item))
		})
	}
}

func savedMatchesQuery(item saved.Article, query string) bool {
	fields := []string{savedTitle(item), item.Story.By, item.Story.URL, item.Article.URL, storyDomain(savedArticleURL(item)), item.Article.Markdown, strings.Join(item.Tags, " ")}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func parseSavedTags(input string) []string {
	parts := strings.Split(input, ",")
	seen := make(map[string]bool, len(parts))
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.ToLower(strings.TrimSpace(part))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func savedArticleURL(item saved.Article) string {
	if strings.TrimSpace(item.Article.URL) != "" {
		return item.Article.URL
	}
	if strings.TrimSpace(item.Story.URL) != "" {
		return item.Story.URL
	}
	if item.ID == 0 {
		return ""
	}
	return fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID)
}
