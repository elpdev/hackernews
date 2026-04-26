package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/hackernews/internal/browser"
	"github.com/elpdev/hackernews/internal/clipboard"
	"github.com/elpdev/hackernews/pkg/hn"
)

type Search struct {
	items    []StorySnapshot
	query    string
	selected int
	listTop  int
	status   string
	opener   func(string) error
	copier   func(string) error
}

func NewSearch() Search {
	return Search{opener: browser.Open, copier: clipboard.Copy}
}

func (s Search) WithItems(items []StorySnapshot) Search {
	s.items = append([]StorySnapshot(nil), items...)
	s.selected = clampIndex(s.selected, len(s.matches()))
	return s
}

func (s Search) Init() tea.Cmd { return nil }

func (s Search) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return s.handleKey(msg)
	}
	return s, nil
}

func (s Search) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	matches := s.matches()
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Search Loaded Stories"))
	b.WriteString("\n")
	query := s.query
	if query == "" {
		query = lipgloss.NewStyle().Faint(true).Render("type to search loaded feeds...")
	}
	b.WriteString(truncateScreen("> "+query, width) + "\n")
	if s.status != "" {
		b.WriteString(truncateScreen(s.status, width) + "\n")
	}
	if len(s.items) == 0 {
		b.WriteString("No loaded stories yet. Open a feed first.\n")
		return b.String()
	}
	if len(matches) == 0 {
		b.WriteString(fmt.Sprintf("No loaded stories match %q.\n", s.query))
		return b.String()
	}
	listHeight := maxScreen(1, (height-4)/3)
	if s.selected < s.listTop {
		s.listTop = s.selected
	}
	if s.selected >= s.listTop+listHeight {
		s.listTop = s.selected - listHeight + 1
	}
	end := minScreen(len(matches), s.listTop+listHeight)
	metaStyle := lipgloss.NewStyle().Faint(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	for i := s.listTop; i < end; i++ {
		item := matches[i]
		line := fmt.Sprintf("%s #%d · %s", item.FeedName, item.Rank+1, item.Story.Title)
		if domain := storyDomain(item.Story.URL); domain != "" {
			line += " (" + domain + ")"
		}
		meta := fmt.Sprintf("   %d points by %s | %d comments", item.Story.Score, item.Story.By, item.Story.Descendants)
		if i == s.selected {
			b.WriteString(selectedStyle.Render(padLine("> "+truncateScreen(line, maxScreen(0, width-2)), width)) + "\n")
			b.WriteString(selectedStyle.Render(padLine(truncateScreen(meta, width), width)) + "\n")
		} else {
			b.WriteString("  " + truncateScreen(line, maxScreen(0, width-2)) + "\n")
			b.WriteString(metaStyle.Render(truncateScreen(meta, width)) + "\n")
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString(truncateScreen("j/k move | o open | c comments | y copy url | ctrl+u clear", width))
	return b.String()
}

func (s Search) Title() string { return "Search" }

func (s Search) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
		key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "comments")),
		key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy url")),
		key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "clear search")),
	}
}

func (s Search) CapturesKey(msg tea.KeyPressMsg) bool { return msg.String() != "ctrl+c" }

func (s Search) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	matches := s.matches()
	switch msg.String() {
	case "esc":
		return s, func() tea.Msg { return NavigateMsg{ScreenID: "top"} }
	case "up", "k":
		if s.selected > 0 {
			s.selected--
		}
		return s, nil
	case "down", "j":
		if s.selected < len(matches)-1 {
			s.selected++
		}
		return s, nil
	case "ctrl+u":
		s.query = ""
		s.selected = 0
		s.listTop = 0
		return s, nil
	case "backspace", "ctrl+h":
		if len(s.query) > 0 {
			s.query = s.query[:len(s.query)-1]
			s.selected = 0
			s.listTop = 0
		}
		return s, nil
	case "space":
		s.query += " "
		s.selected = 0
		s.listTop = 0
		return s, nil
	}
	if len(matches) > 0 {
		s.selected = clampIndex(s.selected, len(matches))
		story := matches[s.selected].Story
		switch msg.String() {
		case "o":
			s.status = commentsOpenURL(s.opener, searchStoryURL(story))
			return s, nil
		case "y":
			s.status = commentsCopyURL(s.copier, searchStoryURL(story))
			return s, nil
		case "c", "enter":
			return s, func() tea.Msg { return OpenCommentsMsg{Story: story, ReturnTo: "search"} }
		}
	}
	if len(msg.String()) == 1 {
		s.query += msg.String()
		s.selected = 0
		s.listTop = 0
	}
	return s, nil
}

func (s Search) matches() []StorySnapshot {
	query := strings.ToLower(strings.TrimSpace(s.query))
	if query == "" {
		return append([]StorySnapshot(nil), s.items...)
	}
	out := make([]StorySnapshot, 0, len(s.items))
	for _, item := range s.items {
		if storyMatchesQuery(item.Story, query) || strings.Contains(strings.ToLower(item.FeedName), query) {
			out = append(out, item)
		}
	}
	return out
}

func searchStoryURL(story hn.Item) string {
	if strings.TrimSpace(story.URL) != "" {
		return story.URL
	}
	if story.ID == 0 {
		return ""
	}
	return fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)
}
