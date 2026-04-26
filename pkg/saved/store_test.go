package saved

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/elpdev/hackernews/pkg/articles"
	"github.com/elpdev/hackernews/pkg/hn"
)

func TestJSONStoreSaveListGetDelete(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "saved.json"))
	store.now = func() time.Time { return time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	story := hn.Item{ID: 1, Title: "Story", URL: "https://example.com", By: "alice"}
	article := articles.Article{Title: "Story", URL: "https://example.com", Markdown: "Body"}
	if err := store.Save(ctx, story, article); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	items, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != 1 || items[0].Article.Markdown != "Body" {
		t.Fatalf("unexpected saved items: %+v", items)
	}

	found, ok, err := store.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok || found.Story.Title != "Story" {
		t.Fatalf("expected saved story, got ok=%t item=%+v", ok, found)
	}

	saved, err := store.IsSaved(ctx, 1)
	if err != nil || !saved {
		t.Fatalf("expected saved=true, got saved=%t err=%v", saved, err)
	}

	if err := store.Delete(ctx, 1); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	items, err = store.List(ctx)
	if err != nil {
		t.Fatalf("list after delete failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no saved items, got %+v", items)
	}
}

func TestJSONStoreKeepsSavedAtOnDuplicateSave(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "saved.json"))
	ctx := context.Background()
	first := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	second := first.Add(time.Hour)

	store.now = func() time.Time { return first }
	if err := store.Save(ctx, hn.Item{ID: 1, Title: "Old"}, articles.Article{Title: "Old"}); err != nil {
		t.Fatalf("first save failed: %v", err)
	}
	if err := store.SetTags(ctx, 1, []string{"Go", "later", "go"}); err != nil {
		t.Fatalf("set tags failed: %v", err)
	}
	store.now = func() time.Time { return second }
	if err := store.Save(ctx, hn.Item{ID: 1, Title: "New"}, articles.Article{Title: "New"}); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	item, ok, err := store.Get(ctx, 1)
	if err != nil || !ok {
		t.Fatalf("get failed: ok=%t err=%v", ok, err)
	}
	if !item.SavedAt.Equal(first) {
		t.Fatalf("expected original saved time %s, got %s", first, item.SavedAt)
	}
	if item.Story.Title != "New" {
		t.Fatalf("expected updated story, got %+v", item.Story)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "go" || item.Tags[1] != "later" {
		t.Fatalf("expected duplicate save to preserve normalized tags, got %+v", item.Tags)
	}
}

func TestJSONStoreSetTags(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "saved.json"))
	ctx := context.Background()
	if err := store.Save(ctx, hn.Item{ID: 1, Title: "Story"}, articles.Article{}); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if err := store.SetTags(ctx, 1, []string{" databases ", "Go", "go", ""}); err != nil {
		t.Fatalf("set tags failed: %v", err)
	}
	item, ok, err := store.Get(ctx, 1)
	if err != nil || !ok {
		t.Fatalf("get failed: ok=%t err=%v", ok, err)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "databases" || item.Tags[1] != "go" {
		t.Fatalf("unexpected tags: %+v", item.Tags)
	}
}

func TestJSONStoreListsNewestFirst(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "saved.json"))
	ctx := context.Background()
	base := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)

	store.now = func() time.Time { return base }
	if err := store.Save(ctx, hn.Item{ID: 1, Title: "Old"}, articles.Article{}); err != nil {
		t.Fatalf("save old failed: %v", err)
	}
	store.now = func() time.Time { return base.Add(time.Hour) }
	if err := store.Save(ctx, hn.Item{ID: 2, Title: "New"}, articles.Article{}); err != nil {
		t.Fatalf("save new failed: %v", err)
	}

	items, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 2 || items[0].ID != 2 || items[1].ID != 1 {
		t.Fatalf("expected newest first, got %+v", items)
	}
}
