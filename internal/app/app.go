package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	hackernews "github.com/elpdev/hackernews"
	"github.com/elpdev/hackernews/internal/debug"
	"github.com/elpdev/hackernews/pkg/commands"
	"github.com/elpdev/hackernews/pkg/config"
	"github.com/elpdev/hackernews/pkg/history"
	"github.com/elpdev/hackernews/pkg/saved"
	"github.com/elpdev/hackernews/pkg/screens"
	"github.com/elpdev/hackernews/pkg/theme"
)

const defaultScreen = "top"

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

type Model struct {
	width  int
	height int

	activeScreen string
	screens      map[string]screens.Screen
	screenOrder  []string
	initialized  map[string]bool

	showSidebar        bool
	showHelp           bool
	showCommandPalette bool

	focus FocusArea
	keys  KeyMap

	commands       *commands.Registry
	commandPalette commands.PaletteModel
	savedStore     saved.Store
	historyStore   history.Store
	configStore    config.Store
	settings       config.Settings
	inlineMediaSeq string

	theme theme.Theme
	logs  *debug.Log
	meta  BuildInfo
}

func New(meta BuildInfo) Model {
	log := debug.NewLog()
	log.Info("App started")
	var savedStore saved.Store
	if path, err := saved.DefaultPath(); err != nil {
		log.Warn(fmt.Sprintf("Saved article store unavailable: %v", err))
	} else {
		savedStore = saved.NewJSONStore(path)
	}
	var historyStore history.Store
	if path, err := history.DefaultPath(); err != nil {
		log.Warn(fmt.Sprintf("Read history unavailable: %v", err))
	} else {
		historyStore = history.NewJSONStore(path)
	}
	settings := config.Defaults()
	var configStore config.Store
	if meta.Version != "test" {
		if path, err := config.DefaultPath(); err != nil {
			log.Warn(fmt.Sprintf("Config unavailable: %v", err))
		} else {
			configStore = config.NewStore(path)
			if loaded, err := configStore.Load(); err != nil {
				log.Warn(fmt.Sprintf("Could not load config: %v", err))
			} else {
				settings = loaded
			}
		}
	}

	m := Model{
		activeScreen: settings.DefaultFeed,
		screens:      make(map[string]screens.Screen),
		initialized:  make(map[string]bool),
		showSidebar:  settings.ShowSidebar,
		focus:        FocusMain,
		keys:         DefaultKeyMap(),
		commands:     commands.NewRegistry(),
		savedStore:   savedStore,
		historyStore: historyStore,
		configStore:  configStore,
		settings:     settings,
		theme:        themeByName(settings.ThemeName),
		logs:         log,
		meta:         meta,
	}

	m.registerScreens()
	if _, ok := m.screens[m.activeScreen]; !ok {
		m.activeScreen = defaultScreen
	}
	m.registerCommands()
	m.commandPalette = commands.NewPaletteModel(m.commands, theme.BuiltIns())
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{func() tea.Msg { return tea.RequestWindowSize() }}
	if active, ok := m.screens[m.activeScreen]; ok {
		m.initialized[m.activeScreen] = true
		cmds = append(cmds, active.Init())
	}
	return tea.Batch(cmds...)
}

func (m *Model) registerScreens() {
	module := hackernews.New(hackernews.Options{SavedStore: m.savedStore, HistoryStore: m.historyStore, Settings: m.settings, Themes: theme.BuiltIns()})
	m.screens = module.Screens()
	m.screenOrder = module.ScreenOrder()
}

func (m *Model) registerCommands() {
	module := hackernews.New(hackernews.Options{SavedStore: m.savedStore, HistoryStore: m.historyStore, Settings: m.settings, Themes: theme.BuiltIns()})
	for _, spec := range module.Commands() {
		m.commands.Register(m.commandFromSpec(spec))
	}
	m.commands.Register(commands.Command{ID: "view", Title: "View", Description: "Layout and reading options", Keywords: []string{"layout", "sidebar", "read"}, Order: 30})
	m.commands.Register(commands.Command{ID: "sync", Title: "Sync", Description: "Git sync actions and setup", Keywords: []string{"git", "remote", "backup"}, Order: 40})
	m.commands.Register(commands.Command{ID: "appearance", Title: "Appearance", Description: "Themes and visual settings", Keywords: []string{"theme", "colors"}, Order: 50})
	m.commands.Register(commands.Command{ID: "toggle-sidebar", ParentID: "view", Title: "Toggle Sidebar", Description: "Show or hide sidebar navigation", Keywords: []string{"sidebar", "layout"}, Order: 10, Run: func() tea.Cmd { return func() tea.Msg { return toggleSidebarMsg{} } }})
	m.commands.Register(commands.Command{ID: "toggle-hide-read", ParentID: "view", Title: "Toggle Hide Read", Description: "Show or hide read stories", Keywords: []string{"read", "visited", "hide"}, Order: 20, Run: func() tea.Cmd { return func() tea.Msg { return toggleHideReadMsg{} } }})
	m.commands.Register(commands.Command{ID: "sync-now", ParentID: "sync", Title: "Sync Now", Description: "Manually sync saved and read articles", Keywords: []string{"sync", "git", "saved", "read"}, Order: 10, Run: func() tea.Cmd { return func() tea.Msg { return syncNowMsg{} } }})
	m.commands.Register(commands.Command{ID: "setup-sync", ParentID: "sync", Title: "Setup Sync", Description: "Configure Git sync settings", Keywords: []string{"sync", "git", "remote", "setup", "config"}, Order: 20})
	m.commands.Register(commands.Command{ID: "themes", ParentID: "appearance", Title: "Themes", Description: "Preview and select a theme", Keywords: []string{"theme", "themes", "appearance", "colors", "dark", "muted", "phosphor", "synthwave", "neon", "retro"}, Order: 10})
	m.commands.Register(commands.Command{ID: "quit", ParentID: "system", Title: "Quit", Description: "Exit Hackernews", Keywords: []string{"exit", "close"}, Order: 10, Run: func() tea.Cmd { return func() tea.Msg { return quitMsg{} } }})
}

func (m Model) commandFromSpec(spec hackernews.CommandSpec) commands.Command {
	command := commands.Command{ID: spec.ID, ParentID: spec.ParentID, Title: spec.Title, Description: spec.Description, Keywords: spec.Keywords, Order: spec.Order}
	if spec.ID == "doctor" {
		command.Run = func() tea.Cmd { return func() tea.Msg { return openDoctorMsg{} } }
		return command
	}
	if spec.ScreenID != "" {
		screenID := spec.ScreenID
		command.Run = func() tea.Cmd { return func() tea.Msg { return routeMsg{screenID} } }
	}
	return command
}

func (m *Model) switchScreen(id string) {
	if _, ok := m.screens[id]; !ok {
		m.logs.Warn(fmt.Sprintf("Unknown screen requested: %s", id))
		return
	}
	if m.activeScreen != id {
		m.activeScreen = id
		if id != "comments" && id != "search" && id != "settings" && id != "doctor" {
			m.settings.DefaultFeed = id
			m.saveSettings()
		}
		m.logs.Info(fmt.Sprintf("Screen changed to %s", id))
	}
}

func (m Model) CurrentScreenID() string { return m.activeScreen }

func (m Model) SwitchScreenForTest(id string) Model {
	m.switchScreen(id)
	return m
}

func themeByName(name string) theme.Theme {
	for _, candidate := range theme.BuiltIns() {
		if candidate.Name == name {
			return candidate
		}
	}
	return theme.Phosphor()
}
