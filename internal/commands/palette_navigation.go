package commands

import "strings"

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
