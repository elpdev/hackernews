package sync

import (
	"testing"
	"time"

	"github.com/elpdev/hackernews/pkg/articles"
	"github.com/elpdev/hackernews/pkg/history"
	"github.com/elpdev/hackernews/pkg/hn"
	"github.com/elpdev/hackernews/pkg/saved"
)

func TestMergeHistoryKeepsEarliestAndLatestReads(t *testing.T) {
	first := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	middle := first.Add(time.Hour)
	last := first.Add(2 * time.Hour)

	got := MergeHistory(
		[]history.Entry{{ID: 1, FirstRead: middle, LastRead: middle}},
		[]history.Entry{{ID: 1, FirstRead: first, LastRead: last}, {ID: 2, FirstRead: last, LastRead: last}},
	)

	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %+v", got)
	}
	byID := historyByID(got)
	if !byID[1].FirstRead.Equal(first) || !byID[1].LastRead.Equal(last) {
		t.Fatalf("unexpected merged read entry: %+v", byID[1])
	}
}

func TestMergeSavedTombstoneDeletesOlderSave(t *testing.T) {
	savedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	deletedAt := savedAt.Add(time.Hour)

	items, deleted := MergeSaved(
		[]saved.Article{{ID: 1, SavedAt: savedAt, Story: hn.Item{Title: "Old"}}},
		nil,
		[]saved.DeletedArticle{{ID: 1, DeletedAt: deletedAt}},
		nil,
	)

	if len(items) != 0 {
		t.Fatalf("expected save to be deleted, got %+v", items)
	}
	if len(deleted) != 1 || deleted[0].ID != 1 {
		t.Fatalf("expected tombstone to remain, got %+v", deleted)
	}
}

func TestMergeSavedNewerSaveBeatsOlderTombstone(t *testing.T) {
	deletedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	savedAt := deletedAt.Add(time.Hour)

	items, deleted := MergeSaved(
		[]saved.Article{{ID: 1, SavedAt: savedAt, Story: hn.Item{Title: "New"}}},
		nil,
		[]saved.DeletedArticle{{ID: 1, DeletedAt: deletedAt}},
		nil,
	)

	if len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("expected save to remain, got %+v", items)
	}
	if len(deleted) != 0 {
		t.Fatalf("expected tombstone to be removed, got %+v", deleted)
	}
}

func TestMergeSavedKeepsRicherArticleForSameSaveTime(t *testing.T) {
	savedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)

	items, _ := MergeSaved(
		[]saved.Article{{ID: 1, SavedAt: savedAt, Article: articles.Article{Markdown: "short"}}},
		[]saved.Article{{ID: 1, SavedAt: savedAt, Article: articles.Article{Markdown: "longer body"}}},
		nil,
		nil,
	)

	if len(items) != 1 || items[0].Article.Markdown != "longer body" {
		t.Fatalf("expected richer article, got %+v", items)
	}
}

func TestMergeSavedUnionsTags(t *testing.T) {
	savedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)

	items, _ := MergeSaved(
		[]saved.Article{{ID: 1, SavedAt: savedAt, Tags: []string{"go", "later"}}},
		[]saved.Article{{ID: 1, SavedAt: savedAt, Tags: []string{"Later", "databases"}}},
		nil,
		nil,
	)

	if len(items) != 1 {
		t.Fatalf("expected one item, got %+v", items)
	}
	want := []string{"databases", "go", "later"}
	if len(items[0].Tags) != len(want) {
		t.Fatalf("expected tags %+v, got %+v", want, items[0].Tags)
	}
	for i := range want {
		if items[0].Tags[i] != want[i] {
			t.Fatalf("expected tags %+v, got %+v", want, items[0].Tags)
		}
	}
}

func historyByID(entries []history.Entry) map[int]history.Entry {
	byID := make(map[int]history.Entry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	return byID
}
