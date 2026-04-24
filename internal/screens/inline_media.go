package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/internal/media"
)

func InlineMediaCmd(view string) tea.Cmd {
	seq := InlineMediaSequence(view)
	if seq == "" {
		return nil
	}
	return tea.Raw(seq)
}

func InlineMediaSequence(view string) string {
	lines := strings.Split(view, "\n")
	var b strings.Builder
	b.WriteString(media.ViewportPrefix())
	b.WriteString(ansi.SaveCursorPosition)
	drew := false
	for row, line := range lines {
		if !containsInlineImage(line) {
			continue
		}
		b.WriteString(ansi.CursorPosition(1, row+1))
		b.WriteString(line)
		drew = true
	}
	if !drew {
		return ""
	}
	b.WriteString(ansi.RestoreCursorPosition)
	return b.String()
}
