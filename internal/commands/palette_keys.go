package commands

import tea "charm.land/bubbletea/v2"

func (m PaletteModel) updateKey(msg tea.KeyPressMsg) PaletteModel {
	if m.page == paletteThemes {
		return m.updateThemes(msg)
	}
	if m.page == paletteSyncSetup {
		return m.updateSyncSetup(msg)
	}
	switch msg.String() {
	case "esc":
		if m.inCategory() {
			m.goBack()
			return m
		}
		m.action = PaletteAction{Type: PaletteActionClose}
	case "up", "ctrl+p":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "ctrl+n":
		if m.selected < len(m.matches())-1 {
			m.selected++
		}
	case "enter":
		return m.selectCurrentCommand()
	case "backspace", "ctrl+h":
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.selected = 0
		} else if m.inCategory() {
			m.goBack()
		}
	case "space":
		m.query += " "
		m.selected = 0
	default:
		if len(msg.String()) == 1 {
			m.query += msg.String()
			m.selected = 0
		}
	}
	return m
}

func (m PaletteModel) selectCurrentCommand() PaletteModel {
	matches := m.matches()
	if len(matches) == 0 {
		return m
	}
	command := matches[m.selected]
	if m.registry.HasChildren(command.ID) {
		m.openCategory(command.ID)
		return m
	}
	if command.ID == "themes" {
		m.openThemes()
		return m
	}
	if command.ID == "setup-sync" {
		m.openSyncSetup()
		return m
	}
	if command.Run == nil {
		return m
	}
	m.executed = &command
	m.action = PaletteAction{Type: PaletteActionExecute, Command: &command}
	return m
}
