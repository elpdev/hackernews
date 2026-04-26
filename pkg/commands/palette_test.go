package commands

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/pkg/theme"
)

func TestPaletteCanNavigateNestedCategory(t *testing.T) {
	model := testPalette()

	model, _ = model.Update(testKeyPress(tea.Key{Code: tea.KeyEnter}))
	view := model.View(testTheme())
	if !strings.Contains(view, "Command Palette / Browse") {
		t.Fatalf("expected browse category title, got %q", view)
	}
	if !strings.Contains(view, "Top Stories") || !strings.Contains(view, "New") {
		t.Fatalf("expected browse commands, got %q", view)
	}

	model, _ = model.Update(testKeyPress(tea.Key{Code: tea.KeyEscape}))
	if model.Action().Type == PaletteActionClose {
		t.Fatal("did not expect esc inside category to close palette")
	}
	if strings.Contains(model.View(testTheme()), "Command Palette / Browse") {
		t.Fatal("expected esc inside category to return to root")
	}
}

func TestPaletteQuerySearchesNestedCommands(t *testing.T) {
	model := testPalette()
	for _, r := range "theme" {
		model, _ = model.Update(testKeyPress(tea.Key{Text: string(r), Code: r}))
	}

	view := model.View(testTheme())
	if !strings.Contains(view, "Themes") {
		t.Fatalf("expected nested themes command, got %q", view)
	}
}

func TestPaletteOpensThemePage(t *testing.T) {
	model := testPalette()
	for _, r := range "theme" {
		model, _ = model.Update(testKeyPress(tea.Key{Text: string(r), Code: r}))
	}
	model, _ = model.Update(testKeyPress(tea.Key{Code: tea.KeyEnter}))

	if got := model.View(testTheme()); !strings.Contains(got, "Command Palette / Themes") {
		t.Fatalf("expected themes page, got %q", got)
	}
}

func TestPaletteConfirmsSyncSetup(t *testing.T) {
	model := testPalette()
	model.SetSyncSetup(SyncSetup{Remote: "origin", Branch: "main", Dir: "sync"})
	for _, r := range "setup" {
		model, _ = model.Update(testKeyPress(tea.Key{Text: string(r), Code: r}))
	}
	model, _ = model.Update(testKeyPress(tea.Key{Code: tea.KeyEnter}))
	model, _ = model.Update(testKeyPress(tea.Key{Code: tea.KeyEnter}))
	model, _ = model.Update(testKeyPress(tea.Key{Code: tea.KeyEnter}))
	model, _ = model.Update(testKeyPress(tea.Key{Code: tea.KeyEnter}))

	action := model.Action()
	if action.Type != PaletteActionConfirmSyncSetup || action.Sync == nil {
		t.Fatalf("expected sync setup action, got %#v", action)
	}
	if action.Sync.Remote != "origin" || action.Sync.Branch != "main" || action.Sync.Dir != "sync" {
		t.Fatalf("unexpected sync setup: %#v", action.Sync)
	}
}

func testPalette() PaletteModel {
	registry := NewRegistry()
	registry.Register(Command{ID: "browse", Title: "Browse", Description: "Feeds and story lists", Order: 10})
	registry.Register(Command{ID: "appearance", Title: "Appearance", Description: "Themes and visual settings", Order: 20})
	registry.Register(Command{ID: "sync", Title: "Sync", Description: "Sync actions", Order: 30})
	registry.Register(Command{ID: "go-top", ParentID: "browse", Title: "Top Stories", Description: "Open Hacker News top stories", Keywords: []string{"top"}, Order: 10, Run: func() tea.Cmd { return nil }})
	registry.Register(Command{ID: "go-new", ParentID: "browse", Title: "New", Description: "Open newest Hacker News stories", Keywords: []string{"new"}, Order: 20, Run: func() tea.Cmd { return nil }})
	registry.Register(Command{ID: "themes", ParentID: "appearance", Title: "Themes", Description: "Preview and select a theme", Keywords: []string{"theme"}, Order: 10})
	registry.Register(Command{ID: "setup-sync", ParentID: "sync", Title: "Setup Sync", Description: "Configure Git sync", Keywords: []string{"setup", "sync"}, Order: 10})
	return NewPaletteModel(registry, []theme.Theme{testTheme()})
}

func testTheme() theme.Theme {
	return theme.Theme{Name: "test"}
}

func testKeyPress(key tea.Key) tea.KeyPressMsg {
	return tea.KeyPressMsg(key)
}
