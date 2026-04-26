package commands

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/pkg/theme"
)

func (m PaletteModel) updateSyncSetup(msg tea.KeyPressMsg) PaletteModel {
	switch msg.String() {
	case "esc":
		m.action = PaletteAction{Type: PaletteActionClose}
	case "up", "ctrl+p":
		if m.field > 0 {
			m.field--
		}
	case "down", "ctrl+n", "tab":
		if m.field < 2 {
			m.field++
		}
	case "enter":
		if m.field < 2 {
			m.field++
		} else {
			setup := m.sync
			m.action = PaletteAction{Type: PaletteActionConfirmSyncSetup, Sync: &setup}
		}
	case "backspace", "ctrl+h":
		m.removeSyncRune()
	case "space":
		m.appendSyncRune(" ")
	default:
		if len(msg.String()) == 1 {
			m.appendSyncRune(msg.String())
		}
	}
	return m
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
