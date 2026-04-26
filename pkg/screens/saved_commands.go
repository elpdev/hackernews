package screens

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/pkg/articles"
	"github.com/elpdev/hackernews/pkg/saved"
)

func (s Saved) load() tea.Cmd {
	if s.store == nil {
		return func() tea.Msg {
			return savedArticlesLoadedMsg{screenID: "saved", err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		items, err := s.store.List(ctx)
		return savedArticlesLoadedMsg{screenID: "saved", items: items, err: err}
	}
}

func (s Saved) delete(id int) tea.Cmd {
	if s.store == nil {
		return func() tea.Msg {
			return savedArticleDeletedMsg{screenID: "saved", id: id, err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return savedArticleDeletedMsg{screenID: "saved", id: id, err: s.store.Delete(ctx, id)}
	}
}

func (s Saved) setTags(id int, tags []string) tea.Cmd {
	if s.store == nil {
		return func() tea.Msg {
			return savedArticleTagsUpdatedMsg{screenID: "saved", id: id, tags: tags, err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return savedArticleTagsUpdatedMsg{screenID: "saved", id: id, tags: tags, err: s.store.SetTags(ctx, id, tags)}
	}
}

func (s Saved) extractSavedArticle(item saved.Article) tea.Cmd {
	if s.store == nil {
		return func() tea.Msg {
			return savedArticleExtractedMsg{screenID: "saved", id: item.ID, err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	if s.extractor == nil {
		return func() tea.Msg {
			return savedArticleExtractedMsg{screenID: "saved", id: item.ID, err: fmt.Errorf("article extractor is unavailable")}
		}
	}
	return func() tea.Msg {
		article, err := extractArticleForSaved(s.extractor, item)
		if err != nil {
			return savedArticleExtractedMsg{screenID: "saved", id: item.ID, err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.store.Save(ctx, item.Story, article); err != nil {
			return savedArticleExtractedMsg{screenID: "saved", id: item.ID, err: err}
		}
		return savedArticleExtractedMsg{screenID: "saved", id: item.ID, article: article}
	}
}

func extractArticleForSaved(extractor articles.Extractor, item saved.Article) (articles.Article, error) {
	if strings.TrimSpace(item.Story.URL) == "" && strings.TrimSpace(item.Article.URL) == "" {
		return articles.Article{
			Title:    savedTitle(item),
			Author:   item.Story.By,
			URL:      savedArticleURL(item),
			Markdown: hnTextMarkdown(item.Story),
		}, nil
	}
	url := item.Article.URL
	if strings.TrimSpace(url) == "" {
		url = item.Story.URL
	}
	article, err := extractor.Extract(url)
	if article.Title == "" {
		article.Title = savedTitle(item)
	}
	if article.Author == "" {
		article.Author = item.Story.By
	}
	if strings.TrimSpace(article.URL) == "" {
		article.URL = url
	}
	return article, err
}
