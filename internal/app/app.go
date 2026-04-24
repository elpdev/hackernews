package app

import (
	"fmt"
	"sort"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/internal/commands"
	"github.com/elpdev/hackernews/internal/debug"
	"github.com/elpdev/hackernews/internal/saved"
	"github.com/elpdev/hackernews/internal/screens"
	"github.com/elpdev/hackernews/internal/theme"
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

	showSidebar        bool
	showHelp           bool
	showCommandPalette bool

	focus FocusArea
	keys  KeyMap

	commands       *commands.Registry
	commandPalette commands.PaletteModel
	savedStore     saved.Store

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

	m := Model{
		activeScreen: defaultScreen,
		screens:      make(map[string]screens.Screen),
		showSidebar:  false,
		focus:        FocusMain,
		keys:         DefaultKeyMap(),
		commands:     commands.NewRegistry(),
		savedStore:   savedStore,
		theme:        theme.Phosphor(),
		logs:         log,
		meta:         meta,
	}

	m.registerScreens()
	m.registerCommands()
	m.commandPalette = commands.NewPaletteModel(m.commands, theme.BuiltIns())
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{func() tea.Msg { return tea.RequestWindowSize() }}
	for _, screen := range m.screens {
		cmds = append(cmds, screen.Init())
	}
	return tea.Batch(cmds...)
}

func (m *Model) registerScreens() {
	m.screens["top"] = screens.NewTop(m.savedStore)
	m.screens["saved"] = screens.NewSaved(m.savedStore)
	m.refreshScreenOrder()
}

func (m *Model) refreshScreenOrder() {
	m.screenOrder = m.screenOrder[:0]
	for id := range m.screens {
		m.screenOrder = append(m.screenOrder, id)
	}
	sort.Strings(m.screenOrder)
	preferred := []string{"top", "saved"}
	ordered := make([]string, 0, len(m.screenOrder))
	seen := make(map[string]bool)
	for _, id := range preferred {
		if _, ok := m.screens[id]; ok {
			ordered = append(ordered, id)
			seen[id] = true
		}
	}
	for _, id := range m.screenOrder {
		if !seen[id] {
			ordered = append(ordered, id)
		}
	}
	m.screenOrder = ordered
}

func (m *Model) registerCommands() {
	m.commands.Register(commands.Command{ID: "go-top", Title: "Go to Top Stories", Description: "Open Hacker News top stories", Keywords: []string{"top", "hacker news", "stories", "news"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"top"} } }})
	m.commands.Register(commands.Command{ID: "go-saved", Title: "Go to Saved", Description: "Open saved articles", Keywords: []string{"saved", "articles", "bookmarks", "offline"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"saved"} } }})
	m.commands.Register(commands.Command{ID: "toggle-sidebar", Title: "Toggle Sidebar", Description: "Show or hide sidebar navigation", Keywords: []string{"sidebar", "layout"}, Run: func() tea.Cmd { return func() tea.Msg { return toggleSidebarMsg{} } }})
	m.commands.Register(commands.Command{ID: "themes", Title: "Themes", Description: "Preview and select a theme", Keywords: []string{"theme", "themes", "appearance", "colors", "dark", "muted", "phosphor", "miami"}})
	m.commands.Register(commands.Command{ID: "quit", Title: "Quit", Description: "Exit Hackernews", Keywords: []string{"exit", "close"}, Run: func() tea.Cmd { return func() tea.Msg { return quitMsg{} } }})
}

func (m *Model) switchScreen(id string) {
	if _, ok := m.screens[id]; !ok {
		m.logs.Warn(fmt.Sprintf("Unknown screen requested: %s", id))
		return
	}
	if m.activeScreen != id {
		m.activeScreen = id
		m.logs.Info(fmt.Sprintf("Screen changed to %s", id))
	}
}

func (m Model) CurrentScreenID() string { return m.activeScreen }

func (m Model) SwitchScreenForTest(id string) Model {
	m.switchScreen(id)
	return m
}
