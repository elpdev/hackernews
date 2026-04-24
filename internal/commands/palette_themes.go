package commands

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/internal/theme"
)

func (m PaletteModel) updateThemes(msg tea.KeyPressMsg) PaletteModel {
	switch msg.String() {
	case "esc", "backspace", "ctrl+h":
		if original, ok := m.themeByName(m.original); ok {
			m.action = PaletteAction{Type: PaletteActionCancelTheme, Theme: &original}
		}
		m.page = paletteRoot
		m.query = ""
		m.selected = 0
	case "up", "ctrl+p":
		if m.selected > 0 {
			m.selected--
			m.previewSelectedTheme()
		}
	case "down", "ctrl+n":
		if m.selected < len(m.themes)-1 {
			m.selected++
			m.previewSelectedTheme()
		}
	case "enter":
		if len(m.themes) > 0 {
			selected := m.themes[m.selected]
			m.action = PaletteAction{Type: PaletteActionConfirmTheme, Theme: &selected}
		}
	}
	return m
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

func (m *PaletteModel) openThemes() {
	m.page = paletteThemes
	m.query = ""
	m.selected = m.themeIndex(m.original)
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
