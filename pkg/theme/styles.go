package theme

import (
	"charm.land/lipgloss/v2"
	"github.com/elpdev/tuitheme"
)

func BuiltIns() []Theme {
	return []Theme{Phosphor(), MutedDark(), Synthwave()}
}

func Next(current string) Theme {
	return tuitheme.Next(BuiltIns(), current)
}

func MutedDark() Theme {
	primary := lipgloss.Color("#A78BFA")
	muted := lipgloss.Color("#9CA3AF")
	border := lipgloss.Color("#374151")
	bg := lipgloss.Color("#111827")
	return Theme{
		Name:       "Muted Dark",
		Background: bg,
		Text:       lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Background(bg),
		Muted:      lipgloss.NewStyle().Foreground(muted).Background(bg),
		Title:      lipgloss.NewStyle().Bold(true).Foreground(primary).Background(bg),
		Selected:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#111827")).Background(primary).Padding(0, 1),
		Header:     lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Background(bg).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(border).Padding(0, 1),
		Sidebar:    lipgloss.NewStyle().Background(bg).Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(border).Padding(1, 1),
		Main:       lipgloss.NewStyle().Background(bg).Padding(1, 2),
		Footer:     lipgloss.NewStyle().Foreground(muted).Background(bg).Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(border).Padding(0, 1),
		Modal:      lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Background(lipgloss.Color("#1F2937")).Border(lipgloss.RoundedBorder()).BorderForeground(primary).Padding(1, 2),
		Border:     lipgloss.NewStyle().Foreground(border).Background(bg),
		Info:       lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA")).Background(bg),
		Warn:       lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Background(bg),
	}
}

func Phosphor() Theme {
	return tuitheme.Phosphor()
}

func Synthwave() Theme {
	bright := lipgloss.Color("#F5E6F7")
	muted := lipgloss.Color("#8E7AB5")
	subtle := lipgloss.Color("#6B5B8E")
	bg := lipgloss.Color("#1A0933")
	surface := lipgloss.Color("#2B1055")
	selected := lipgloss.Color("#3D1E6D")
	divider := lipgloss.Color("#4A2B7A")
	pink := lipgloss.Color("#FF3CAC")
	violet := lipgloss.Color("#9B5DE5")
	cyan := lipgloss.Color("#51E2F5")
	gold := lipgloss.Color("#FFB86C")

	return Theme{
		Name:       "Synthwave",
		Background: bg,
		Text:       lipgloss.NewStyle().Foreground(bright).Background(bg),
		Muted:      lipgloss.NewStyle().Foreground(muted).Background(bg),
		Title:      lipgloss.NewStyle().Bold(true).Foreground(pink).Background(bg),
		Selected:   lipgloss.NewStyle().Bold(true).Foreground(bright).Background(selected).Padding(0, 1),
		Header:     lipgloss.NewStyle().Foreground(bright).Background(bg).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(pink).Padding(0, 1),
		Sidebar:    lipgloss.NewStyle().Background(bg).Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(violet).Padding(1, 1),
		Main:       lipgloss.NewStyle().Foreground(bright).Background(bg).Padding(1, 2),
		Footer:     lipgloss.NewStyle().Foreground(subtle).Background(bg).Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(divider).Padding(0, 1),
		Modal:      lipgloss.NewStyle().Foreground(bright).Background(surface).Border(lipgloss.RoundedBorder()).BorderForeground(pink).Padding(1, 2),
		Border:     lipgloss.NewStyle().Foreground(divider).Background(bg),
		Info:       lipgloss.NewStyle().Foreground(cyan).Background(bg),
		Warn:       lipgloss.NewStyle().Foreground(gold).Background(bg),
	}
}
