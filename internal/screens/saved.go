package screens

import (
	"context"
	"fmt"
	"sort"
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

	searching   bool
	searchQuery string
	sortMode    savedSortMode
	tagEditing  bool
	tagID       int
	tagInput    string
}

type savedSortMode int

const (
	savedSortSavedAt savedSortMode = iota
	savedSortStoryDate
	savedSortTitle
)

func (m savedSortMode) label() string {
	switch m {
	case savedSortStoryDate:
		return "story date"
	case savedSortTitle:
		return "title"
	default:
		return "saved date"
	}
}

type savedListItem struct {
	index int
	item  saved.Article
}

type savedArticlesLoadedMsg struct {
	screenID string
	items    []saved.Article
	err      error
}

func (m savedArticlesLoadedMsg) TargetScreenID() string { return m.screenID }

type savedArticleDeletedMsg struct {
	screenID string
	id       int
	err      error
}

func (m savedArticleDeletedMsg) TargetScreenID() string { return m.screenID }

type savedArticleTagsUpdatedMsg struct {
	screenID string
	id       int
	tags     []string
	err      error
}

func (m savedArticleTagsUpdatedMsg) TargetScreenID() string { return m.screenID }

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
	case savedArticleTagsUpdatedMsg:
		if msg.err != nil {
			s.status = "Could not update tags: " + msg.err.Error()
			return s, nil
		}
		for i := range s.items {
			if s.items[i].ID == msg.id {
				s.items[i].Tags = msg.tags
				break
			}
		}
		s.status = "Tags updated"
		return s, nil
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
		key.NewBinding(key.WithKeys("left", "p"), key.WithHelp("left/p", "prev paragraph")),
		key.NewBinding(key.WithKeys("right", "n"), key.WithHelp("right/n", "next paragraph")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "clear search")),
		key.NewBinding(key.WithKeys("O"), key.WithHelp("O", "cycle sort")),
		key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "edit tags")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "read")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "comments")),
		key.NewBinding(key.WithKeys("s", "d"), key.WithHelp("s/d", "delete")),
		key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
		key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy url")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func (s Saved) CapturesKey(msg tea.KeyPressMsg) bool {
	return (s.searching || s.tagEditing) && s.readID == 0 && msg.String() != "ctrl+c"
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
		case "c":
			if item, ok := s.itemByID(s.readID); ok {
				return s, func() tea.Msg {
					return OpenCommentsMsg{Story: item.Story, ReturnTo: "saved"}
				}
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
		s.searchQuery = ""
		s.selected = 0
		s.listTop = 0
		return s, nil
	case "O":
		s.sortMode = (s.sortMode + 1) % 3
		s.selected = 0
		s.listTop = 0
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
		s.readID = matches[s.selected].item.ID
		s.readLine = 0
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
		item := matches[s.selected].item
		return s, func() tea.Msg {
			return OpenCommentsMsg{Story: item.Story, ReturnTo: "saved"}
		}
	}
	return s, nil
}

func (s Saved) handleTagKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.tagEditing = false
		s.tagID = 0
		s.tagInput = ""
		return s, nil
	case "enter":
		id := s.tagID
		tags := parseSavedTags(s.tagInput)
		s.tagEditing = false
		s.tagID = 0
		s.tagInput = ""
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
		s.searchQuery = ""
		s.selected = 0
		s.listTop = 0
		return s, nil
	case "backspace", "ctrl+h":
		if len(s.searchQuery) > 0 {
			s.searchQuery = s.searchQuery[:len(s.searchQuery)-1]
			s.selected = 0
			s.listTop = 0
		}
		return s, nil
	case "space":
		s.searchQuery += " "
		s.selected = 0
		s.listTop = 0
		return s, nil
	}
	if len(msg.String()) == 1 {
		s.searchQuery += msg.String()
		s.selected = 0
		s.listTop = 0
	}
	return s, nil
}

func (s Saved) load() tea.Cmd {
	if s.store == nil {
		return func() tea.Msg {
			return savedArticlesLoadedMsg{screenID: "saved", err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		items, err := s.store.List(ctx)
		return savedArticlesLoadedMsg{screenID: "saved", items: items, err: err}
	}
}

func (s Saved) delete(id int) tea.Cmd {
	if s.store == nil {
		return func() tea.Msg {
			return savedArticleDeletedMsg{screenID: "saved", id: id, err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return savedArticleDeletedMsg{screenID: "saved", id: id, err: s.store.Delete(ctx, id)}
	}
}

func (s Saved) setTags(id int, tags []string) tea.Cmd {
	if s.store == nil {
		return func() tea.Msg {
			return savedArticleTagsUpdatedMsg{screenID: "saved", id: id, tags: tags, err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return savedArticleTagsUpdatedMsg{screenID: "saved", id: id, tags: tags, err: s.store.SetTags(ctx, id, tags)}
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
	matches := s.filteredItems()
	if len(s.items) == 0 {
		if s.loading == "" {
			b.WriteString("No saved articles yet. Press s on a top story to save it.\n")
		}
		return b.String()
	}
	if s.searching || s.searchQuery != "" || s.sortMode != savedSortSavedAt {
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
	if s.tagEditing {
		input := s.tagInput
		if input == "" {
			input = lipgloss.NewStyle().Faint(true).Render("comma-separated tags")
		}
		b.WriteString(truncateScreen("Tags: "+input+"  (enter save, esc cancel)", width) + "\n")
	}
	if len(matches) == 0 {
		b.WriteString(fmt.Sprintf("No saved articles match %q. Press ctrl+u to clear.\n", s.searchQuery))
		return b.String()
	}

	listHeight := savedMax(1, (height-3)/3)
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
		entry := matches[i]
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
	b.WriteString(truncateScreen("j/k scroll | / search | O sort | t tags | enter read | o open | s/d delete | y copy url | r refresh", width))
	return b.String()
}

func (s Saved) articleView(width, height int) string {
	item, ok := s.itemByID(s.readID)
	if !ok {
		return "Saved article not found. Press esc to go back."
	}
	header := []string{"esc back | s/d delete | o open | y copy | j/k line | left/right or p/n paragraph"}
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

func (s Saved) filteredItems() []savedListItem {
	items := make([]savedListItem, 0, len(s.items))
	query := strings.ToLower(strings.TrimSpace(s.searchQuery))
	for i, item := range s.items {
		if query == "" || savedMatchesQuery(item, query) {
			items = append(items, savedListItem{index: i, item: item})
		}
	}
	switch s.sortMode {
	case savedSortStoryDate:
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].item.Story.Time > items[j].item.Story.Time
		})
	case savedSortTitle:
		sort.SliceStable(items, func(i, j int) bool {
			return strings.ToLower(savedTitle(items[i].item)) < strings.ToLower(savedTitle(items[j].item))
		})
	}
	return items
}

func savedMatchesQuery(item saved.Article, query string) bool {
	fields := []string{savedTitle(item), item.Story.By, item.Story.URL, item.Article.URL, storyDomain(savedArticleURL(item)), item.Article.Markdown, strings.Join(item.Tags, " ")}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func parseSavedTags(input string) []string {
	parts := strings.Split(input, ",")
	seen := make(map[string]bool, len(parts))
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.ToLower(strings.TrimSpace(part))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
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
