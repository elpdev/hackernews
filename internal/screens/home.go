package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
)

type Home struct{}

func NewHome() Home { return Home{} }

func (h Home) Init() tea.Cmd { return nil }

func (h Home) Update(msg tea.Msg) (Screen, tea.Cmd) { return h, nil }

func (h Home) View(width, height int) string {
	content := strings.Join([]string{
		"Hackernews",
		"",
		"A terminal UI for reading Hacker News top stories and extracted article text.",
		"",
		"Open Top Stories from the sidebar to browse the current front page.",
		"Press ctrl+k to open command palette.",
		"Press ? for help.",
	}, "\n")
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

func (h Home) Title() string { return "Home" }

func (h Home) KeyBindings() []key.Binding { return nil }
