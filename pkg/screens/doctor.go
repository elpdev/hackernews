package screens

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/hackernews/internal/doctor"
	"github.com/elpdev/hackernews/pkg/config"
)

type Doctor struct {
	settings config.Settings
	checks   []doctor.Check
	loading  string
	err      string
	returnTo string
}

type doctorLoadedMsg struct {
	screenID string
	checks   []doctor.Check
	err      error
}

func (m doctorLoadedMsg) TargetScreenID() string { return m.screenID }

func NewDoctor(settings config.Settings) Doctor {
	return Doctor{settings: settings}
}

func (d Doctor) WithSettings(settings config.Settings) Doctor {
	d.settings = settings
	return d
}

func (d Doctor) Open(returnTo string, settings config.Settings) (Doctor, tea.Cmd) {
	d.returnTo = returnTo
	d.settings = settings
	d.loading = "Running checks..."
	d.err = ""
	d.checks = nil
	return d, d.run()
}

func (d Doctor) Init() tea.Cmd { return nil }

func (d Doctor) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case doctorLoadedMsg:
		d.loading = ""
		if msg.err != nil {
			d.err = msg.err.Error()
		} else {
			d.err = ""
		}
		d.checks = msg.checks
		return d, nil
	case tea.KeyPressMsg:
		return d.handleKey(msg)
	}
	return d, nil
}

func (d Doctor) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Doctor"))
	b.WriteString("\n")
	b.WriteString("r rerun | esc back\n")
	if d.loading != "" {
		b.WriteString(d.loading + "\n")
	}
	if d.err != "" {
		b.WriteString(d.err + "\n")
	}
	b.WriteString("\n")
	for _, check := range d.checks {
		line := fmt.Sprintf("%-5s %-18s %s", check.Status.String(), check.Name, check.Message)
		b.WriteString(truncateScreen(line, width) + "\n")
	}
	return b.String()
}

func (d Doctor) Title() string { return "Doctor" }

func (d Doctor) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rerun")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func (d Doctor) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "r":
		d.loading = "Running checks..."
		d.err = ""
		return d, d.run()
	case "esc":
		dest := d.returnTo
		if dest == "" || dest == "doctor" {
			dest = "top"
		}
		return d, func() tea.Msg { return NavigateMsg{ScreenID: dest} }
	}
	return d, nil
}

func (d Doctor) run() tea.Cmd {
	settings := d.settings
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		checks := doctor.Run(ctx, doctor.Options{SyncEnabled: settings.SyncEnabled, SyncRemote: settings.SyncRemote, SyncBranch: settings.SyncBranch, SyncDir: settings.SyncDir})
		return doctorLoadedMsg{screenID: "doctor", checks: checks}
	}
}
