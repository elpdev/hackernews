package sidebar

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/elpdev/hackernews/internal/theme"
)

type Item struct {
	ID    string
	Title string
}

type Model struct {
	Items    []Item
	ActiveID string
	Focused  bool
}

func View(m Model, width, height int, t theme.Theme) string {
	frameWidth, _ := t.Sidebar.GetFrameSize()
	contentWidth := max(0, width-frameWidth)
	var b strings.Builder
	if m.Focused {
		b.WriteString(t.Title.Render("Navigation"))
	} else {
		b.WriteString(t.Muted.Render("Navigation"))
	}
	b.WriteString("\n\n")
	for _, item := range m.Items {
		if item.ID == m.ActiveID {
			b.WriteString(renderRow(t.Selected.Padding(0, 0), item.Title, contentWidth))
		} else {
			b.WriteString(renderRow(t.Text, item.Title, contentWidth))
		}
		b.WriteString("\n")
	}
	return t.Sidebar.Width(max(0, width)).Height(max(0, height)).Align(lipgloss.Left).Render(b.String())
}

func renderRow(style lipgloss.Style, content string, width int) string {
	return style.Width(width).Align(lipgloss.Left).Render(content)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
