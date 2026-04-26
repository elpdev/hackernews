package commands

import (
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/pkg/theme"
	"github.com/elpdev/tuipalette"
)

const (
	pageThemes    = "themes"
	pageSyncSetup = "setup-sync"
)

type SyncSetup struct {
	Remote string
	Branch string
	Dir    string
}

type PaletteModel struct {
	registry *Registry
	themes   []theme.Theme
	inner    tuipalette.PaletteModel
	original string
	sync     SyncSetup
	executed *Command
	action   PaletteAction
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
	m := PaletteModel{registry: registry, themes: themes}
	m.rebuildInner()
	return m
}

func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd) {
	m.executed = nil
	m.action = PaletteAction{}
	inner, cmd := m.inner.Update(msg)
	m.inner = inner
	m.translateAction(m.inner.Action())
	return m, cmd
}

func (m PaletteModel) View(t theme.Theme) string {
	m.inner.SetStyles(stylesFromTheme(t))
	return m.inner.View()
}

func (m *PaletteModel) Reset(currentTheme string) {
	m.original = currentTheme
	m.executed = nil
	m.action = PaletteAction{}
	m.rebuildInner()
	m.inner.Reset(tuipalette.Context{})
}

func (m PaletteModel) ExecutedCommand() *Command { return m.executed }

func (m PaletteModel) Action() PaletteAction { return m.action }

func (m *PaletteModel) ClearAction() {
	m.action = PaletteAction{}
	m.inner.ClearAction()
}

func (m *PaletteModel) SetSyncSetup(setup SyncSetup) {
	m.sync = setup
	m.rebuildInner()
	m.inner.Reset(tuipalette.Context{})
}

func (m *PaletteModel) rebuildInner() {
	registry := tuipalette.NewRegistry()
	for _, cmd := range m.registry.List() {
		command := cmd
		opensPage := ""
		switch command.ID {
		case pageThemes:
			opensPage = pageThemes
		case pageSyncSetup:
			opensPage = pageSyncSetup
		}
		registry.Register(tuipalette.Command{
			ID:          command.ID,
			ParentID:    command.ParentID,
			Title:       command.Title,
			Description: command.Description,
			Keywords:    command.Keywords,
			Order:       command.Order,
			OpensPage:   opensPage,
			Run:         command.Run,
		})
	}

	m.inner = tuipalette.NewPaletteModel(registry, tuipalette.Options{
		Title:        "Command Palette",
		Placeholder:  "type to search all commands...",
		Width:        86,
		Hierarchical: true,
		Pages: map[string]tuipalette.Page{
			pageThemes:    newThemePage(m.themes, m.original),
			pageSyncSetup: newSyncPage(m.sync),
		},
	})
}

func (m *PaletteModel) translateAction(action tuipalette.PaletteAction) {
	switch action.Type {
	case tuipalette.PaletteActionClose:
		m.action = PaletteAction{Type: PaletteActionClose}
	case tuipalette.PaletteActionExecute:
		if action.Command == nil {
			return
		}
		command, ok := m.registry.Find(action.Command.ID)
		if !ok {
			return
		}
		m.executed = &command
		m.action = PaletteAction{Type: PaletteActionExecute, Command: &command}
	case tuipalette.PaletteActionBack:
		if action.Page == "theme-cancel" {
			if selected, ok := action.Data.(theme.Theme); ok {
				m.action = PaletteAction{Type: PaletteActionCancelTheme, Theme: &selected}
			}
		}
	case tuipalette.PaletteActionPage:
		switch action.Page {
		case "theme-preview":
			if selected, ok := action.Data.(theme.Theme); ok {
				m.action = PaletteAction{Type: PaletteActionPreviewTheme, Theme: &selected}
			}
		case "theme-confirm":
			if selected, ok := action.Data.(theme.Theme); ok {
				m.action = PaletteAction{Type: PaletteActionConfirmTheme, Theme: &selected}
			}
		case "sync-confirm":
			if fields, ok := action.Data.([]tuipalette.FormField); ok {
				setup := syncSetupFromFields(fields)
				m.action = PaletteAction{Type: PaletteActionConfirmSyncSetup, Sync: &setup}
			}
		}
	}
}

func stylesFromTheme(t theme.Theme) tuipalette.Styles {
	return tuipalette.Styles{
		Modal:    t.Modal,
		Title:    t.Title,
		Text:     t.Text,
		Muted:    t.Muted,
		Selected: t.Selected,
		Accent:   t.Info,
	}
}

func newThemePage(themes []theme.Theme, original string) tuipalette.SelectPage {
	items := make([]tuipalette.SelectItem, 0, len(themes))
	selected := 0
	var cancelData any
	for i, candidate := range themes {
		current := candidate.Name == original
		if current {
			selected = i
			cancelData = candidate
		}
		items = append(items, tuipalette.SelectItem{Label: candidate.Name, Current: current, Value: candidate})
	}
	return tuipalette.NewSelectPage(tuipalette.SelectPageOptions{
		Title:       "Command Palette / Themes",
		Subtitle:    "Move to preview, enter to select, esc to go back.",
		Items:       items,
		Selected:    selected,
		PreviewPage: "theme-preview",
		ConfirmPage: "theme-confirm",
		CancelPage:  "theme-cancel",
		CancelData:  cancelData,
		Width:       62,
	})
}

func newSyncPage(setup SyncSetup) tuipalette.FormPage {
	if setup.Branch == "" {
		setup.Branch = "main"
	}
	if setup.Dir == "" {
		setup.Dir = "~/.hackernews/sync"
	}
	return tuipalette.NewFormPage(tuipalette.FormPageOptions{
		Title:       "Command Palette / Setup Sync",
		Subtitle:    "Enter advances fields, final enter saves, esc cancels.",
		ConfirmPage: "sync-confirm",
		Width:       72,
		Fields: []tuipalette.FormField{
			{Label: "Git remote", Value: setup.Remote},
			{Label: "Branch", Value: setup.Branch},
			{Label: "Sync dir", Value: setup.Dir},
		},
	})
}

func syncSetupFromFields(fields []tuipalette.FormField) SyncSetup {
	setup := SyncSetup{}
	if len(fields) > 0 {
		setup.Remote = fields[0].Value
	}
	if len(fields) > 1 {
		setup.Branch = fields[1].Value
	}
	if len(fields) > 2 {
		setup.Dir = fields[2].Value
	}
	return setup
}
