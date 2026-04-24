package screens

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/internal/articles"
	"github.com/elpdev/hackernews/internal/hn"
	"github.com/elpdev/hackernews/internal/media"
	"github.com/elpdev/hackernews/internal/saved"
)

var articleRenderCache = struct {
	sync.Mutex
	lines map[string][]string
}{lines: make(map[string][]string)}

const (
	topStoryLimit     = 500
	topStoriesPerPage = 100
	articleImageLimit = 10 << 20
	articleMaxWidth   = 100
)

type Top struct {
	client    hn.Client
	extractor articles.Extractor
	saved     saved.Store
	storyIDs  []int
	stories   []hn.Item
	pages     map[int][]hn.Item
	articles  map[int]articles.Article
	images    map[int]articleImage
	savedIDs  map[int]bool

	selected int
	page     int
	listTop  int
	readID   int
	readTop  int
	readLine int
	loading  string
	err      string
	status   string

	searching   bool
	searchQuery string
	sortMode    sortMode
}

type sortMode int

const (
	sortDefault sortMode = iota
	sortRecent
	sortPoints
)

func (m sortMode) label() string {
	switch m {
	case sortRecent:
		return "recent"
	case sortPoints:
		return "points"
	default:
		return ""
	}
}

type storyListItem struct {
	index int
	story hn.Item
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

type savedIDsLoadedMsg struct {
	ids map[int]bool
	err error
}

type articleSavedToggledMsg struct {
	id    int
	saved bool
	err   error
}

type articleImage struct {
	url   string
	bytes []byte
	err   string
}

type articleImageLoadedMsg struct {
	id    int
	url   string
	bytes []byte
	err   error
}

func NewTop(stores ...saved.Store) Top {
	var store saved.Store
	if len(stores) > 0 {
		store = stores[0]
	}
	return Top{
		client:    hn.NewClient(nil),
		extractor: articles.NewTrafilaturaExtractor(),
		saved:     store,
		pages:     make(map[int][]hn.Item),
		articles:  make(map[int]articles.Article),
		images:    make(map[int]articleImage),
		savedIDs:  make(map[int]bool),
		loading:   "Loading top stories...",
	}
}

func (t Top) Init() tea.Cmd { return tea.Batch(t.loadStories(), t.loadSavedIDs()) }

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
		t.readLine = 0
		return t.startArticleImageLoad(msg.id, msg.article)
	case savedIDsLoadedMsg:
		if msg.err != nil {
			t.status = "Could not load saved articles: " + msg.err.Error()
			return t, nil
		}
		t.savedIDs = msg.ids
		return t, nil
	case articleSavedToggledMsg:
		if msg.err != nil {
			t.status = "Could not update saved article: " + msg.err.Error()
			return t, nil
		}
		if t.savedIDs == nil {
			t.savedIDs = make(map[int]bool)
		}
		if msg.saved {
			t.savedIDs[msg.id] = true
			t.status = "Article saved"
		} else {
			delete(t.savedIDs, msg.id)
			t.status = "Article removed from saved"
		}
		return t, nil
	case articleImageLoadedMsg:
		if msg.err != nil {
			t.images[msg.id] = articleImage{url: msg.url, err: msg.err.Error()}
		} else {
			t.images[msg.id] = articleImage{url: msg.url, bytes: msg.bytes}
		}
		clearArticleRenderCache(msg.id)
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
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "clear search")),
		key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "sort")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "read")),
		key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save/unsave")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func (t Top) CapturesKey(msg tea.KeyPressMsg) bool {
	return t.searching && t.readID == 0 && msg.String() != "ctrl+c"
}

func (t Top) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if t.readID != 0 {
		switch msg.String() {
		case "s":
			return t, t.toggleSaved(t.readID)
		case "esc":
			t.readID = 0
			t.readTop = 0
			t.readLine = 0
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
		}
		t.readLine = clampIndex(t.readLine, cachedArticleLineCount(t.readID))
		return t, nil
	}

	if t.searching {
		return t.handleSearchKey(msg)
	}

	switch msg.String() {
	case "r":
		t.loading = "Loading top stories..."
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
	case "o":
		t.sortMode = (t.sortMode + 1) % 3
		t.selected = 0
		t.listTop = 0
		return t, nil
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
		if article, ok := t.articles[story.ID]; ok {
			t.readID = story.ID
			t.readTop = 0
			t.readLine = 0
			return t.startArticleImageLoad(story.ID, article)
		}
		t.loading = "Fetching article..."
		t.err = ""
		return t, t.loadArticle(story)
	case "s":
		return t, t.toggleSaved(matches[t.selected].story.ID)
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

func (t Top) loadSavedIDs() tea.Cmd {
	if t.saved == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		items, err := t.saved.List(ctx)
		if err != nil {
			return savedIDsLoadedMsg{err: err}
		}
		ids := make(map[int]bool, len(items))
		for _, item := range items {
			ids[item.ID] = true
		}
		return savedIDsLoadedMsg{ids: ids}
	}
}

func (t Top) toggleSaved(id int) tea.Cmd {
	if t.saved == nil {
		return func() tea.Msg {
			return articleSavedToggledMsg{id: id, err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	story, ok := t.storyByID(id)
	if !ok {
		story = hn.Item{ID: id, Type: "story"}
	}
	article := t.articleForStory(story)
	alreadySaved := t.savedIDs[id]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if alreadySaved {
			return articleSavedToggledMsg{id: id, saved: false, err: t.saved.Delete(ctx, id)}
		}
		return articleSavedToggledMsg{id: id, saved: true, err: t.saved.Save(ctx, story, article)}
	}
}

func (t Top) storyByID(id int) (hn.Item, bool) {
	for _, page := range t.pages {
		for _, story := range page {
			if story.ID == id {
				return story, true
			}
		}
	}
	for _, story := range t.stories {
		if story.ID == id {
			return story, true
		}
	}
	return hn.Item{}, false
}

func (t Top) articleForStory(story hn.Item) articles.Article {
	if article, ok := t.articles[story.ID]; ok {
		return article
	}
	articleURL := strings.TrimSpace(story.URL)
	if articleURL == "" {
		articleURL = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)
	}
	article := articles.Article{Title: story.Title, Author: story.By, URL: articleURL}
	if strings.TrimSpace(story.Text) != "" {
		article.Markdown = hnTextMarkdown(story)
	}
	return article
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

func (t Top) startArticleImageLoad(id int, article articles.Article) (Top, tea.Cmd) {
	imageURL := resolveArticleImageURL(article)
	if imageURL == "" {
		return t, nil
	}
	current := t.images[id]
	if current.url == imageURL && len(current.bytes) > 0 {
		return t, nil
	}
	t.images[id] = articleImage{url: imageURL}
	clearArticleRenderCache(id)
	return t, t.loadArticleImage(id, imageURL)
}

func resolveArticleImageURL(article articles.Article) string {
	imageURL := strings.TrimSpace(article.Image)
	if imageURL == "" {
		return ""
	}
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		return parsed.String()
	}
	baseURL := strings.TrimSpace(article.URL)
	if baseURL == "" {
		if strings.HasPrefix(imageURL, "//") {
			return "https:" + imageURL
		}
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return base.ResolveReference(parsed).String()
}

func (t Top) loadArticleImage(id int, rawURL string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return articleImageLoadedMsg{id: id, url: rawURL, err: err}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return articleImageLoadedMsg{id: id, url: rawURL, err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return articleImageLoadedMsg{id: id, url: rawURL, err: fmt.Errorf("image returned %s", resp.Status)}
		}
		bytes, err := io.ReadAll(io.LimitReader(resp.Body, articleImageLimit+1))
		if err != nil {
			return articleImageLoadedMsg{id: id, url: rawURL, err: err}
		}
		if len(bytes) > articleImageLimit {
			return articleImageLoadedMsg{id: id, url: rawURL, err: fmt.Errorf("image is larger than %d bytes", articleImageLimit)}
		}
		return articleImageLoadedMsg{id: id, url: rawURL, bytes: bytes}
	}
}

func (t Top) listView(width, height int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Top Hacker News"))
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
			b.WriteString("Press r to load top stories.\n")
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
	} else if sortLabel != "" {
		b.WriteString(truncateScreen("Sort: "+sortLabel, width) + "\n")
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
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	for i := t.listTop; i < end; i++ {
		item := matches[i]
		story := item.story
		line := fmt.Sprintf("%d. %s", item.index+1, story.Title)
		if domain := storyDomain(story.URL); domain != "" {
			line += " (" + domain + ")"
		}
		meta := fmt.Sprintf("   %d points by %s | %d comments", story.Score, story.By, story.Descendants)
		if t.savedIDs[story.ID] {
			meta += " | saved"
		}
		if i == selectedInPage {
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
	loadedCount := 0
	for _, page := range t.pages {
		loadedCount += len(page)
	}
	var footer string
	switch {
	case t.searchQuery != "" && sortLabel != "":
		footer = fmt.Sprintf("%d loaded | sort: %s | %d matches | o cycle sort | / edit search | ctrl+u clear | enter read | s save", loadedCount, sortLabel, len(matches))
	case t.searchQuery != "":
		footer = fmt.Sprintf("%d loaded | %d matches | / edit search | ctrl+u clear | o sort | enter read | s save", loadedCount, len(matches))
	case sortLabel != "":
		footer = fmt.Sprintf("%d loaded | sort: %s | showing %d-%d of %d | o cycle sort | / search | enter read | s save", loadedCount, sortLabel, t.listTop+1, end, len(matches))
	default:
		footer = fmt.Sprintf("Page %d/%d | showing %d-%d of %d | / search | o sort | n/p next/prev 100 | j/k scroll | enter read | s save | r refresh", t.page+1, t.pageCount(), matches[t.listTop].index+1, matches[end-1].index+1, len(t.storyIDs))
	}
	b.WriteString(truncateScreen(footer, width))
	return b.String()
}

// allLoadedStories returns every story cached in t.pages, ordered by HN rank.
// Each item's index is its global rank (position in t.storyIDs).
func (t Top) allLoadedStories() []storyListItem {
	loaded := make(map[int]hn.Item, len(t.pages)*topStoriesPerPage)
	for _, page := range t.pages {
		for _, story := range page {
			loaded[story.ID] = story
		}
	}
	for _, story := range t.stories {
		loaded[story.ID] = story
	}
	items := make([]storyListItem, 0, len(loaded))
	for rank, id := range t.storyIDs {
		if story, ok := loaded[id]; ok {
			items = append(items, storyListItem{index: rank, story: story})
		}
	}
	return items
}

func (t Top) filteredStories() []storyListItem {
	var scope []storyListItem
	if t.searchQuery != "" || t.sortMode != sortDefault {
		scope = t.allLoadedStories()
	} else {
		scope = make([]storyListItem, 0, len(t.stories))
		base := t.page * topStoriesPerPage
		for i, story := range t.stories {
			scope = append(scope, storyListItem{index: base + i, story: story})
		}
	}
	query := strings.ToLower(strings.TrimSpace(t.searchQuery))
	if query == "" {
		return scope
	}
	out := scope[:0]
	for _, item := range scope {
		if storyMatchesQuery(item.story, query) {
			out = append(out, item)
		}
	}
	return out
}

func (t Top) sortedFilteredStories() []storyListItem {
	items := t.filteredStories()
	switch t.sortMode {
	case sortRecent:
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].story.Time > items[j].story.Time
		})
	case sortPoints:
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].story.Score > items[j].story.Score
		})
	}
	return items
}

func storyMatchesQuery(story hn.Item, query string) bool {
	fields := []string{story.Title, story.By, story.URL, storyDomain(story.URL)}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func (t Top) articleView(width, height int) string {
	article := t.articles[t.readID]
	saveHelp := "s save"
	if t.savedIDs[t.readID] {
		saveHelp = "s unsave"
	}
	header := []string{"esc back | " + saveHelp + " | j/k move highlight | pgup/pgdn jump"}
	if t.err != "" {
		header = append(header, t.err)
	}
	if t.status != "" {
		header = append(header, t.status)
	}
	contentHeight := maxScreen(1, height-len(header)-1)
	contentWidth := articleContentWidth(width)
	lines := renderedArticleLines(t.readID, contentWidth, article, t.images[t.readID])
	maxTop := maxScreen(0, len(lines)-contentHeight)
	cursor := clampIndex(t.readLine, len(lines))
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

func articleContentWidth(width int) int {
	return minScreen(width, articleMaxWidth)
}

func containsInlineImage(line string) bool {
	return strings.Contains(line, "\x1b_G") || strings.Contains(line, "]1337;") || strings.Contains(line, "\x1bP1;1q")
}

func articleLineHighlight(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FDE68A")).
		Background(lipgloss.Color("#334155")).
		MaxWidth(maxScreen(0, width))
}

func padLine(line string, width int) string {
	if width <= 0 {
		return line
	}
	lineWidth := lipgloss.Width(line)
	if lineWidth >= width {
		return line
	}
	return line + strings.Repeat(" ", width-lineWidth)
}

func renderedArticleLines(id, width int, article articles.Article, image articleImage) []string {
	key := fmt.Sprintf("%d:%d:%s", id, width, image.cacheKey())
	articleRenderCache.Lock()
	if lines, ok := articleRenderCache.lines[key]; ok {
		articleRenderCache.Unlock()
		return lines
	}
	articleRenderCache.Unlock()

	rendered := renderArticle(article, image, width)
	lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	articleRenderCache.Lock()
	articleRenderCache.lines[key] = lines
	articleRenderCache.Unlock()
	return lines
}

func (i articleImage) cacheKey() string {
	if i.url == "" {
		return "none"
	}
	if len(i.bytes) > 0 {
		return fmt.Sprintf("loaded:%s:%d", i.url, len(i.bytes))
	}
	if i.err != "" {
		return "err:" + i.url
	}
	return "loading:" + i.url
}

func clearArticleRenderCache(id int) {
	prefix := fmt.Sprintf("%d:", id)
	articleRenderCache.Lock()
	defer articleRenderCache.Unlock()
	for key := range articleRenderCache.lines {
		if strings.HasPrefix(key, prefix) {
			delete(articleRenderCache.lines, key)
		}
	}
}

func cachedArticleLineCount(id int) int {
	prefix := fmt.Sprintf("%d:", id)
	articleRenderCache.Lock()
	defer articleRenderCache.Unlock()
	for key, lines := range articleRenderCache.lines {
		if strings.HasPrefix(key, prefix) {
			return len(lines)
		}
	}
	return 0
}

func renderMarkdown(markdown string, width int) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(maxScreen(20, width)),
	)
	if err != nil {
		return markdown
	}
	out, err := r.Render(markdown)
	if err != nil {
		return markdown
	}
	return out
}

func renderArticle(article articles.Article, image articleImage, width int) string {
	sections := make([]string, 0, 4)
	if article.Title != "" {
		sections = append(sections, renderMarkdown("# "+article.Title+"\n", width))
	}
	if block := articleImageBlock(article, image, width); block != "" {
		sections = append(sections, block)
	}
	hasMeta := false
	if meta := articleMeta(article); meta != "" {
		sections = append(sections, renderMarkdown("> "+meta+"\n", width))
		hasMeta = true
	}
	hasBody := strings.TrimSpace(article.Markdown) != ""
	if hasBody {
		sections = append(sections, renderMarkdown(article.Markdown, width))
	}
	trimmed := trimRenderedSections(sections)
	if hasMeta && hasBody && len(trimmed) >= 2 {
		return strings.Join(trimmed[:len(trimmed)-1], "\n") + "\n\n" + trimmed[len(trimmed)-1]
	}
	return strings.Join(trimmed, "\n")
}

func articleImageBlock(article articles.Article, image articleImage, width int) string {
	imageURL := strings.TrimSpace(article.Image)
	if imageURL == "" {
		return ""
	}
	if len(image.bytes) == 0 {
		if image.err == "" {
			return "Image: loading..."
		}
		if image.url != "" {
			return "Image: " + image.url
		}
		return "Image: " + imageURL
	}
	imageWidth := minScreen(maxScreen(12, width-6), 48)
	block, _, err := media.RenderBytes(image.bytes, imageWidth)
	if err != nil || block == "" {
		return "Image: " + imageURL
	}
	return block
}

func trimRenderedSections(sections []string) []string {
	trimmed := sections[:0]
	for _, section := range sections {
		section = strings.Trim(section, "\n")
		if section != "" {
			trimmed = append(trimmed, section)
		}
	}
	return trimmed
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
