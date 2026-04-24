package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSwitchScreenForTest(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = model.SwitchScreenForTest("saved")

	if model.CurrentScreenID() != "saved" {
		t.Fatalf("expected saved screen, got %q", model.CurrentScreenID())
	}
}

func TestSidebarOpenByDefault(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})

	if !model.showSidebar {
		t.Fatal("expected sidebar to be open by default")
	}
}

func TestCommandPaletteThemePreviewCanReturnToRoot(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = openThemePalette(t, model)

	model = sendKey(t, model, tea.Key{Code: tea.KeyDown})
	if model.theme.Name != "Muted Dark" {
		t.Fatalf("expected preview to switch to Muted Dark, got %q", model.theme.Name)
	}

	model = sendKey(t, model, tea.Key{Code: tea.KeyEscape})
	if model.theme.Name != "Phosphor" {
		t.Fatalf("expected esc to restore Phosphor theme, got %q", model.theme.Name)
	}
	if !model.showCommandPalette {
		t.Fatal("expected esc from theme page to keep command palette open")
	}
	if model.commandPalette.Action().Type != 0 {
		t.Fatal("expected palette action to be cleared after handling")
	}

	model = sendKey(t, model, tea.Key{Code: tea.KeyEscape})
	if model.showCommandPalette {
		t.Fatal("expected esc from root command palette to close palette")
	}
}

func TestCommandPaletteThemeSelectionConfirmsPreview(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = openThemePalette(t, model)

	model = sendKey(t, model, tea.Key{Code: tea.KeyDown})
	model = sendKey(t, model, tea.Key{Code: tea.KeyDown})
	model = sendKey(t, model, tea.Key{Code: tea.KeyEnter})

	if model.theme.Name != "Miami" {
		t.Fatalf("expected confirmed Miami theme, got %q", model.theme.Name)
	}
	if model.showCommandPalette {
		t.Fatal("expected theme selection to close command palette")
	}
}

func TestCapturedScreenKeyBypassesGlobalQuit(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = sendKey(t, model, tea.Key{Text: "/", Code: '/'})

	updated, cmd := model.Update(keyPress(tea.Key{Text: "q", Code: 'q'}))
	model = updated.(Model)
	if cmd != nil {
		t.Fatal("expected q to be captured by search instead of quitting")
	}
	if model.showHelp {
		t.Fatal("did not expect captured key to trigger global UI")
	}
}

func TestSidebarNavigationInitializesSavedScreen(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model = sendKey(t, model, tea.Key{Code: tea.KeyTab})
	for m := range model.screenOrder {
		if model.screenOrder[m] == "saved" {
			break
		}
		model = sendKey(t, model, tea.Key{Code: tea.KeyDown})
	}

	updated, cmd := model.Update(keyPress(tea.Key{Code: tea.KeyEnter}))
	model = updated.(Model)
	if model.CurrentScreenID() != "saved" {
		t.Fatalf("expected saved screen, got %q", model.CurrentScreenID())
	}
	if cmd == nil {
		t.Fatal("expected saved screen init command")
	}
}

func openThemePalette(t *testing.T, model Model) Model {
	t.Helper()
	model = sendKey(t, model, tea.Key{Code: 'k', Mod: tea.ModCtrl})
	if !model.showCommandPalette {
		t.Fatal("expected command palette to open")
	}

	for _, r := range "theme" {
		model = sendKey(t, model, tea.Key{Text: string(r), Code: r})
	}
	model = sendKey(t, model, tea.Key{Code: tea.KeyEnter})
	return model
}

func sendKey(t *testing.T, model Model, key tea.Key) Model {
	t.Helper()
	updated, cmd := model.Update(keyPress(key))
	model = updated.(Model)
	for cmd != nil {
		updated, cmd = model.Update(cmd())
		model = updated.(Model)
	}
	return model
}

func keyPress(key tea.Key) tea.KeyPressMsg {
	return tea.KeyPressMsg(key)
}
