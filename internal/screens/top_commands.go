package screens

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/internal/articles"
	"github.com/elpdev/hackernews/internal/hn"
)

func (t Top) loadStories() tea.Cmd {
	feed := t.feed
	if feed == "" {
		feed = hn.FeedTop
	}
	screenID := t.screenID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		ids, err := t.client.StoryIDs(ctx, feed)
		if err != nil {
			return topStoriesLoadedMsg{screenID: screenID, err: err}
		}
		if len(ids) > topStoryLimit {
			ids = ids[:topStoryLimit]
		}
		end := minScreen(len(ids), topStoriesPerPage)
		stories, err := t.client.Stories(ctx, ids[:end])
		return topStoriesLoadedMsg{screenID: screenID, ids: ids, stories: stories, err: err}
	}
}

func (t Top) loadSavedIDs() tea.Cmd {
	if t.saved == nil {
		return nil
	}
	screenID := t.screenID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		items, err := t.saved.List(ctx)
		if err != nil {
			return savedIDsLoadedMsg{screenID: screenID, err: err}
		}
		ids := make(map[int]bool, len(items))
		for _, item := range items {
			ids[item.ID] = true
		}
		return savedIDsLoadedMsg{screenID: screenID, ids: ids}
	}
}

func (t Top) loadReadIDs() tea.Cmd {
	if t.history == nil {
		return nil
	}
	screenID := t.screenID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ids, err := t.history.ReadIDs(ctx)
		return readIDsLoadedMsg{screenID: screenID, ids: ids, err: err}
	}
}

func (t Top) markRead(id int) tea.Cmd {
	if t.history == nil || id == 0 {
		return nil
	}
	screenID := t.screenID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return storyMarkedReadMsg{screenID: screenID, id: id, err: t.history.MarkRead(ctx, id)}
	}
}

func (t Top) toggleSaved(id int) tea.Cmd {
	if t.saved == nil {
		return func() tea.Msg {
			return articleSavedToggledMsg{screenID: t.screenID(), id: id, err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	screenID := t.screenID()
	story, ok := t.storyByID(id)
	if !ok {
		story = hn.Item{ID: id, Type: "story"}
	}
	article := t.articleForStory(story)
	alreadySaved := t.savedIDs[id]
	return func() tea.Msg {
		if alreadySaved {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return articleSavedToggledMsg{screenID: screenID, id: id, saved: false, err: t.saved.Delete(ctx, id)}
		}
		if !t.hasExtractedArticle(story) {
			var err error
			article, err = t.extractArticleForStory(story)
			if err != nil {
				return articleSavedToggledMsg{screenID: screenID, id: id, err: err}
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return articleSavedToggledMsg{screenID: screenID, id: id, article: article, saved: true, err: t.saved.Save(ctx, story, article)}
	}
}

func (t Top) loadArticle(story hn.Item) tea.Cmd {
	screenID := t.screenID()
	return func() tea.Msg {
		article, err := t.extractArticleForStory(story)
		return articleLoadedMsg{screenID: screenID, id: story.ID, article: article, err: err}
	}
}

func (t Top) extractArticleForStory(story hn.Item) (articles.Article, error) {
	if strings.TrimSpace(story.URL) == "" {
		return articles.Article{
			Title:    story.Title,
			Author:   story.By,
			URL:      fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID),
			Markdown: hnTextMarkdown(story),
		}, nil
	}
	article, err := t.extractor.Extract(story.URL)
	if article.Title == "" {
		article.Title = story.Title
	}
	if article.Author == "" {
		article.Author = story.By
	}
	if strings.TrimSpace(article.URL) == "" {
		article.URL = story.URL
	}
	return article, err
}

func (t Top) startBodyImageLoad(id int, imageURL string) (Top, tea.Cmd) {
	if imageURL == "" {
		return t, nil
	}
	if t.bodyImages == nil {
		t.bodyImages = make(map[int]map[string]articleImage)
	}
	if t.bodyImages[id] == nil {
		t.bodyImages[id] = make(map[string]articleImage)
	}
	current := t.bodyImages[id][imageURL]
	if current.url == imageURL && (len(current.bytes) > 0 || current.err != "") {
		return t, nil
	}
	t.bodyImages[id][imageURL] = articleImage{url: imageURL}
	return t, t.loadArticleImage(id, imageURL)
}

func resolveArticleImageURL(article articles.Article) string {
	return resolveImageURL(article.Image, article.URL)
}

func (t Top) loadArticleImage(id int, rawURL string) tea.Cmd {
	screenID := t.screenID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, err: err}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, err: fmt.Errorf("image returned %s", resp.Status)}
		}
		bytes, err := io.ReadAll(io.LimitReader(resp.Body, articleImageLimit+1))
		if err != nil {
			return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, err: err}
		}
		if len(bytes) > articleImageLimit {
			return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, err: fmt.Errorf("image is larger than %d bytes", articleImageLimit)}
		}
		return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, bytes: bytes}
	}
}
