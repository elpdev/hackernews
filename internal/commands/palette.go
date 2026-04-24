package commands

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/elpdev/hackernews/internal/theme"
)

type PaletteModel struct {
	registry *Registry
	themes   []theme.Theme
	query    string
	selected int
	executed *Command
	action   PaletteAction
	page     palettePage
	parents  []string
	original string
	sync     SyncSetup
	field    int
}

type palettePage int

const (
	paletteRoot palettePage = iota
	paletteThemes
	paletteSyncSetup
)

const (
	paletteModalWidth           = 86
	paletteTitleWidth           = 18
	paletteContentWidth         = 80
	paletteSelectedContentWidth = 78
)

type SyncSetup struct {
	Remote string
	Branch string
	Dir    string
}

type PaletteAction struct {
	Type    PaletteActionType
	Command *Command
	Theme   *theme.Theme
	Sync    *SyncSetup
}

type PaletteActionType int

const (
	PaletteActionNone PaletteActionType = iota
	PaletteActionClose
	PaletteActionExecute
	PaletteActionPreviewTheme
	PaletteActionConfirmTheme
	PaletteActionCancelTheme
	PaletteActionConfirmSyncSetup
)

func NewPaletteModel(registry *Registry, themes []theme.Theme) PaletteModel {
	return PaletteModel{registry: registry, themes: themes}
}

func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd) {
	m.executed = nil
	m.action = PaletteAction{}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.page == paletteThemes {
			return m.updateThemes(msg), nil
		}
		if m.page == paletteSyncSetup {
			return m.updateSyncSetup(msg), nil
		}
		switch msg.String() {
		case "esc":
			if m.inCategory() {
				m.goBack()
				return m, nil
			}
			m.action = PaletteAction{Type: PaletteActionClose}
			return m, nil
		case "up", "ctrl+p":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil
		case "down", "ctrl+n":
			if m.selected < len(m.matches())-1 {
				m.selected++
			}
			return m, nil
		case "enter":
			matches := m.matches()
			if len(matches) == 0 {
				return m, nil
			}
			command := matches[m.selected]
			if m.registry.HasChildren(command.ID) {
				m.openCategory(command.ID)
				return m, nil
			}
			if command.ID == "themes" {
				m.openThemes()
				return m, nil
			}
			if command.ID == "setup-sync" {
				m.openSyncSetup()
				return m, nil
			}
			if command.Run == nil {
				return m, nil
			}
			m.executed = &command
			m.action = PaletteAction{Type: PaletteActionExecute, Command: &command}
			return m, nil
		case "backspace", "ctrl+h":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.selected = 0
			} else if m.inCategory() {
				m.goBack()
			}
			return m, nil
		case "space":
			m.query += " "
			m.selected = 0
			return m, nil
		}
		if len(msg.String()) == 1 {
			m.query += msg.String()
			m.selected = 0
			return m, nil
		}
	}
	if m.selected >= len(m.matches()) {
		m.selected = 0
	}
	return m, nil
}

func (m PaletteModel) updateThemes(msg tea.KeyPressMsg) PaletteModel {
	switch msg.String() {
	case "esc", "backspace", "ctrl+h":
		if original, ok := m.themeByName(m.original); ok {
			m.action = PaletteAction{Type: PaletteActionCancelTheme, Theme: &original}
		}
		m.page = paletteRoot
		m.query = ""
		m.selected = 0
		return m
	case "up", "ctrl+p":
		if m.selected > 0 {
			m.selected--
			m.previewSelectedTheme()
		}
		return m
	case "down", "ctrl+n":
		if m.selected < len(m.themes)-1 {
			m.selected++
			m.previewSelectedTheme()
		}
		return m
	case "enter":
		if len(m.themes) == 0 {
			return m
		}
		selected := m.themes[m.selected]
		m.action = PaletteAction{Type: PaletteActionConfirmTheme, Theme: &selected}
		return m
	}
	return m
}

func (m PaletteModel) updateSyncSetup(msg tea.KeyPressMsg) PaletteModel {
	switch msg.String() {
	case "esc":
		m.action = PaletteAction{Type: PaletteActionClose}
		return m
	case "up", "ctrl+p":
		if m.field > 0 {
			m.field--
		}
		return m
	case "down", "ctrl+n", "tab":
		if m.field < 2 {
			m.field++
		}
		return m
	case "enter":
		if m.field < 2 {
			m.field++
			return m
		}
		setup := m.sync
		m.action = PaletteAction{Type: PaletteActionConfirmSyncSetup, Sync: &setup}
		return m
	case "backspace", "ctrl+h":
		m.removeSyncRune()
		return m
	case "space":
		m.appendSyncRune(" ")
		return m
	}
	if len(msg.String()) == 1 {
		m.appendSyncRune(msg.String())
	}
	return m
}

func (m PaletteModel) View(t theme.Theme) string {
	if m.page == paletteThemes {
		return m.themeView(t)
	}
	if m.page == paletteSyncSetup {
		return m.syncSetupView(t)
	}

	matches := m.matches()
	var b strings.Builder
	b.WriteString(t.Title.Render(m.title()))
	b.WriteString("\n")
	query := m.query
	if query == "" {
		query = t.Muted.Render("type to search all commands...")
	}
	b.WriteString("> " + query)
	b.WriteString("\n\n")

	if len(matches) == 0 {
		b.WriteString(t.Muted.Render("No commands found"))
	} else {
		for i, command := range matches {
			if i == m.selected {
				line := m.commandLine(command, paletteSelectedContentWidth)
				line = t.Selected.Render(line)
				b.WriteString(line + "\n")
			} else {
				line := m.commandLine(command, paletteContentWidth)
				line = t.Text.Render(line)
				b.WriteString(line + "\n")
			}
		}
	}

	return t.Modal.Width(paletteModalWidth).Render(b.String())
}

func (m PaletteModel) commandLine(command Command, width int) string {
	title := command.Title
	if m.registry.HasChildren(command.ID) {
		title += " ›"
	}
	if width <= paletteTitleWidth {
		return truncatePaletteLine(title, width)
	}
	descriptionWidth := width - paletteTitleWidth - 1
	return fmt.Sprintf("%-*s %s", paletteTitleWidth, truncatePaletteLine(title, paletteTitleWidth), truncatePaletteLine(command.Description, descriptionWidth))
}

func truncatePaletteLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(line) <= width {
		return line
	}
	if width <= 3 {
		return takePaletteWidth(line, width)
	}
	return takePaletteWidth(line, width-3) + "..."
}

func takePaletteWidth(line string, width int) string {
	var b strings.Builder
	used := 0
	for _, r := range line {
		w := lipgloss.Width(string(r))
		if used+w > width {
			break
		}
		b.WriteRune(r)
		used += w
	}
	return b.String()
}

func (m PaletteModel) themeView(t theme.Theme) string {
	var b strings.Builder
	b.WriteString(t.Title.Render("Command Palette / Themes"))
	b.WriteString("\n")
	b.WriteString(t.Muted.Render("Move to preview, enter to select, esc to go back."))
	b.WriteString("\n\n")

	for i, candidate := range m.themes {
		line := candidate.Name
		if candidate.Name == t.Name {
			line += "  current"
		}
		if i == m.selected {
			line = t.Selected.Render(line)
		} else {
			line = t.Text.Render("  " + line)
		}
		b.WriteString(line + "\n")
	}

	return t.Modal.Width(62).Render(b.String())
}

func (m PaletteModel) syncSetupView(t theme.Theme) string {
	labels := []string{"Git remote", "Branch", "Sync dir"}
	values := []string{m.sync.Remote, m.sync.Branch, m.sync.Dir}
	var b strings.Builder
	b.WriteString(t.Title.Render("Command Palette / Setup Sync"))
	b.WriteString("\n")
	b.WriteString(t.Muted.Render("Enter advances fields, final enter saves, esc cancels."))
	b.WriteString("\n\n")
	for i, label := range labels {
		value := values[i]
		if value == "" {
			value = t.Muted.Render("empty")
		}
		line := fmt.Sprintf("%-11s %s", label+":", value)
		if i == m.field {
			line = t.Selected.Render(line)
		} else {
			line = t.Text.Render("  " + line)
		}
		b.WriteString(line + "\n")
	}
	return t.Modal.Width(72).Render(b.String())
}

func (m *PaletteModel) Reset(currentTheme string) {
	m.query = ""
	m.selected = 0
	m.executed = nil
	m.action = PaletteAction{}
	m.page = paletteRoot
	m.parents = nil
	m.original = currentTheme
	m.field = 0
}

func (m PaletteModel) ExecutedCommand() *Command { return m.executed }

func (m PaletteModel) Action() PaletteAction { return m.action }

func (m *PaletteModel) ClearAction() { m.action = PaletteAction{} }

func (m PaletteModel) matches() []Command {
	if strings.TrimSpace(m.query) != "" {
		matches := m.registry.Filter(m.query)
		commands := make([]Command, 0, len(matches))
		for _, command := range matches {
			if !m.registry.HasChildren(command.ID) {
				commands = append(commands, command)
			}
		}
		return commands
	}
	return m.registry.Children(m.currentParent())
}

func (m PaletteModel) currentParent() string {
	if len(m.parents) == 0 {
		return ""
	}
	return m.parents[len(m.parents)-1]
}

func (m PaletteModel) inCategory() bool { return len(m.parents) > 0 }

func (m PaletteModel) title() string {
	if len(m.parents) == 0 {
		return "Command Palette"
	}
	parts := []string{"Command Palette"}
	for _, id := range m.parents {
		if command, ok := m.registry.Find(id); ok {
			parts = append(parts, command.Title)
		}
	}
	return strings.Join(parts, " / ")
}

func (m *PaletteModel) openCategory(id string) {
	m.parents = append(m.parents, id)
	m.query = ""
	m.selected = 0
}

func (m *PaletteModel) goBack() {
	if len(m.parents) == 0 {
		return
	}
	m.parents = m.parents[:len(m.parents)-1]
	m.query = ""
	m.selected = 0
}

func (m *PaletteModel) openThemes() {
	m.page = paletteThemes
	m.query = ""
	m.selected = m.themeIndex(m.original)
}

func (m *PaletteModel) openSyncSetup() {
	m.page = paletteSyncSetup
	m.query = ""
	m.selected = 0
	m.field = 0
	if m.sync.Branch == "" {
		m.sync.Branch = "main"
	}
	if m.sync.Dir == "" {
		m.sync.Dir = "~/.hackernews/sync"
	}
}

func (m *PaletteModel) SetSyncSetup(setup SyncSetup) {
	m.sync = setup
}

func (m *PaletteModel) appendSyncRune(value string) {
	switch m.field {
	case 0:
		m.sync.Remote += value
	case 1:
		m.sync.Branch += value
	case 2:
		m.sync.Dir += value
	}
}

func (m *PaletteModel) removeSyncRune() {
	switch m.field {
	case 0:
		if len(m.sync.Remote) > 0 {
			m.sync.Remote = m.sync.Remote[:len(m.sync.Remote)-1]
		}
	case 1:
		if len(m.sync.Branch) > 0 {
			m.sync.Branch = m.sync.Branch[:len(m.sync.Branch)-1]
		}
	case 2:
		if len(m.sync.Dir) > 0 {
			m.sync.Dir = m.sync.Dir[:len(m.sync.Dir)-1]
		}
	}
}

func (m *PaletteModel) previewSelectedTheme() {
	if len(m.themes) == 0 {
		return
	}
	selected := m.themes[m.selected]
	m.action = PaletteAction{Type: PaletteActionPreviewTheme, Theme: &selected}
}

func (m PaletteModel) themeIndex(name string) int {
	for i, candidate := range m.themes {
		if candidate.Name == name {
			return i
		}
	}
	return 0
}

func (m PaletteModel) themeByName(name string) (theme.Theme, bool) {
	for _, candidate := range m.themes {
		if candidate.Name == name {
			return candidate, true
		}
	}
	return theme.Theme{}, false
}
