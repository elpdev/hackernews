package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/hackernews/pkg/config"
	"github.com/elpdev/hackernews/pkg/theme"
)

type Settings struct {
	settings config.Settings
	themes   []theme.Theme
	selected int
	status   string
}

type settingRow int

const (
	settingTheme settingRow = iota
	settingSidebar
	settingDefaultFeed
	settingSort
	settingHideRead
	settingSyncEnabled
	settingSyncRemote
	settingSyncBranch
	settingSyncDir
	settingCount
)

func NewSettings(settings config.Settings, themes []theme.Theme) Settings {
	return Settings{settings: settings, themes: themes}
}

func (s Settings) WithSettings(settings config.Settings) Settings {
	s.settings = settings
	return s
}

func (s Settings) Init() tea.Cmd { return nil }

func (s Settings) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return s.handleKey(msg)
	}
	return s, nil
}

func (s Settings) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Settings"))
	b.WriteString("\n")
	b.WriteString("enter cycles editable settings | r resets selected | sync fields use Setup Sync\n")
	if s.status != "" {
		b.WriteString(s.status + "\n")
	}
	b.WriteString("\n")

	rows := s.rows()
	for i, row := range rows {
		line := fmt.Sprintf("%-16s %s", row.label, row.value)
		if !row.editable {
			line += "  " + lipgloss.NewStyle().Faint(true).Render("display only")
		}
		line = truncateScreen(line, width)
		if i == s.selected {
			line = lipgloss.NewStyle().Reverse(true).Render(padLine(line, width))
		} else {
			line = "  " + line
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (s Settings) Title() string { return "Settings" }

func (s Settings) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "change")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reset selected")),
	}
}

func (s Settings) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if s.selected > 0 {
			s.selected--
		}
		return s, nil
	case "down", "j":
		if s.selected < int(settingCount)-1 {
			s.selected++
		}
		return s, nil
	case "enter":
		updated, changed := s.cycleSelected()
		if !changed {
			s.status = "Use Setup Sync in the command palette to edit sync fields"
			return s, nil
		}
		s.settings = updated
		s.status = "Setting updated"
		return s, func() tea.Msg { return SettingsChangedMsg{Settings: updated} }
	case "r":
		updated, changed := s.resetSelected()
		if !changed {
			s.status = "Nothing to reset for this setting"
			return s, nil
		}
		s.settings = updated
		s.status = "Setting reset"
		return s, func() tea.Msg { return SettingsChangedMsg{Settings: updated} }
	}
	return s, nil
}

type settingsRowView struct {
	label    string
	value    string
	editable bool
}

func (s Settings) rows() []settingsRowView {
	return []settingsRowView{
		{label: "Theme", value: s.settings.ThemeName, editable: true},
		{label: "Sidebar", value: boolLabel(s.settings.ShowSidebar), editable: true},
		{label: "Default feed", value: s.settings.DefaultFeed, editable: true},
		{label: "Sort", value: nonEmpty(s.settings.SortMode, "default"), editable: true},
		{label: "Hide read", value: boolLabel(s.settings.HideRead), editable: true},
		{label: "Sync enabled", value: boolLabel(s.settings.SyncEnabled), editable: true},
		{label: "Sync remote", value: nonEmpty(s.settings.SyncRemote, "not configured")},
		{label: "Sync branch", value: nonEmpty(s.settings.SyncBranch, "main")},
		{label: "Sync dir", value: nonEmpty(s.settings.SyncDir, "not configured")},
	}
}

func (s Settings) cycleSelected() (config.Settings, bool) {
	updated := s.settings
	switch settingRow(s.selected) {
	case settingTheme:
		updated.ThemeName = s.nextTheme()
	case settingSidebar:
		updated.ShowSidebar = !updated.ShowSidebar
	case settingDefaultFeed:
		updated.DefaultFeed = nextString(updated.DefaultFeed, []string{"top", "new", "best", "ask", "show", "jobs", "saved"})
	case settingSort:
		updated.SortMode = nextString(updated.SortMode, []string{"", "recent", "points"})
	case settingHideRead:
		updated.HideRead = !updated.HideRead
	case settingSyncEnabled:
		updated.SyncEnabled = !updated.SyncEnabled
	default:
		return updated, false
	}
	return updated, true
}

func (s Settings) resetSelected() (config.Settings, bool) {
	updated := s.settings
	defaults := config.Defaults()
	switch settingRow(s.selected) {
	case settingTheme:
		updated.ThemeName = defaults.ThemeName
	case settingSidebar:
		updated.ShowSidebar = defaults.ShowSidebar
	case settingDefaultFeed:
		updated.DefaultFeed = defaults.DefaultFeed
	case settingSort:
		updated.SortMode = defaults.SortMode
	case settingHideRead:
		updated.HideRead = defaults.HideRead
	case settingSyncEnabled:
		updated.SyncEnabled = defaults.SyncEnabled
	case settingSyncRemote:
		updated.SyncRemote = defaults.SyncRemote
	case settingSyncBranch:
		updated.SyncBranch = defaults.SyncBranch
	case settingSyncDir:
		updated.SyncDir = defaults.SyncDir
	default:
		return updated, false
	}
	return updated, true
}

func (s Settings) nextTheme() string {
	if len(s.themes) == 0 {
		return s.settings.ThemeName
	}
	names := make([]string, 0, len(s.themes))
	for _, candidate := range s.themes {
		names = append(names, candidate.Name)
	}
	return nextString(s.settings.ThemeName, names)
}

func nextString(current string, values []string) string {
	for i, value := range values {
		if value == current {
			return values[(i+1)%len(values)]
		}
	}
	return values[0]
}

func boolLabel(value bool) string {
	if value {
		return "on"
	}
	return "off"
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
