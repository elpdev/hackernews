package modal

import (
	"github.com/elpdev/hackernews/pkg/theme"
	"charm.land/lipgloss/v2"
)

func Overlay(base, content string, width, height int, _ theme.Theme) string {
	contentW, contentH := lipgloss.Size(content)
	x := (width - contentW) / 2
	y := (height - contentH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	baseLayer := lipgloss.NewLayer(base)
	contentLayer := lipgloss.NewLayer(content).X(x).Y(y).Z(1)

	return lipgloss.NewCompositor(baseLayer, contentLayer).Render()
}
