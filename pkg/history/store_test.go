package history

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestJSONStoreMarkReadAndReadIDs(t *testing.T) {
	store := NewJSONStore(filepath.Join(t.TempDir(), "history.json"))
	store.now = func() time.Time { return time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	if err := store.MarkRead(ctx, 42); err != nil {
		t.Fatalf("mark read failed: %v", err)
	}
	ids, err := store.ReadIDs(ctx)
	if err != nil {
		t.Fatalf("read IDs failed: %v", err)
	}
	if !ids[42] {
		t.Fatalf("expected story 42 to be read: %+v", ids)
	}
}
