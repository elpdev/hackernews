package commands

import (
	"fmt"
	"strings"

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
			if setup, ok := action.Data.(SyncSetup); ok {
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

type themePage struct {
	themes   []theme.Theme
	original string
	selected int
}

func newThemePage(themes []theme.Theme, original string) themePage {
	page := themePage{themes: append([]theme.Theme(nil), themes...), original: original}
	page.selected = page.themeIndex(original)
	return page
}

func (p themePage) Update(msg tea.KeyPressMsg) (tuipalette.Page, tuipalette.PaletteAction) {
	switch msg.String() {
	case "esc", "backspace", "ctrl+h":
		if original, ok := p.themeByName(p.original); ok {
			return p, tuipalette.PaletteAction{Type: tuipalette.PaletteActionBack, Page: "theme-cancel", Data: original}
		}
		return p, tuipalette.PaletteAction{Type: tuipalette.PaletteActionBack, Page: "theme-cancel"}
	case "up", "ctrl+p":
		if p.selected > 0 {
			p.selected--
			return p, p.previewSelectedTheme()
		}
	case "down", "ctrl+n":
		if p.selected < len(p.themes)-1 {
			p.selected++
			return p, p.previewSelectedTheme()
		}
	case "enter":
		if len(p.themes) > 0 {
			selected := p.themes[p.selected]
			return p, tuipalette.PaletteAction{Type: tuipalette.PaletteActionPage, Page: "theme-confirm", Data: selected}
		}
	}
	return p, tuipalette.PaletteAction{}
}

func (p themePage) View(styles tuipalette.Styles, width int) string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("Command Palette / Themes"))
	b.WriteString("\n")
	b.WriteString(styles.Muted.Render("Move to preview, enter to select, esc to go back."))
	b.WriteString("\n\n")

	for i, candidate := range p.themes {
		line := candidate.Name
		if candidate.Name == p.original {
			line += "  current"
		}
		if i == p.selected {
			line = styles.Selected.Render(line)
		} else {
			line = styles.Text.Render("  " + line)
		}
		b.WriteString(line + "\n")
	}

	return styles.Modal.Width(62).Render(b.String())
}

func (p themePage) Reset() {
	p.selected = p.themeIndex(p.original)
}

func (p themePage) previewSelectedTheme() tuipalette.PaletteAction {
	if len(p.themes) == 0 {
		return tuipalette.PaletteAction{}
	}
	selected := p.themes[p.selected]
	return tuipalette.PaletteAction{Type: tuipalette.PaletteActionPage, Page: "theme-preview", Data: selected}
}

func (p themePage) themeIndex(name string) int {
	for i, candidate := range p.themes {
		if candidate.Name == name {
			return i
		}
	}
	return 0
}

func (p themePage) themeByName(name string) (theme.Theme, bool) {
	for _, candidate := range p.themes {
		if candidate.Name == name {
			return candidate, true
		}
	}
	return theme.Theme{}, false
}

type syncPage struct {
	setup SyncSetup
	field int
}

func newSyncPage(setup SyncSetup) syncPage {
	if setup.Branch == "" {
		setup.Branch = "main"
	}
	if setup.Dir == "" {
		setup.Dir = "~/.hackernews/sync"
	}
	return syncPage{setup: setup}
}

func (p syncPage) Update(msg tea.KeyPressMsg) (tuipalette.Page, tuipalette.PaletteAction) {
	switch msg.String() {
	case "esc":
		return p, tuipalette.PaletteAction{Type: tuipalette.PaletteActionClose}
	case "up", "ctrl+p":
		if p.field > 0 {
			p.field--
		}
	case "down", "ctrl+n", "tab":
		if p.field < 2 {
			p.field++
		}
	case "enter":
		if p.field < 2 {
			p.field++
		} else {
			return p, tuipalette.PaletteAction{Type: tuipalette.PaletteActionPage, Page: "sync-confirm", Data: p.setup}
		}
	case "backspace", "ctrl+h":
		p.removeRune()
	case "space":
		p.appendRune(" ")
	default:
		if len(msg.String()) == 1 {
			p.appendRune(msg.String())
		}
	}
	return p, tuipalette.PaletteAction{}
}

func (p syncPage) View(styles tuipalette.Styles, width int) string {
	labels := []string{"Git remote", "Branch", "Sync dir"}
	values := []string{p.setup.Remote, p.setup.Branch, p.setup.Dir}
	var b strings.Builder
	b.WriteString(styles.Title.Render("Command Palette / Setup Sync"))
	b.WriteString("\n")
	b.WriteString(styles.Muted.Render("Enter advances fields, final enter saves, esc cancels."))
	b.WriteString("\n\n")
	for i, label := range labels {
		value := values[i]
		if value == "" {
			value = styles.Muted.Render("empty")
		}
		line := fmt.Sprintf("%-11s %s", label+":", value)
		if i == p.field {
			line = styles.Selected.Render(line)
		} else {
			line = styles.Text.Render("  " + line)
		}
		b.WriteString(line + "\n")
	}
	return styles.Modal.Width(72).Render(b.String())
}

func (p syncPage) Reset() {
	p.field = 0
}

func (p *syncPage) appendRune(value string) {
	switch p.field {
	case 0:
		p.setup.Remote += value
	case 1:
		p.setup.Branch += value
	case 2:
		p.setup.Dir += value
	}
}

func (p *syncPage) removeRune() {
	switch p.field {
	case 0:
		if len(p.setup.Remote) > 0 {
			p.setup.Remote = p.setup.Remote[:len(p.setup.Remote)-1]
		}
	case 1:
		if len(p.setup.Branch) > 0 {
			p.setup.Branch = p.setup.Branch[:len(p.setup.Branch)-1]
		}
	case 2:
		if len(p.setup.Dir) > 0 {
			p.setup.Dir = p.setup.Dir[:len(p.setup.Dir)-1]
		}
	}
}
