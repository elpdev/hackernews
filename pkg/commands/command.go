package commands

import tea "charm.land/bubbletea/v2"

type Command struct {
	ID          string
	ParentID    string
	Title       string
	Description string
	Keywords    []string
	Order       int
	Run         func() tea.Cmd
}
