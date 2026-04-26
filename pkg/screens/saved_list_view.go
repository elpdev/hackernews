package screens

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (s Saved) listView(width, height int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Saved Articles"))
	b.WriteString("\n")
	if s.loading != "" {
		b.WriteString(s.loading + "\n")
	}
	if s.err != "" {
		b.WriteString(s.err + "\n")
	}
	if s.status != "" {
		b.WriteString(s.status + "\n")
	}
	matches := s.filteredItems()
	if len(s.items) == 0 {
		if s.loading == "" {
			b.WriteString("No saved articles yet. Press s on a top story to save it.\n")
		}
		return b.String()
	}
	s.writeFilterStatus(&b, width)
	s.writeTagEditor(&b, width)
	if len(matches) == 0 {
		b.WriteString(fmt.Sprintf("No saved articles match %q. Press ctrl+u to clear.\n", s.searchQuery))
		return b.String()
	}

	listHeight := maxScreen(1, (height-3)/3)
	if s.selected < s.listTop {
		s.listTop = s.selected
	}
	if s.selected >= s.listTop+listHeight {
		s.listTop = s.selected - listHeight + 1
	}
	end := minScreen(len(matches), s.listTop+listHeight)
	s.writeSavedRows(&b, matches, width, end)
	b.WriteString(truncateScreen("j/k scroll | / search | O sort | t tags | enter read | o open | s/d delete | y copy url | r refresh", width))
	return b.String()
}

func (s Saved) writeFilterStatus(b *strings.Builder, width int) {
	if !s.searching && s.searchQuery == "" && s.sortMode == savedSortSavedAt {
		return
	}
	label := "Filter"
	if s.searching {
		label = "Search"
	}
	query := s.searchQuery
	if query == "" {
		query = lipgloss.NewStyle().Faint(true).Render("type to filter saved articles...")
	}
	b.WriteString(truncateScreen(fmt.Sprintf("%s: %s | sort: %s", label, query, s.sortMode.label()), width) + "\n")
}

func (s Saved) writeTagEditor(b *strings.Builder, width int) {
	if !s.tagEditing {
		return
	}
	input := s.tagInput
	if input == "" {
		input = lipgloss.NewStyle().Faint(true).Render("comma-separated tags")
	}
	b.WriteString(truncateScreen("Tags: "+input+"  (enter save, esc cancel)", width) + "\n")
}

func (s Saved) writeSavedRows(b *strings.Builder, matches []savedListItem, width, end int) {
	metaStyle := lipgloss.NewStyle().Faint(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	for i := s.listTop; i < end; i++ {
		entry := matches[i]
		line, meta := savedRowText(entry)
		if i == s.selected {
			title := "> " + truncateScreen(line, maxScreen(0, width-2))
			b.WriteString(selectedStyle.Render(padLine(title, width)) + "\n")
			b.WriteString(selectedStyle.Render(padLine(truncateScreen(meta, width), width)) + "\n")
		} else {
			b.WriteString("  " + truncateScreen(line, maxScreen(0, width-2)) + "\n")
			b.WriteString(metaStyle.Render(truncateScreen(meta, width)) + "\n")
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}
}

func savedRowText(entry savedListItem) (string, string) {
	item := entry.item
	line := fmt.Sprintf("%2d. %s", entry.index+1, savedTitle(item))
	if domain := storyDomain(item.Article.URL); domain != "" {
		line += " (" + domain + ")"
	}
	meta := fmt.Sprintf("     saved %s", item.SavedAt.Local().Format("2006-01-02 15:04"))
	if item.Story.By != "" {
		meta += " | by " + item.Story.By
	}
	if len(item.Tags) > 0 {
		meta += " | tags: " + strings.Join(item.Tags, ", ")
	}
	return line, meta
}
