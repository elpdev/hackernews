package screens

import (
	"strings"

	"charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/internal/saved"
)

func (s Saved) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if s.readID != 0 {
		return s.handleReaderKey(msg)
	}

	if s.tagEditing {
		return s.handleTagKey(msg)
	}

	if s.searching {
		return s.handleSearchKey(msg)
	}

	switch msg.String() {
	case "r":
		s.loading = "Loading saved articles..."
		s.err = ""
		return s, s.load()
	case "/":
		s.searching = true
		return s, nil
	case "ctrl+u":
		s.clearSearch()
		return s, nil
	case "O":
		s.sortMode = (s.sortMode + 1) % 3
		s.resetListPosition()
		return s, nil
	}

	matches := s.filteredItems()
	if len(matches) == 0 {
		return s, nil
	}

	s.selected = clampIndex(s.selected, len(matches))
	switch msg.String() {
	case "up", "k":
		if s.selected > 0 {
			s.selected--
		}
	case "down", "j":
		if s.selected < len(matches)-1 {
			s.selected++
		}
	case "pgup":
		s.selected -= 10
		if s.selected < 0 {
			s.selected = 0
		}
	case "pgdown":
		s.selected += 10
		if s.selected >= len(matches) {
			s.selected = len(matches) - 1
		}
	case "enter":
		return s.openSavedArticle(matches[s.selected].item)
	case "s", "d":
		return s, s.delete(matches[s.selected].item.ID)
	case "t":
		item := matches[s.selected].item
		s.tagEditing = true
		s.tagID = item.ID
		s.tagInput = strings.Join(item.Tags, ", ")
		return s, nil
	case "y":
		s.status = s.copyArticleURL(savedArticleURL(matches[s.selected].item))
	case "o":
		s.status = s.openArticleURL(savedArticleURL(matches[s.selected].item))
	case "c":
		return s, s.openComments(matches[s.selected].item)
	}
	return s, nil
}

func (s Saved) handleReaderKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.readID = 0
		s.readLine = 0
		return s, nil
	case "s", "d":
		return s, s.delete(s.readID)
	case "o":
		if item, ok := s.itemByID(s.readID); ok {
			s.status = s.openArticleURL(savedArticleURL(item))
		} else {
			s.status = "No URL to open"
		}
		return s, nil
	case "y":
		if item, ok := s.itemByID(s.readID); ok {
			s.status = s.copyArticleURL(savedArticleURL(item))
		} else {
			s.status = "No URL to copy"
		}
		return s, nil
	case "c":
		if item, ok := s.itemByID(s.readID); ok {
			return s, s.openComments(item)
		}
		return s, nil
	case "up", "k":
		if s.readLine > 0 {
			s.readLine--
		}
	case "down", "j":
		s.readLine++
	case "pgup":
		s.readLine -= 10
		if s.readLine < 0 {
			s.readLine = 0
		}
	case "pgdown":
		s.readLine += 10
	case "]", "right", "n":
		s.readLine = nextParagraphLine(cachedArticleLines(s.readID), s.readLine)
	case "[", "left", "p":
		s.readLine = previousParagraphLine(cachedArticleLines(s.readID), s.readLine)
	}
	s.readLine = clampIndex(s.readLine, cachedArticleLineCount(s.readID))
	return s, nil
}

func (s Saved) handleTagKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.cancelTagEdit()
		return s, nil
	case "enter":
		id := s.tagID
		tags := parseSavedTags(s.tagInput)
		s.cancelTagEdit()
		return s, s.setTags(id, tags)
	case "backspace", "ctrl+h":
		if len(s.tagInput) > 0 {
			s.tagInput = s.tagInput[:len(s.tagInput)-1]
		}
		return s, nil
	case "space":
		s.tagInput += " "
		return s, nil
	}
	if len(msg.String()) == 1 {
		s.tagInput += msg.String()
	}
	return s, nil
}

func (s Saved) handleSearchKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.searching = false
		return s, nil
	case "ctrl+u":
		s.clearSearch()
		return s, nil
	case "backspace", "ctrl+h":
		if len(s.searchQuery) > 0 {
			s.searchQuery = s.searchQuery[:len(s.searchQuery)-1]
			s.resetListPosition()
		}
		return s, nil
	case "space":
		s.searchQuery += " "
		s.resetListPosition()
		return s, nil
	}
	if len(msg.String()) == 1 {
		s.searchQuery += msg.String()
		s.resetListPosition()
	}
	return s, nil
}

func (s Saved) openSavedArticle(item saved.Article) (Screen, tea.Cmd) {
	if strings.TrimSpace(item.Article.Markdown) == "" {
		s.loading = "Fetching saved article..."
		s.status = ""
		return s, s.extractSavedArticle(item)
	}
	s.readID = item.ID
	s.readLine = 0
	return s, nil
}

func (s Saved) openComments(item saved.Article) tea.Cmd {
	return func() tea.Msg {
		return OpenCommentsMsg{Story: item.Story, ReturnTo: "saved"}
	}
}

func (s *Saved) clearSearch() {
	s.searchQuery = ""
	s.resetListPosition()
}

func (s *Saved) resetListPosition() {
	s.selected = 0
	s.listTop = 0
}

func (s *Saved) cancelTagEdit() {
	s.tagEditing = false
	s.tagID = 0
	s.tagInput = ""
}
