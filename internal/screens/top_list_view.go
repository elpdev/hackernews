package screens

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (t Top) listView(width, height int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(t.listHeader()))
	b.WriteString("\n\n")
	if t.loading != "" {
		b.WriteString(t.loading + "\n")
	}
	if t.err != "" {
		b.WriteString(t.err + "\n")
	}
	if t.status != "" {
		b.WriteString(t.status + "\n")
	}
	if len(t.stories) == 0 {
		if t.loading == "" {
			b.WriteString("Press r to load " + strings.ToLower(t.feedTitle()) + ".\n")
		}
		return b.String()
	}
	sortLabel := t.sortMode.label()
	if t.searching || t.searchQuery != "" {
		label := "Filter"
		if t.searching {
			label = "Search"
		}
		query := t.searchQuery
		if query == "" {
			query = lipgloss.NewStyle().Faint(true).Render("type to filter...")
		}
		line := label + ": " + query
		if sortLabel != "" {
			line += " | sort: " + sortLabel
		}
		b.WriteString(truncateScreen(line, width) + "\n")
	} else if sortLabel != "" || t.hideRead {
		parts := make([]string, 0, 2)
		if sortLabel != "" {
			parts = append(parts, "sort: "+sortLabel)
		}
		if t.hideRead {
			parts = append(parts, "hiding read")
		}
		b.WriteString(truncateScreen(strings.Join(parts, " | "), width) + "\n")
	}

	matches := t.sortedFilteredStories()
	if len(matches) == 0 {
		b.WriteString(fmt.Sprintf("No stories match %q. Press ctrl+u to clear.\n", t.searchQuery))
		return b.String()
	}
	selectedInPage := clampIndex(t.selected, len(matches))
	listHeight := maxScreen(1, (height-3)/3)
	if t.searching || t.searchQuery != "" || sortLabel != "" {
		listHeight = maxScreen(1, (height-4)/3)
	}
	if selectedInPage < t.listTop {
		t.listTop = selectedInPage
	}
	if selectedInPage >= t.listTop+listHeight {
		t.listTop = selectedInPage - listHeight + 1
	}
	end := minScreen(len(matches), t.listTop+listHeight)
	metaStyle := lipgloss.NewStyle().Faint(true)
	readStyle := lipgloss.NewStyle().Faint(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	for i := t.listTop; i < end; i++ {
		item := matches[i]
		story := item.story
		line := fmt.Sprintf("%d. %s", item.index+1, story.Title)
		if domain := storyDomain(story.URL); domain != "" {
			line += " (" + domain + ")"
		}
		meta := fmt.Sprintf("   %d points by %s | %d comments", story.Score, story.By, story.Descendants)
		if t.readIDs[story.ID] {
			meta += " | read"
		}
		if t.savedIDs[story.ID] {
			meta += " | saved"
		}
		if i == selectedInPage {
			title := "> " + truncateScreen(line, maxScreen(0, width-2))
			b.WriteString(selectedStyle.Render(padLine(title, width)) + "\n")
			b.WriteString(selectedStyle.Render(padLine(truncateScreen(meta, width), width)) + "\n")
		} else {
			title := "  " + truncateScreen(line, maxScreen(0, width-2))
			if t.readIDs[story.ID] {
				title = readStyle.Render(title)
			}
			b.WriteString(title + "\n")
			b.WriteString(metaStyle.Render(truncateScreen(meta, width)) + "\n")
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	loadedCount := 0
	for _, page := range t.pages {
		loadedCount += len(page)
	}
	var footer string
	switch {
	case t.searchQuery != "" && sortLabel != "":
		footer = fmt.Sprintf("%d loaded | sort: %s | %d matches | O sort | h hide read | / edit search | ctrl+u clear | enter read | o open | s save", loadedCount, sortLabel, len(matches))
	case t.searchQuery != "":
		footer = fmt.Sprintf("%d loaded | %d matches | / edit search | ctrl+u clear | O sort | h hide read | enter read | o open | s save", loadedCount, len(matches))
	case sortLabel != "":
		footer = fmt.Sprintf("%d loaded | sort: %s | showing %d-%d of %d | O sort | h hide read | / search | enter read | o open | s save", loadedCount, sortLabel, t.listTop+1, end, len(matches))
	default:
		footer = fmt.Sprintf("Page %d/%d | showing %d-%d of %d | / search | O sort | h hide read | left/right page | j/k move | enter read | o open | s save | y copy | r refresh", t.page+1, t.pageCount(), matches[t.listTop].index+1, matches[end-1].index+1, len(t.storyIDs))
	}
	b.WriteString(truncateScreen(footer, width))
	return b.String()
}

// allLoadedStories returns every story cached in t.pages, ordered by HN rank.
// Each item's index is its global rank (position in t.storyIDs).
