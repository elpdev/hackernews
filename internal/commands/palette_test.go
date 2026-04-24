package commands

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestPaletteRootShowsCategoriesOnly(t *testing.T) {
	model := testPalette()

	matches := model.matches()
	if len(matches) != 2 {
		t.Fatalf("expected two root categories, got %d", len(matches))
	}
	if matches[0].ID != "browse" || matches[1].ID != "appearance" {
		t.Fatalf("unexpected root matches: %#v", matches)
	}
}

func TestPaletteCanNavigateNestedCategory(t *testing.T) {
	model := testPalette()

	model, _ = model.Update(testKeyPress(tea.Key{Code: tea.KeyEnter}))
	if !model.inCategory() {
		t.Fatal("expected enter on root category to open nested category")
	}
	matches := model.matches()
	if len(matches) != 2 {
		t.Fatalf("expected two browse commands, got %d", len(matches))
	}
	if matches[0].ID != "go-top" || matches[1].ID != "go-new" {
		t.Fatalf("unexpected nested matches: %#v", matches)
	}

	model, _ = model.Update(testKeyPress(tea.Key{Code: tea.KeyEscape}))
	if model.inCategory() {
		t.Fatal("expected esc inside category to return to root")
	}
	if model.Action().Type == PaletteActionClose {
		t.Fatal("did not expect esc inside category to close palette")
	}
}

func TestPaletteQuerySearchesNestedCommands(t *testing.T) {
	model := testPalette()
	for _, r := range "theme" {
		model, _ = model.Update(testKeyPress(tea.Key{Text: string(r), Code: r}))
	}

	matches := model.matches()
	if len(matches) != 1 {
		t.Fatalf("expected one global search match, got %d", len(matches))
	}
	if matches[0].ID != "themes" {
		t.Fatalf("expected nested themes command, got %q", matches[0].ID)
	}
}

func TestPaletteCommandLinesAreTruncatedForSelectionPadding(t *testing.T) {
	line := truncatePaletteLine("Sync Now           Manually sync saved and read articles", paletteSelectedContentWidth)
	if len([]rune(line)) > paletteSelectedContentWidth {
		t.Fatalf("expected line width <= %d, got %d", paletteSelectedContentWidth, len([]rune(line)))
	}
	if line == "Sync Now           Manually sync saved and read articles" {
		t.Fatal("expected long command line to be truncated")
	}
}

func testPalette() PaletteModel {
	registry := NewRegistry()
	registry.Register(Command{ID: "browse", Title: "Browse", Description: "Feeds and story lists", Order: 10})
	registry.Register(Command{ID: "appearance", Title: "Appearance", Description: "Themes and visual settings", Order: 20})
	registry.Register(Command{ID: "go-top", ParentID: "browse", Title: "Top Stories", Description: "Open Hacker News top stories", Keywords: []string{"top"}, Order: 10, Run: func() tea.Cmd { return nil }})
	registry.Register(Command{ID: "go-new", ParentID: "browse", Title: "New", Description: "Open newest Hacker News stories", Keywords: []string{"new"}, Order: 20, Run: func() tea.Cmd { return nil }})
	registry.Register(Command{ID: "themes", ParentID: "appearance", Title: "Themes", Description: "Preview and select a theme", Keywords: []string{"theme"}, Order: 10})
	return NewPaletteModel(registry, nil)
}

func testKeyPress(key tea.Key) tea.KeyPressMsg {
	return tea.KeyPressMsg(key)
}
