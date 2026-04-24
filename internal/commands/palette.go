package commands

import (
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/internal/theme"
)

type PaletteModel struct {
	registry *Registry
	themes   []theme.Theme
	query    string
	selected int
	executed *Command
	action   PaletteAction
	page     palettePage
	parents  []string
	original string
	sync     SyncSetup
	field    int
}

type palettePage int

const (
	paletteRoot palettePage = iota
	paletteThemes
	paletteSyncSetup
)

const (
	paletteModalWidth           = 86
	paletteTitleWidth           = 18
	paletteContentWidth         = 80
	paletteSelectedContentWidth = 78
)

type SyncSetup struct {
	Remote string
	Branch string
	Dir    string
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
	return PaletteModel{registry: registry, themes: themes}
}

func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd) {
	m.executed = nil
	m.action = PaletteAction{}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.updateKey(msg), nil
	}
	if m.selected >= len(m.matches()) {
		m.selected = 0
	}
	return m, nil
}

func (m *PaletteModel) Reset(currentTheme string) {
	m.query = ""
	m.selected = 0
	m.executed = nil
	m.action = PaletteAction{}
	m.page = paletteRoot
	m.parents = nil
	m.original = currentTheme
	m.field = 0
}

func (m PaletteModel) ExecutedCommand() *Command { return m.executed }

func (m PaletteModel) Action() PaletteAction { return m.action }

func (m *PaletteModel) ClearAction() { m.action = PaletteAction{} }
