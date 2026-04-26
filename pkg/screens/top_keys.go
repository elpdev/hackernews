package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (t Top) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if t.readID != 0 {
		if t.imageURL != "" {
			switch msg.String() {
			case "esc", "i":
				t.imageURL = ""
				return t, nil
			case "o":
				t.status = t.openArticleURL(t.imageURL)
				return t, nil
			case "y":
				t.status = t.copyArticleURL(t.imageURL)
				return t, nil
			}
			return t, nil
		}
		switch msg.String() {
		case "s":
			return t, t.toggleSaved(t.readID)
		case "o":
			t.status = t.openArticleURL(t.articles[t.readID].URL)
			return t, nil
		case "y":
			t.status = t.copyArticleURL(t.articleURLForID(t.readID))
			return t, nil
		case "i":
			lines := cachedArticleLines(t.readID)
			imageURL := articleBodyImageURLForCursor(t.articles[t.readID], lines, t.readLine)
			if imageURL == "" {
				t.status = "No article image reference nearby"
				return t, nil
			}
			t.imageURL = imageURL
			return t.startBodyImageLoad(t.readID, imageURL)
		case "c":
			if story, ok := t.storyByID(t.readID); ok {
				return t, func() tea.Msg {
					return OpenCommentsMsg{Story: story, ReturnTo: t.screenID()}
				}
			}
			return t, nil
		case "esc":
			t.readID = 0
			t.readTop = 0
			t.readLine = 0
			t.imageURL = ""
			return t, nil
		case "up", "k":
			if t.readLine > 0 {
				t.readLine--
			}
		case "down", "j":
			t.readLine++
		case "pgup":
			t.readLine -= 10
			if t.readLine < 0 {
				t.readLine = 0
			}
		case "pgdown":
			t.readLine += 10
		case "]", "right", "n":
			t.readLine = nextParagraphLine(cachedArticleLines(t.readID), t.readLine)
		case "[", "left", "p":
			t.readLine = previousParagraphLine(cachedArticleLines(t.readID), t.readLine)
		}
		t.readLine = clampIndex(t.readLine, cachedArticleLineCount(t.readID))
		return t, nil
	}

	if t.searching {
		return t.handleSearchKey(msg)
	}

	switch msg.String() {
	case "r":
		t.loading = "Loading " + strings.ToLower(t.feedTitle()) + "..."
		t.err = ""
		return t, tea.Batch(t.loadStories(), t.loadSavedIDs())
	case "/":
		t.searching = true
		return t, nil
	case "ctrl+u":
		t.searchQuery = ""
		t.selected = 0
		t.listTop = 0
		return t, nil
	case "O":
		t.sortMode = (t.sortMode + 1) % 3
		t.selected = 0
		t.listTop = 0
		return t, func() tea.Msg { return SortModeChangedMsg{Mode: t.sortMode.String()} }
	case "h":
		t.hideRead = !t.hideRead
		t.selected = 0
		t.listTop = 0
		return t, func() tea.Msg { return HideReadToggledMsg{HideRead: t.hideRead} }
	}
	matches := t.sortedFilteredStories()
	if len(matches) == 0 {
		return t, nil
	}
	t.selected = clampIndex(t.selected, len(matches))

	switch msg.String() {
	case "left", "p":
		if t.page > 0 {
			return t.goToPage(t.page - 1)
		}
	case "right", "n":
		if t.page < t.pageCount()-1 {
			return t.goToPage(t.page + 1)
		}
	case "up", "k":
		if t.selected > 0 {
			t.selected--
		}
	case "down", "j":
		if t.selected < len(matches)-1 {
			t.selected++
		}
	case "pgup":
		t.selected -= 10
		if t.selected < 0 {
			t.selected = 0
		}
	case "pgdown":
		t.selected += 10
		if t.selected >= len(matches) {
			t.selected = len(matches) - 1
		}
	case "enter":
		if t.loading != "" {
			return t, nil
		}
		story := matches[t.selected].story
		markRead := t.markRead(story.ID)
		if _, ok := t.articles[story.ID]; ok {
			t.readID = story.ID
			t.readTop = 0
			t.readLine = 0
			t.imageURL = ""
			return t, markRead
		}
		t.loading = "Fetching article..."
		t.err = ""
		return t, tea.Batch(markRead, t.loadArticle(story))
	case "c":
		story := matches[t.selected].story
		return t, func() tea.Msg {
			return OpenCommentsMsg{Story: story, ReturnTo: t.screenID()}
		}
	case "s":
		story := matches[t.selected].story
		if !t.savedIDs[story.ID] && !t.hasExtractedArticle(story) {
			t.loading = "Fetching article to save..."
			t.err = ""
		}
		return t, t.toggleSaved(matches[t.selected].story.ID)
	case "y":
		t.status = t.copyArticleURL(t.articleURLForStory(matches[t.selected].story))
	case "o":
		story := matches[t.selected].story
		t.status = t.openArticleURL(t.articleURLForStory(story))
		return t, t.markRead(story.ID)
	}
	return t, nil
}

func (t Top) handleSearchKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		t.searching = false
		return t, nil
	case "ctrl+u":
		t.searchQuery = ""
		t.selected = 0
		t.listTop = 0
		return t, nil
	case "backspace", "ctrl+h":
		if len(t.searchQuery) > 0 {
			t.searchQuery = t.searchQuery[:len(t.searchQuery)-1]
			t.selected = 0
			t.listTop = 0
		}
		return t, nil
	case "space":
		t.searchQuery += " "
		t.selected = 0
		t.listTop = 0
		return t, nil
	}
	if len(msg.String()) == 1 {
		t.searchQuery += msg.String()
		t.selected = 0
		t.listTop = 0
	}
	return t, nil
}
