package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/hackernews/internal/commands"
	"github.com/elpdev/hackernews/internal/screens"
	hnsync "github.com/elpdev/hackernews/internal/sync"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case routeMsg:
		if msg.ScreenID == "search" {
			m = m.refreshSearchScreen()
		}
		m.switchScreen(msg.ScreenID)
		m.showCommandPalette = false
		return m, m.initScreenIfNeeded(msg.ScreenID)
	case screens.NavigateMsg:
		m.switchScreen(msg.ScreenID)
		return m, m.initScreenIfNeeded(msg.ScreenID)
	case screens.OpenCommentsMsg:
		if existing, ok := m.screens["comments"].(screens.Comments); ok {
			updated, cmd := existing.Open(msg.Story, msg.ReturnTo)
			m.screens["comments"] = updated
			m.switchScreen("comments")
			return m, cmd
		}
		m.logs.Warn("Comments screen unavailable")
		return m, nil
	case toggleSidebarMsg:
		m.showSidebar = !m.showSidebar
		m.settings.ShowSidebar = m.showSidebar
		m.saveSettings()
		m.logs.Info(fmt.Sprintf("Sidebar toggled: %t", m.showSidebar))
		return m, nil
	case toggleHideReadMsg:
		return m.applyHideRead(!m.settings.HideRead), nil
	case syncNowMsg:
		m.logs.Info("Manual sync started")
		return m, m.syncNow()
	case syncCompletedMsg:
		if msg.Err != nil {
			m.logs.Warn(fmt.Sprintf("Sync failed: %v", msg.Err))
			return m, nil
		}
		if msg.Committed {
			m.logs.Info(fmt.Sprintf("Sync complete: %d saved, %d read", msg.SavedCount, msg.ReadCount))
		} else {
			m.logs.Info(fmt.Sprintf("Sync complete with no remote changes: %d saved, %d read", msg.SavedCount, msg.ReadCount))
		}
		if _, ok := m.screens[m.activeScreen].(screens.Top); ok {
			return m, m.screens[m.activeScreen].Init()
		}
		if m.activeScreen == "saved" {
			return m, m.screens[m.activeScreen].Init()
		}
		return m, nil
	case screens.HideReadToggledMsg:
		return m.applyHideRead(msg.HideRead), nil
	case screens.SortModeChangedMsg:
		m.settings.SortMode = msg.Mode
		m.saveSettings()
		return m, nil
	case quitMsg:
		m.logs.Info("Command executed: Quit")
		return m, tea.Quit
	case commandsExecutedMsg:
		m.logs.Info(fmt.Sprintf("Command executed: %s", msg.Title))
		return m, msg.Cmd
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	if targeted, ok := msg.(screens.TargetedMsg); ok {
		if id := targeted.TargetScreenID(); id != "" {
			return m.updateScreen(id, msg)
		}
	}

	return m.updateScreen(m.activeScreen, msg)
}

func (m Model) refreshSearchScreen() Model {
	search, ok := m.screens["search"].(screens.Search)
	if !ok {
		return m
	}
	var items []screens.StorySnapshot
	for _, screen := range m.screens {
		if stories, ok := screen.(screens.Top); ok {
			items = append(items, stories.Snapshot()...)
		}
	}
	m.screens["search"] = search.WithItems(items)
	return m
}

type commandsExecutedMsg struct {
	Title string
	Cmd   tea.Cmd
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.ForceQuit) {
		return m, tea.Quit
	}

	if m.showCommandPalette {
		palette, cmd := m.commandPalette.Update(msg)
		m.commandPalette = palette
		if action := m.commandPalette.Action(); action.Type != commands.PaletteActionNone {
			return m.handlePaletteAction(action)
		}
		if executed := m.commandPalette.ExecutedCommand(); executed != nil {
			m.showCommandPalette = false
			m.commandPalette.Reset(m.theme.Name)
			return m, func() tea.Msg { return commandsExecutedMsg{Title: executed.Title, Cmd: executed.Run()} }
		}
		return m, cmd
	}

	if m.showHelp {
		if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Help) {
			m.showHelp = false
		}
		return m, nil
	}

	active := m.screens[m.activeScreen]
	if capturer, ok := active.(screens.KeyCapturer); ok && capturer.CapturesKey(msg) {
		updated, cmd := active.Update(msg)
		m.screens[m.activeScreen] = updated
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.Commands):
		m.showCommandPalette = true
		m.commandPalette.Reset(m.theme.Name)
		m.commandPalette.SetSyncSetup(commands.SyncSetup{Remote: m.settings.SyncRemote, Branch: m.settings.SyncBranch, Dir: m.settings.SyncDir})
		return m, nil
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	case key.Matches(msg, m.keys.Cancel):
		active := m.screens[m.activeScreen]
		updated, cmd := active.Update(msg)
		m.screens[m.activeScreen] = updated
		return m, cmd
	case key.Matches(msg, m.keys.Focus):
		if m.focus == FocusMain && m.showSidebar {
			m.focus = FocusSidebar
		} else {
			m.focus = FocusMain
		}
		return m, nil
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}

	if m.focus == FocusSidebar && m.showSidebar {
		return m.handleSidebarKey(msg)
	}

	active = m.screens[m.activeScreen]
	updated, cmd := active.Update(msg)
	m.screens[m.activeScreen] = updated
	return m, cmd
}

func (m Model) handlePaletteAction(action commands.PaletteAction) (tea.Model, tea.Cmd) {
	m.commandPalette.ClearAction()
	switch action.Type {
	case commands.PaletteActionClose:
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name)
		return m, nil
	case commands.PaletteActionExecute:
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name)
		return m, func() tea.Msg { return commandsExecutedMsg{Title: action.Command.Title, Cmd: action.Command.Run()} }
	case commands.PaletteActionPreviewTheme:
		m.theme = *action.Theme
		return m, nil
	case commands.PaletteActionConfirmTheme:
		m.theme = *action.Theme
		m.settings.ThemeName = m.theme.Name
		m.saveSettings()
		m.logs.Info(fmt.Sprintf("Theme selected: %s", m.theme.Name))
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name)
		return m, nil
	case commands.PaletteActionConfirmSyncSetup:
		if action.Sync == nil {
			return m, nil
		}
		remote := strings.TrimSpace(action.Sync.Remote)
		branch := strings.TrimSpace(action.Sync.Branch)
		dir := strings.TrimSpace(action.Sync.Dir)
		if branch == "" {
			branch = "main"
		}
		if dir == "" {
			dir = m.settings.SyncDir
		}
		m.settings.SyncEnabled = remote != ""
		m.settings.SyncBackend = "git"
		m.settings.SyncRemote = remote
		m.settings.SyncBranch = branch
		m.settings.SyncDir = dir
		m.saveSettings()
		m.logs.Info("Sync settings saved")
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name)
		return m, nil
	case commands.PaletteActionCancelTheme:
		m.theme = *action.Theme
		return m, nil
	}
	return m, nil
}

func (m Model) syncNow() tea.Cmd {
	settings := m.settings
	return func() tea.Msg {
		if !settings.SyncEnabled || strings.TrimSpace(settings.SyncRemote) == "" {
			return syncCompletedMsg{Err: fmt.Errorf("sync is not configured; run Setup Sync first")}
		}
		if settings.SyncBackend != "" && settings.SyncBackend != "git" {
			return syncCompletedMsg{Err: fmt.Errorf("unsupported sync backend %q", settings.SyncBackend)}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		result, err := hnsync.SyncNow(ctx, hnsync.Options{Remote: settings.SyncRemote, Branch: settings.SyncBranch, Dir: settings.SyncDir})
		return syncCompletedMsg{SavedCount: result.SavedCount, ReadCount: result.ReadCount, DeletedCount: result.DeletedCount, Committed: result.Committed, Err: err}
	}
}

func (m Model) applyHideRead(hide bool) Model {
	m.settings.HideRead = hide
	m.saveSettings()
	for id, screen := range m.screens {
		if stories, ok := screen.(screens.Top); ok {
			stories.SetHideRead(hide)
			m.screens[id] = stories
		}
	}
	return m
}

func (m Model) saveSettings() {
	if err := m.configStore.Save(m.settings); err != nil {
		m.logs.Warn(fmt.Sprintf("Could not save config: %v", err))
	}
}

func (m Model) handleSidebarKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	idx := 0
	for i, id := range m.screenOrder {
		if id == m.activeScreen {
			idx = i
			break
		}
	}
	if key.Matches(msg, m.keys.Up) && idx > 0 {
		idx--
	} else if key.Matches(msg, m.keys.Down) && idx < len(m.screenOrder)-1 {
		idx++
	} else if !key.Matches(msg, m.keys.Enter) {
		return m, nil
	}
	m.switchScreen(m.screenOrder[idx])
	return m, m.initScreenIfNeeded(m.activeScreen)
}

func (m Model) updateScreen(id string, msg tea.Msg) (tea.Model, tea.Cmd) {
	active, ok := m.screens[id]
	if !ok {
		m.logs.Warn(fmt.Sprintf("Message targeted unknown screen: %s", id))
		return m, nil
	}
	updated, cmd := active.Update(msg)
	m.screens[id] = updated
	return m, cmd
}

func (m Model) initScreenIfNeeded(id string) tea.Cmd {
	screen, ok := m.screens[id]
	if !ok {
		return nil
	}
	if id == "saved" {
		return screen.Init()
	}
	if m.initialized[id] {
		return nil
	}
	m.initialized[id] = true
	return screen.Init()
}
