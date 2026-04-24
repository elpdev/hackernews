package screens

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/hackernews/internal/articles"
	"github.com/elpdev/hackernews/internal/hn"
)

const (
	topStoryLimit     = 500
	topStoriesPerPage = 100
)

type Top struct {
	client    hn.Client
	extractor articles.Extractor
	storyIDs  []int
	stories   []hn.Item
	pages     map[int][]hn.Item
	articles  map[int]articles.Article

	selected int
	page     int
	listTop  int
	readID   int
	readTop  int
	loading  string
	err      string
}

type topStoriesLoadedMsg struct {
	ids     []int
	stories []hn.Item
	err     error
}

type storyPageLoadedMsg struct {
	page    int
	stories []hn.Item
	err     error
}

type articleLoadedMsg struct {
	id      int
	article articles.Article
	err     error
}

func NewTop() Top {
	return Top{
		client:    hn.NewClient(nil),
		extractor: articles.NewTrafilaturaExtractor(),
		pages:     make(map[int][]hn.Item),
		articles:  make(map[int]articles.Article),
		loading:   "Loading top stories...",
	}
}

func (t Top) Init() tea.Cmd { return t.loadStories() }

func (t Top) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case topStoriesLoadedMsg:
		t.loading = ""
		if msg.err != nil {
			t.err = msg.err.Error()
			return t, nil
		}
		t.err = ""
		t.storyIDs = msg.ids
		t.pages = map[int][]hn.Item{0: msg.stories}
		t.stories = msg.stories
		t.page = 0
		t.selected = 0
		t.listTop = 0
		return t, nil
	case storyPageLoadedMsg:
		t.loading = ""
		if msg.err != nil {
			t.err = msg.err.Error()
			return t, nil
		}
		t.err = ""
		t.page = msg.page
		t.pages[msg.page] = msg.stories
		t.stories = msg.stories
		t.selected = 0
		t.listTop = 0
		if !t.selectedInPage() {
			t.selected = 0
			t.listTop = 0
		}
		return t, nil
	case articleLoadedMsg:
		t.loading = ""
		if msg.err != nil {
			t.err = msg.err.Error()
			return t, nil
		}
		t.err = ""
		t.articles[msg.id] = msg.article
		t.readID = msg.id
		t.readTop = 0
		return t, nil
	case tea.KeyPressMsg:
		return t.handleKey(msg)
	}
	return t, nil
}

func (t Top) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if t.readID != 0 {
		return t.articleView(width, height)
	}
	return t.listView(width, height)
}

func (t Top) Title() string { return "Top Stories" }

func (t Top) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
		key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
		key.NewBinding(key.WithKeys("left", "p"), key.WithHelp("left/p", "prev 100")),
		key.NewBinding(key.WithKeys("right", "n"), key.WithHelp("right/n", "next 100")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "read")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func (t Top) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if t.readID != 0 {
		switch msg.String() {
		case "esc":
			t.readID = 0
			t.readTop = 0
		case "up", "k":
			if t.readTop > 0 {
				t.readTop--
			}
		case "down", "j":
			t.readTop++
		case "pgup":
			t.readTop -= 10
			if t.readTop < 0 {
				t.readTop = 0
			}
		case "pgdown":
			t.readTop += 10
		}
		return t, nil
	}

	switch msg.String() {
	case "r":
		t.loading = "Loading top stories..."
		t.err = ""
		return t, t.loadStories()
	}
	if len(t.stories) == 0 {
		return t, nil
	}

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
		if t.selected < len(t.stories)-1 {
			t.selected++
		}
	case "pgup":
		t.selected -= 10
		if t.selected < 0 {
			t.selected = 0
		}
	case "pgdown":
		t.selected += 10
		if t.selected >= len(t.stories) {
			t.selected = len(t.stories) - 1
		}
	case "enter":
		if t.loading != "" {
			return t, nil
		}
		story := t.stories[t.selected]
		if _, ok := t.articles[story.ID]; ok {
			t.readID = story.ID
			t.readTop = 0
			return t, nil
		}
		t.loading = "Fetching article..."
		t.err = ""
		return t, t.loadArticle(story)
	}
	return t, nil
}

func (t Top) loadStories() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		ids, err := t.client.TopStoryIDs(ctx)
		if err != nil {
			return topStoriesLoadedMsg{err: err}
		}
		if len(ids) > topStoryLimit {
			ids = ids[:topStoryLimit]
		}
		end := minScreen(len(ids), topStoriesPerPage)
		stories, err := t.client.Stories(ctx, ids[:end])
		return topStoriesLoadedMsg{ids: ids, stories: stories, err: err}
	}
}

func (t Top) goToPage(page int) (Screen, tea.Cmd) {
	if stories, ok := t.pages[page]; ok {
		t.page = page
		t.stories = stories
		t.selected = 0
		t.listTop = 0
		return t, nil
	}
	t.loading = fmt.Sprintf("Loading page %d...", page+1)
	t.err = ""
	return t, t.loadStoryPage(page)
}

func (t Top) loadStoryPage(page int) tea.Cmd {
	ids := append([]int(nil), t.storyIDs...)
	return func() tea.Msg {
		start := page * topStoriesPerPage
		if start >= len(ids) {
			return storyPageLoadedMsg{page: page, stories: nil}
		}
		end := minScreen(len(ids), start+topStoriesPerPage)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		stories, err := t.client.Stories(ctx, ids[start:end])
		return storyPageLoadedMsg{page: page, stories: stories, err: err}
	}
}

func (t Top) loadArticle(story hn.Item) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(story.URL) == "" {
			return articleLoadedMsg{id: story.ID, article: articles.Article{
				Title:    story.Title,
				Author:   story.By,
				URL:      fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID),
				Markdown: hnTextMarkdown(story),
			}}
		}
		article, err := t.extractor.Extract(story.URL)
		if article.Title == "" {
			article.Title = story.Title
		}
		if article.Author == "" {
			article.Author = story.By
		}
		return articleLoadedMsg{id: story.ID, article: article, err: err}
	}
}

func (t Top) listView(width, height int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Top Hacker News"))
	b.WriteString("\n")
	if t.loading != "" {
		b.WriteString(t.loading + "\n")
	}
	if t.err != "" {
		b.WriteString(t.err + "\n")
	}
	if len(t.stories) == 0 {
		if t.loading == "" {
			b.WriteString("Press r to load top stories.\n")
		}
		return b.String()
	}

	selectedInPage := t.selected
	listHeight := maxScreen(1, (height-4)/2)
	if selectedInPage < t.listTop {
		t.listTop = selectedInPage
	}
	if selectedInPage >= t.listTop+listHeight {
		t.listTop = selectedInPage - listHeight + 1
	}
	end := minScreen(len(t.stories), t.listTop+listHeight)
	for i := t.listTop; i < end; i++ {
		story := t.stories[i]
		line := fmt.Sprintf("%2d. %s", t.page*topStoriesPerPage+i+1, story.Title)
		if domain := storyDomain(story.URL); domain != "" {
			line += " (" + domain + ")"
		}
		meta := fmt.Sprintf("    %d points by %s | %d comments", story.Score, story.By, story.Descendants)
		if i == t.selected {
			b.WriteString(lipgloss.NewStyle().Bold(true).Render("> "+truncateScreen(line, maxScreen(0, width-2))) + "\n")
		} else {
			b.WriteString("  " + truncateScreen(line, maxScreen(0, width-2)) + "\n")
		}
		b.WriteString(truncateScreen(meta, width) + "\n")
	}
	b.WriteString(truncateScreen(fmt.Sprintf("Page %d/%d | showing %d-%d of %d | n/p next/prev 100 | j/k scroll | enter read | r refresh", t.page+1, t.pageCount(), t.page*topStoriesPerPage+t.listTop+1, t.page*topStoriesPerPage+end, len(t.storyIDs)), width))
	return b.String()
}

func (t Top) articleView(width, height int) string {
	article := t.articles[t.readID]
	rendered := renderMarkdown(article.Markdown, width)
	header := []string{article.Title}
	meta := articleMeta(article)
	if meta != "" {
		header = append(header, meta)
	}
	header = append(header, "esc back | j/k scroll | pgup/pgdn page")
	if t.err != "" {
		header = append(header, t.err)
	}
	contentHeight := maxScreen(1, height-len(header)-1)
	lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	maxTop := maxScreen(0, len(lines)-contentHeight)
	top := t.readTop
	if top > maxTop {
		top = maxTop
	}
	end := minScreen(len(lines), top+contentHeight)
	var b strings.Builder
	for _, line := range header {
		b.WriteString(truncateScreen(line, width) + "\n")
	}
	if end > top {
		b.WriteString(strings.Join(lines[top:end], "\n"))
	}
	return b.String()
}

func renderMarkdown(markdown string, width int) string {
	r, err := glamour.NewTermRenderer(glamour.WithWordWrap(maxScreen(20, width)))
	if err != nil {
		return markdown
	}
	out, err := r.Render(markdown)
	if err != nil {
		return markdown
	}
	return out
}

func articleMeta(article articles.Article) string {
	parts := make([]string, 0, 4)
	if article.Author != "" {
		parts = append(parts, "by "+article.Author)
	}
	if article.Date != "" {
		parts = append(parts, article.Date)
	}
	if article.URL != "" {
		parts = append(parts, article.URL)
	}
	return strings.Join(parts, " | ")
}

func hnTextMarkdown(story hn.Item) string {
	text := html.UnescapeString(story.Text)
	text = strings.ReplaceAll(text, "<p>", "\n\n")
	text = strings.ReplaceAll(text, "<pre><code>", "\n\n```")
	text = strings.ReplaceAll(text, "</code></pre>", "```\n\n")
	return "# " + story.Title + "\n\n" + text
}

func storyDomain(raw string) string {
	if raw == "" {
		return "news.ycombinator.com"
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Hostname(), "www.")
}

func clampIndex(idx, length int) int {
	if length == 0 {
		return 0
	}
	if idx < 0 {
		return 0
	}
	if idx >= length {
		return length - 1
	}
	return idx
}

func (t Top) pageCount() int {
	if len(t.storyIDs) == 0 {
		return 1
	}
	return (len(t.storyIDs) + topStoriesPerPage - 1) / topStoriesPerPage
}

func (t Top) selectedInPage() bool {
	return t.selected >= 0 && t.selected < len(t.stories)
}

func clampPage(page, length int) int {
	if length <= 0 {
		return 0
	}
	pages := (length + topStoriesPerPage - 1) / topStoriesPerPage
	if page < 0 {
		return 0
	}
	if page >= pages {
		return pages - 1
	}
	return page
}

func truncateScreen(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width == 1 {
		return "."
	}
	return s[:width-1] + "."
}

func minScreen(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxScreen(a, b int) int {
	if a > b {
		return a
	}
	return b
}
