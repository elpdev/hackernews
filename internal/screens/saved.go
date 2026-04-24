package screens

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/internal/browser"
	"github.com/elpdev/hackernews/internal/clipboard"
	"github.com/elpdev/hackernews/internal/media"
	"github.com/elpdev/hackernews/internal/saved"
)

type Saved struct {
	store    saved.Store
	opener   func(string) error
	copier   func(string) error
	items    []saved.Article
	selected int
	listTop  int
	readID   int
	readLine int
	loading  string
	err      string
	status   string
}

type savedArticlesLoadedMsg struct {
	items []saved.Article
	err   error
}

type savedArticleDeletedMsg struct {
	id  int
	err error
}

func NewSaved(store saved.Store) Saved {
	return Saved{store: store, opener: browser.Open, copier: clipboard.Copy, loading: "Loading saved articles..."}
}

func (s Saved) Init() tea.Cmd { return s.load() }

func (s Saved) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case savedArticlesLoadedMsg:
		s.loading = ""
		if msg.err != nil {
			s.err = msg.err.Error()
			return s, nil
		}
		s.err = ""
		s.items = msg.items
		s.selected = clampIndex(s.selected, len(s.items))
		if len(s.items) == 0 {
			s.readID = 0
		}
		return s, nil
	case savedArticleDeletedMsg:
		if msg.err != nil {
			s.status = "Could not delete saved article: " + msg.err.Error()
			return s, nil
		}
		s.status = "Article removed from saved"
		if s.readID == msg.id {
			s.readID = 0
			s.readLine = 0
		}
		return s, s.load()
	case tea.KeyPressMsg:
		return s.handleKey(msg)
	}
	return s, nil
}

func (s Saved) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if s.readID != 0 {
		return s.articleView(width, height)
	}
	return s.listView(width, height)
}

func (s Saved) Title() string { return "Saved" }

func (s Saved) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
		key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
		key.NewBinding(key.WithKeys("[", "]"), key.WithHelp("[/]", "paragraph")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "read")),
		key.NewBinding(key.WithKeys("s", "d"), key.WithHelp("s/d", "delete")),
		key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
		key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy url")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func (s Saved) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if s.readID != 0 {
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
		case "]":
			s.readLine = nextParagraphLine(cachedArticleLines(s.readID), s.readLine)
		case "[":
			s.readLine = previousParagraphLine(cachedArticleLines(s.readID), s.readLine)
		}
		s.readLine = clampIndex(s.readLine, cachedArticleLineCount(s.readID))
		return s, nil
	}

	switch msg.String() {
	case "r":
		s.loading = "Loading saved articles..."
		s.err = ""
		return s, s.load()
	}
	if len(s.items) == 0 {
		return s, nil
	}

	s.selected = clampIndex(s.selected, len(s.items))
	switch msg.String() {
	case "up", "k":
		if s.selected > 0 {
			s.selected--
		}
	case "down", "j":
		if s.selected < len(s.items)-1 {
			s.selected++
		}
	case "pgup":
		s.selected -= 10
		if s.selected < 0 {
			s.selected = 0
		}
	case "pgdown":
		s.selected += 10
		if s.selected >= len(s.items) {
			s.selected = len(s.items) - 1
		}
	case "enter":
		s.readID = s.items[s.selected].ID
		s.readLine = 0
	case "s", "d":
		return s, s.delete(s.items[s.selected].ID)
	case "y":
		s.status = s.copyArticleURL(savedArticleURL(s.items[s.selected]))
	}
	return s, nil
}

func (s Saved) load() tea.Cmd {
	if s.store == nil {
		return func() tea.Msg { return savedArticlesLoadedMsg{err: fmt.Errorf("saved article store is unavailable")} }
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		items, err := s.store.List(ctx)
		return savedArticlesLoadedMsg{items: items, err: err}
	}
}

func (s Saved) delete(id int) tea.Cmd {
	if s.store == nil {
		return func() tea.Msg {
			return savedArticleDeletedMsg{id: id, err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return savedArticleDeletedMsg{id: id, err: s.store.Delete(ctx, id)}
	}
}

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
	if len(s.items) == 0 {
		if s.loading == "" {
			b.WriteString("No saved articles yet. Press s on a top story to save it.\n")
		}
		return b.String()
	}

	listHeight := savedMax(1, (height-3)/3)
	if s.selected < s.listTop {
		s.listTop = s.selected
	}
	if s.selected >= s.listTop+listHeight {
		s.listTop = s.selected - listHeight + 1
	}
	end := minScreen(len(s.items), s.listTop+listHeight)
	metaStyle := lipgloss.NewStyle().Faint(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	for i := s.listTop; i < end; i++ {
		item := s.items[i]
		line := fmt.Sprintf("%2d. %s", i+1, savedTitle(item))
		if domain := storyDomain(item.Article.URL); domain != "" {
			line += " (" + domain + ")"
		}
		meta := fmt.Sprintf("     saved %s", item.SavedAt.Local().Format("2006-01-02 15:04"))
		if item.Story.By != "" {
			meta += " | by " + item.Story.By
		}
		if i == s.selected {
			title := "> " + truncateScreen(line, savedMax(0, width-2))
			b.WriteString(selectedStyle.Render(padLine(title, width)) + "\n")
			b.WriteString(selectedStyle.Render(padLine(truncateScreen(meta, width), width)) + "\n")
		} else {
			b.WriteString("  " + truncateScreen(line, savedMax(0, width-2)) + "\n")
			b.WriteString(metaStyle.Render(truncateScreen(meta, width)) + "\n")
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString(truncateScreen("j/k scroll | enter read | s/d delete | y copy url | r refresh", width))
	return b.String()
}

func (s Saved) articleView(width, height int) string {
	item, ok := s.itemByID(s.readID)
	if !ok {
		return "Saved article not found. Press esc to go back."
	}
	header := []string{"esc back | s/d delete | o open in browser | y copy url | j/k move | [/ ] paragraph"}
	if s.status != "" {
		header = append(header, s.status)
	}
	contentHeight := savedMax(1, height-len(header)-1)
	contentWidth := articleContentWidth(width)
	lines := renderedArticleLines(item.ID, contentWidth, item.Article, articleImage{})
	maxTop := savedMax(0, len(lines)-contentHeight)
	cursor := clampIndex(s.readLine, len(lines))
	top := cursor - contentHeight/2
	if top < 0 {
		top = 0
	} else if top > maxTop {
		top = maxTop
	}
	end := minScreen(len(lines), top+contentHeight)
	var b strings.Builder
	for _, line := range header {
		b.WriteString(truncateScreen(line, width) + "\n")
	}
	for i := top; i < end; i++ {
		line := lines[i]
		if i == cursor && !containsInlineImage(line) {
			line = articleLineHighlight(contentWidth).Render(padLine(ansi.Strip(line), contentWidth))
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return media.ViewportPrefix() + b.String()
}

func (s Saved) openArticleURL(url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to open"
	}
	if s.opener == nil {
		return "Could not open browser: no opener configured"
	}
	if err := s.opener(url); err != nil {
		return "Could not open browser: " + err.Error()
	}
	return "Opening in browser..."
}

func (s Saved) copyArticleURL(url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to copy"
	}
	if s.copier == nil {
		return "Could not copy URL: no clipboard configured"
	}
	if err := s.copier(url); err != nil {
		return "Could not copy URL: " + err.Error()
	}
	return "Copied URL to clipboard"
}

func (s Saved) itemByID(id int) (saved.Article, bool) {
	for _, item := range s.items {
		if item.ID == id {
			return item, true
		}
	}
	return saved.Article{}, false
}

func savedTitle(item saved.Article) string {
	if item.Article.Title != "" {
		return item.Article.Title
	}
	if item.Story.Title != "" {
		return item.Story.Title
	}
	return fmt.Sprintf("HN item %d", item.ID)
}

func savedArticleURL(item saved.Article) string {
	if strings.TrimSpace(item.Article.URL) != "" {
		return item.Article.URL
	}
	if strings.TrimSpace(item.Story.URL) != "" {
		return item.Story.URL
	}
	if item.ID == 0 {
		return ""
	}
	return fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID)
}

func savedMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
