package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/hackernews/internal/screens"
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

	if model.theme.Name != "Synthwave" {
		t.Fatalf("expected confirmed Synthwave theme, got %q", model.theme.Name)
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

func TestCommandPaletteDoesNotSwallowScreenMessages(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model.screens[model.activeScreen] = recordingScreen{}
	model.showCommandPalette = true

	updated, _ := model.Update(testScreenMsg{})
	model = updated.(Model)

	screen := model.screens[model.activeScreen].(recordingScreen)
	if screen.updates != 1 {
		t.Fatalf("expected active screen to receive message while palette is open, got %d updates", screen.updates)
	}
}

func TestTargetedMessageUpdatesOwningScreenWhenInactive(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model.screens["top"] = recordingScreen{}
	model.screens["new"] = recordingScreen{}
	model.activeScreen = "new"

	updated, _ := model.Update(targetedTestScreenMsg{screenID: "top"})
	model = updated.(Model)

	top := model.screens["top"].(recordingScreen)
	if top.updates != 1 {
		t.Fatalf("expected top screen to receive targeted message, got %d updates", top.updates)
	}
	newScreen := model.screens["new"].(recordingScreen)
	if newScreen.updates != 0 {
		t.Fatalf("expected active new screen to be untouched, got %d updates", newScreen.updates)
	}
}

func TestTargetedMessageUpdatesOwningScreenWithPaletteOpen(t *testing.T) {
	model := New(BuildInfo{Version: "test", Commit: "none", Date: "unknown"})
	model.screens["top"] = recordingScreen{}
	model.screens["new"] = recordingScreen{}
	model.activeScreen = "new"
	model.showCommandPalette = true

	updated, _ := model.Update(targetedTestScreenMsg{screenID: "top"})
	model = updated.(Model)

	top := model.screens["top"].(recordingScreen)
	if top.updates != 1 {
		t.Fatalf("expected top screen to receive targeted message behind palette, got %d updates", top.updates)
	}
	if !model.showCommandPalette {
		t.Fatal("expected command palette to remain open")
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

type testScreenMsg struct{}

type targetedTestScreenMsg struct {
	screenID string
}

func (m targetedTestScreenMsg) TargetScreenID() string { return m.screenID }

type recordingScreen struct {
	updates int
}

func (s recordingScreen) Init() tea.Cmd { return nil }

func (s recordingScreen) Update(tea.Msg) (screens.Screen, tea.Cmd) {
	s.updates++
	return s, nil
}

func (s recordingScreen) View(int, int) string { return "" }

func (s recordingScreen) Title() string { return "Recording" }

func (s recordingScreen) KeyBindings() []key.Binding { return nil }
