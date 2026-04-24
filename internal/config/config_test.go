package config

import (
	"path/filepath"
	"testing"
)

func TestStoreLoadDefaultsWhenMissing(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "config.json"))
	settings, err := store.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if settings.ThemeName != "Phosphor" || !settings.ShowSidebar || settings.DefaultFeed != "top" {
		t.Fatalf("unexpected defaults: %+v", settings)
	}
}

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "config.json"))
	want := Settings{ThemeName: "Synthwave", ShowSidebar: false, DefaultFeed: "saved", SortMode: "points", HideRead: true}
	if err := store.Save(want); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
