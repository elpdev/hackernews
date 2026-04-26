package commands

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/hackernews/pkg/theme"
)

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
			b.WriteString(m.renderCommandLine(t, command, i == m.selected) + "\n")
		}
	}

	return t.Modal.Width(paletteModalWidth).Render(b.String())
}

func (m PaletteModel) renderCommandLine(t theme.Theme, command Command, selected bool) string {
	if selected {
		return t.Selected.Render(m.commandLine(command, paletteSelectedContentWidth))
	}
	return t.Text.Render(m.commandLine(command, paletteContentWidth))
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
