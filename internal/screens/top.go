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
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/internal/articles"
	"github.com/elpdev/hackernews/internal/browser"
	"github.com/elpdev/hackernews/internal/clipboard"
	"github.com/elpdev/hackernews/internal/history"
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
	feed      hn.Feed
	client    hn.Client
	extractor articles.Extractor
	opener    func(string) error
	copier    func(string) error
	saved     saved.Store
	history   history.Store
	storyIDs  []int
	stories   []hn.Item
	pages     map[int][]hn.Item
	articles  map[int]articles.Article
	images    map[int]articleImage
	savedIDs  map[int]bool
	readIDs   map[int]bool

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
	hideRead    bool
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

func (m sortMode) String() string {
	s := m.label()
	if s == "" {
		return "default"
	}
	return s
}

func sortModeFromString(value string) sortMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "recent":
		return sortRecent
	case "points":
		return sortPoints
	default:
		return sortDefault
	}
}

type storyListItem struct {
	index int
	story hn.Item
}

type StorySnapshot struct {
	ScreenID string
	FeedName string
	Rank     int
	Story    hn.Item
}

type topStoriesLoadedMsg struct {
	screenID string
	ids      []int
	stories  []hn.Item
	err      error
}

func (m topStoriesLoadedMsg) TargetScreenID() string { return m.screenID }

type storyPageLoadedMsg struct {
	screenID string
	page     int
	stories  []hn.Item
	err      error
}

func (m storyPageLoadedMsg) TargetScreenID() string { return m.screenID }

type articleLoadedMsg struct {
	screenID string
	id       int
	article  articles.Article
	err      error
}

func (m articleLoadedMsg) TargetScreenID() string { return m.screenID }

type savedIDsLoadedMsg struct {
	screenID string
	ids      map[int]bool
	err      error
}

func (m savedIDsLoadedMsg) TargetScreenID() string { return m.screenID }

type articleSavedToggledMsg struct {
	screenID string
	id       int
	article  articles.Article
	saved    bool
	err      error
}

func (m articleSavedToggledMsg) TargetScreenID() string { return m.screenID }

type readIDsLoadedMsg struct {
	screenID string
	ids      map[int]bool
	err      error
}

func (m readIDsLoadedMsg) TargetScreenID() string { return m.screenID }

type storyMarkedReadMsg struct {
	screenID string
	id       int
	err      error
}

func (m storyMarkedReadMsg) TargetScreenID() string { return m.screenID }

type articleImage struct {
	url   string
	bytes []byte
	err   string
}

type articleImageLoadedMsg struct {
	screenID string
	id       int
	url      string
	bytes    []byte
	err      error
}

func (m articleImageLoadedMsg) TargetScreenID() string { return m.screenID }

func NewTop(stores ...saved.Store) Top {
	var store saved.Store
	if len(stores) > 0 {
		store = stores[0]
	}
	return NewStories(store, hn.FeedTop)
}

func NewStories(store saved.Store, feed hn.Feed, options ...any) Top {
	if feed == "" {
		feed = hn.FeedTop
	}
	var historyStore history.Store
	var hideRead bool
	sort := sortDefault
	for _, option := range options {
		switch value := option.(type) {
		case history.Store:
			historyStore = value
		case bool:
			hideRead = value
		case string:
			sort = sortModeFromString(value)
		}
	}
	return Top{
		feed:      feed,
		client:    hn.NewClient(nil),
		extractor: articles.NewTrafilaturaExtractor(),
		opener:    browser.Open,
		copier:    clipboard.Copy,
		saved:     store,
		history:   historyStore,
		pages:     make(map[int][]hn.Item),
		articles:  make(map[int]articles.Article),
		images:    make(map[int]articleImage),
		savedIDs:  make(map[int]bool),
		readIDs:   make(map[int]bool),
		loading:   "Loading " + strings.ToLower(feed.Title()) + "...",
		sortMode:  sort,
		hideRead:  hideRead,
	}
}

func (t Top) Init() tea.Cmd { return tea.Batch(t.loadStories(), t.loadSavedIDs(), t.loadReadIDs()) }

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
		} else {
			t.err = ""
		}
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
		t.loading = ""
		if msg.err != nil {
			t.status = "Could not update saved article: " + msg.err.Error()
			return t, nil
		}
		if t.savedIDs == nil {
			t.savedIDs = make(map[int]bool)
		}
		if msg.saved {
			t.savedIDs[msg.id] = true
			if strings.TrimSpace(msg.article.Markdown) != "" {
				t.articles[msg.id] = msg.article
			}
			t.status = "Article saved"
		} else {
			delete(t.savedIDs, msg.id)
			t.status = "Article removed from saved"
		}
		return t, nil
	case readIDsLoadedMsg:
		if msg.err != nil {
			t.status = "Could not load read history: " + msg.err.Error()
			return t, nil
		}
		t.readIDs = msg.ids
		return t, nil
	case storyMarkedReadMsg:
		if msg.err != nil {
			t.status = "Could not update read history: " + msg.err.Error()
			return t, nil
		}
		if t.readIDs == nil {
			t.readIDs = make(map[int]bool)
		}
		t.readIDs[msg.id] = true
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

func (t Top) Title() string {
	if t.feed == "" {
		return hn.FeedTop.Title()
	}
	return t.feed.Title()
}

func (t Top) Snapshot() []StorySnapshot {
	items := t.allLoadedStories()
	out := make([]StorySnapshot, 0, len(items))
	for _, item := range items {
		out = append(out, StorySnapshot{ScreenID: t.screenID(), FeedName: t.feedTitle(), Rank: item.index, Story: item.story})
	}
	return out
}

func (t Top) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
		key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
		key.NewBinding(key.WithKeys("left", "p"), key.WithHelp("left/p", "prev page/paragraph")),
		key.NewBinding(key.WithKeys("right", "n"), key.WithHelp("right/n", "next page/paragraph")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "clear search")),
		key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
		key.NewBinding(key.WithKeys("O"), key.WithHelp("O", "cycle sort")),
		key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "hide read")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "read")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "comments")),
		key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save/unsave")),
		key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy url")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func (t *Top) SetHideRead(hide bool) {
	t.hideRead = hide
	t.selected = 0
	t.listTop = 0
}

func (t *Top) SetSortMode(mode string) {
	t.sortMode = sortModeFromString(mode)
	t.selected = 0
	t.listTop = 0
}

func (t Top) CapturesKey(msg tea.KeyPressMsg) bool {
	return t.searching && t.readID == 0 && msg.String() != "ctrl+c"
}

func (t Top) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if t.readID != 0 {
		switch msg.String() {
		case "s":
			return t, t.toggleSaved(t.readID)
		case "o":
			t.status = t.openArticleURL(t.articles[t.readID].URL)
			return t, nil
		case "y":
			t.status = t.copyArticleURL(t.articleURLForID(t.readID))
			return t, nil
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
		if article, ok := t.articles[story.ID]; ok {
			t.readID = story.ID
			t.readTop = 0
			t.readLine = 0
			t, cmd := t.startArticleImageLoad(story.ID, article)
			return t, tea.Batch(markRead, cmd)
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

func (t Top) loadStories() tea.Cmd {
	feed := t.feed
	if feed == "" {
		feed = hn.FeedTop
	}
	screenID := t.screenID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		ids, err := t.client.StoryIDs(ctx, feed)
		if err != nil {
			return topStoriesLoadedMsg{screenID: screenID, err: err}
		}
		if len(ids) > topStoryLimit {
			ids = ids[:topStoryLimit]
		}
		end := minScreen(len(ids), topStoriesPerPage)
		stories, err := t.client.Stories(ctx, ids[:end])
		return topStoriesLoadedMsg{screenID: screenID, ids: ids, stories: stories, err: err}
	}
}

func (t Top) feedTitle() string {
	if t.feed == "" {
		return hn.FeedTop.Title()
	}
	return t.feed.Title()
}

func (t Top) screenID() string {
	if t.feed == "" {
		return hn.FeedTop.ScreenID()
	}
	return t.feed.ScreenID()
}

func (t Top) listHeader() string {
	if t.feed == "" || t.feed == hn.FeedTop {
		return "Top Hacker News"
	}
	return "Hacker News · " + t.feed.Title()
}

func (t Top) loadSavedIDs() tea.Cmd {
	if t.saved == nil {
		return nil
	}
	screenID := t.screenID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		items, err := t.saved.List(ctx)
		if err != nil {
			return savedIDsLoadedMsg{screenID: screenID, err: err}
		}
		ids := make(map[int]bool, len(items))
		for _, item := range items {
			ids[item.ID] = true
		}
		return savedIDsLoadedMsg{screenID: screenID, ids: ids}
	}
}

func (t Top) loadReadIDs() tea.Cmd {
	if t.history == nil {
		return nil
	}
	screenID := t.screenID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ids, err := t.history.ReadIDs(ctx)
		return readIDsLoadedMsg{screenID: screenID, ids: ids, err: err}
	}
}

func (t Top) markRead(id int) tea.Cmd {
	if t.history == nil || id == 0 {
		return nil
	}
	screenID := t.screenID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return storyMarkedReadMsg{screenID: screenID, id: id, err: t.history.MarkRead(ctx, id)}
	}
}

func (t Top) toggleSaved(id int) tea.Cmd {
	if t.saved == nil {
		return func() tea.Msg {
			return articleSavedToggledMsg{screenID: t.screenID(), id: id, err: fmt.Errorf("saved article store is unavailable")}
		}
	}
	screenID := t.screenID()
	story, ok := t.storyByID(id)
	if !ok {
		story = hn.Item{ID: id, Type: "story"}
	}
	article := t.articleForStory(story)
	alreadySaved := t.savedIDs[id]
	return func() tea.Msg {
		if alreadySaved {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return articleSavedToggledMsg{screenID: screenID, id: id, saved: false, err: t.saved.Delete(ctx, id)}
		}
		if !t.hasExtractedArticle(story) {
			var err error
			article, err = t.extractArticleForStory(story)
			if err != nil {
				return articleSavedToggledMsg{screenID: screenID, id: id, err: err}
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return articleSavedToggledMsg{screenID: screenID, id: id, article: article, saved: true, err: t.saved.Save(ctx, story, article)}
	}
}

func (t Top) hasExtractedArticle(story hn.Item) bool {
	article, ok := t.articles[story.ID]
	if ok && strings.TrimSpace(article.Markdown) != "" {
		return true
	}
	return strings.TrimSpace(story.URL) == "" || strings.TrimSpace(story.Text) != ""
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
	articleURL := t.articleURLForStory(story)
	article := articles.Article{Title: story.Title, Author: story.By, URL: articleURL}
	if strings.TrimSpace(story.Text) != "" {
		article.Markdown = hnTextMarkdown(story)
	}
	return article
}

func (t Top) articleURLForID(id int) string {
	if article, ok := t.articles[id]; ok && strings.TrimSpace(article.URL) != "" {
		return article.URL
	}
	if story, ok := t.storyByID(id); ok {
		return t.articleURLForStory(story)
	}
	return ""
}

func (t Top) articleURLForStory(story hn.Item) string {
	if strings.TrimSpace(story.URL) != "" {
		return story.URL
	}
	if story.ID == 0 {
		return ""
	}
	return fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)
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
	screenID := t.screenID()
	return func() tea.Msg {
		start := page * topStoriesPerPage
		if start >= len(ids) {
			return storyPageLoadedMsg{screenID: screenID, page: page, stories: nil}
		}
		end := minScreen(len(ids), start+topStoriesPerPage)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		stories, err := t.client.Stories(ctx, ids[start:end])
		return storyPageLoadedMsg{screenID: screenID, page: page, stories: stories, err: err}
	}
}

func (t Top) openArticleURL(url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to open"
	}
	if t.opener == nil {
		return "Could not open browser: no opener configured"
	}
	if err := t.opener(url); err != nil {
		return "Could not open browser: " + err.Error()
	}
	return "Opening in browser..."
}

func (t Top) copyArticleURL(url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to copy"
	}
	if t.copier == nil {
		return "Could not copy URL: no clipboard configured"
	}
	if err := t.copier(url); err != nil {
		return "Could not copy URL: " + err.Error()
	}
	return "Copied URL to clipboard"
}

func (t Top) loadArticle(story hn.Item) tea.Cmd {
	screenID := t.screenID()
	return func() tea.Msg {
		article, err := t.extractArticleForStory(story)
		return articleLoadedMsg{screenID: screenID, id: story.ID, article: article, err: err}
	}
}

func (t Top) extractArticleForStory(story hn.Item) (articles.Article, error) {
	if strings.TrimSpace(story.URL) == "" {
		return articles.Article{
			Title:    story.Title,
			Author:   story.By,
			URL:      fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID),
			Markdown: hnTextMarkdown(story),
		}, nil
	}
	article, err := t.extractor.Extract(story.URL)
	if article.Title == "" {
		article.Title = story.Title
	}
	if article.Author == "" {
		article.Author = story.By
	}
	if strings.TrimSpace(article.URL) == "" {
		article.URL = story.URL
	}
	return article, err
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
	screenID := t.screenID()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, err: err}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, err: fmt.Errorf("image returned %s", resp.Status)}
		}
		bytes, err := io.ReadAll(io.LimitReader(resp.Body, articleImageLimit+1))
		if err != nil {
			return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, err: err}
		}
		if len(bytes) > articleImageLimit {
			return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, err: fmt.Errorf("image is larger than %d bytes", articleImageLimit)}
		}
		return articleImageLoadedMsg{screenID: screenID, id: id, url: rawURL, bytes: bytes}
	}
}

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
	if t.hideRead {
		out := scope[:0]
		for _, item := range scope {
			if !t.readIDs[item.story.ID] {
				out = append(out, item)
			}
		}
		scope = out
	}
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
	header := []string{"esc back | " + saveHelp + " | o open in browser | y copy url | j/k move | [/ ] paragraph"}
	header[0] = "esc back | " + saveHelp + " | o open | y copy | j/k line | left/right or p/n paragraph"
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

func cachedArticleLines(id int) []string {
	prefix := fmt.Sprintf("%d:", id)
	articleRenderCache.Lock()
	defer articleRenderCache.Unlock()
	for key, lines := range articleRenderCache.lines {
		if strings.HasPrefix(key, prefix) {
			return append([]string(nil), lines...)
		}
	}
	return nil
}

func nextParagraphLine(lines []string, cursor int) int {
	if len(lines) == 0 {
		return cursor
	}
	cursor = clampIndex(cursor, len(lines))
	for i := cursor + 1; i < len(lines); i++ {
		if isBlankRenderedLine(lines[i-1]) && !isBlankRenderedLine(lines[i]) {
			return i
		}
	}
	return len(lines) - 1
}

func previousParagraphLine(lines []string, cursor int) int {
	if len(lines) == 0 {
		return cursor
	}
	cursor = clampIndex(cursor, len(lines))
	for i := cursor - 1; i > 0; i-- {
		if isBlankRenderedLine(lines[i-1]) && !isBlankRenderedLine(lines[i]) {
			return i
		}
	}
	return 0
}

func isBlankRenderedLine(line string) bool {
	return strings.TrimSpace(ansi.Strip(line)) == ""
}

func renderMarkdown(markdown string, width int) string {
	markdown = repairLooseListItems(markdown)
	markdown = fenceLooseArticleCode(markdown)
	markdown = labelUnlabeledCodeFences(markdown)
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

func repairLooseListItems(markdown string) string {
	lines := strings.Split(markdown, "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !isMarkdownListItem(line) {
			out = append(out, line)
			continue
		}

		item := line
		for i+2 < len(lines) && strings.TrimSpace(lines[i+1]) == "" {
			next := strings.TrimSpace(lines[i+2])
			if next == "" || startsMarkdownBlock(next) {
				break
			}
			item = strings.TrimRight(item, " ") + " " + next
			i += 2
		}
		out = append(out, item)
	}
	return strings.Join(out, "\n")
}

func labelUnlabeledCodeFences(markdown string) string {
	lines := strings.Split(markdown, "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if !isUnlabeledCodeFence(trimmed) {
			out = append(out, line)
			continue
		}

		fenceMarker := trimmed[:3]
		end := codeFenceEnd(lines, i+1, fenceMarker)
		if end == -1 {
			out = append(out, line)
			continue
		}
		out = append(out, normalizeUnlabeledCodeFence(fenceMarker, lines[i+1:end])...)
		i = end
	}
	return strings.Join(out, "\n")
}

func isUnlabeledCodeFence(line string) bool {
	return line == "```" || line == "~~~"
}

func codeFenceEnd(lines []string, start int, marker string) int {
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), marker) {
			return i
		}
	}
	return -1
}

func normalizeUnlabeledCodeFence(marker string, lines []string) []string {
	if blocks := splitShellHeredocFence(marker, lines); len(blocks) > 0 {
		return blocks
	}
	if lang := inferCodeBlockLanguage(lines); lang != "" {
		return fencedCodeBlock(marker, lang, lines)
	}
	return fencedCodeBlock(marker, "", lines)
}

func fencedCodeBlock(marker, lang string, lines []string) []string {
	out := make([]string, 0, len(lines)+2)
	out = append(out, marker+lang)
	out = append(out, indentLooseCode(lang, lines)...)
	return append(out, marker)
}

func isMarkdownListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
		return true
	}
	for i, r := range trimmed {
		if r >= '0' && r <= '9' {
			continue
		}
		return i > 0 && (r == '.' || r == ')') && len(trimmed) > i+1 && trimmed[i+1] == ' '
	}
	return false
}

func startsMarkdownBlock(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, ">") ||
		strings.HasPrefix(trimmed, "```") ||
		strings.HasPrefix(trimmed, "~~~") ||
		strings.HasPrefix(trimmed, "---") ||
		strings.HasPrefix(trimmed, "***") ||
		strings.HasPrefix(trimmed, "___") ||
		strings.HasPrefix(trimmed, "| ") ||
		isMarkdownListItem(trimmed)
}

func fenceLooseArticleCode(markdown string) string {
	lines := strings.Split(markdown, "\n")
	langs := looseCodeLanguages(lines)
	out := make([]string, 0, len(lines)+len(langs)*2)
	inCode := false
	codeLang := "text"
	codeLines := make([]string, 0)
	for _, line := range lines {
		if isLooseCodeStart(line) {
			if inCode {
				out = appendLooseCodeBlock(out, codeLang, codeLines)
				out = append(out, "")
			}
			codeLang = "text"
			if len(langs) > 0 {
				codeLang = langs[0]
				langs = langs[1:]
			} else if inferred := inferLooseCodeLanguage(line); inferred != "" {
				codeLang = inferred
			}
			codeLines = append(codeLines[:0], cleanLooseCodeStart(line))
			inCode = true
			continue
		}
		if inCode && strings.HasPrefix(strings.TrimSpace(line), "##") {
			out = appendLooseCodeBlock(out, codeLang, codeLines)
			codeLines = codeLines[:0]
			inCode = false
		}
		if inCode {
			codeLines = append(codeLines, line)
			continue
		}
		out = append(out, line)
	}
	if inCode {
		out = appendLooseCodeBlock(out, codeLang, codeLines)
	}
	return strings.Join(out, "\n")
}

func appendLooseCodeBlock(out []string, lang string, lines []string) []string {
	out = append(out, "```"+lang)
	out = append(out, indentLooseCode(lang, compactLooseCodeLines(lines))...)
	return append(out, "```")
}

func compactLooseCodeLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = cleanLooseCodeStart(line)
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func indentLooseCode(lang string, lines []string) []string {
	switch lang {
	case "python", "javascript", "ruby", "go", "rust", "java", "cpp", "csharp", "php", "swift", "kotlin", "scala":
		return indentLooseBlockCode(lines)
	case "bash":
		return indentLooseBashCode(lines)
	default:
		return lines
	}
}

func indentLooseBlockCode(lines []string) []string {
	out := make([]string, 0, len(lines))
	indent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, "")
			continue
		}
		if startsBlockDedent(trimmed) {
			indent = maxScreen(0, indent-1)
		}
		out = append(out, strings.Repeat("  ", indent)+trimmed)
		indent = maxScreen(0, indent+blockIndentDelta(trimmed))
		if startsBlockContinuation(trimmed) {
			indent++
		}
	}
	return out
}

func startsBlockDedent(line string) bool {
	return startsWithClosingDelimiter(line) || line == "end" || line == "else" || strings.HasPrefix(line, "elsif ") || strings.HasPrefix(line, "elif ") || strings.HasPrefix(line, "when ") || strings.HasPrefix(line, "catch ") || strings.HasPrefix(line, "rescue") || line == "ensure" || strings.HasPrefix(line, "finally") || (strings.HasPrefix(line, "case ") && strings.HasSuffix(line, ":"))
}

func startsBlockContinuation(line string) bool {
	if line == "else" || strings.HasPrefix(line, "elsif ") || strings.HasPrefix(line, "elif ") || strings.HasPrefix(line, "when ") || strings.HasPrefix(line, "case ") || strings.HasPrefix(line, "catch ") || strings.HasPrefix(line, "rescue") || line == "ensure" || strings.HasPrefix(line, "finally") {
		return true
	}
	return strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "class ") || strings.HasPrefix(line, "module ") ||
		strings.HasPrefix(line, "if ") || strings.HasPrefix(line, "unless ") || strings.HasPrefix(line, "case ") ||
		strings.HasPrefix(line, "while ") || strings.HasPrefix(line, "until ") || strings.HasPrefix(line, "for ") ||
		strings.HasPrefix(line, "begin") || strings.HasSuffix(line, " do") || strings.HasSuffix(line, ":")
}

func blockIndentDelta(line string) int {
	delta := looseIndentDelta(line)
	if delta > 0 {
		return 1
	}
	if delta < 0 {
		return -1
	}
	return 0
}

func indentLooseBashCode(lines []string) []string {
	out := make([]string, 0, len(lines))
	inJSON := false
	jsonIndent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inJSON {
			content, quoted := strings.CutSuffix(trimmed, "'")
			if startsWithClosingDelimiter(content) {
				jsonIndent = maxScreen(0, jsonIndent-1)
			}
			if quoted {
				out = append(out, "  "+strings.Repeat("  ", jsonIndent)+content+"'")
				inJSON = false
				continue
			}
			out = append(out, "  "+strings.Repeat("  ", jsonIndent)+content)
			jsonIndent = maxScreen(0, jsonIndent+looseIndentDelta(content))
			continue
		}
		if strings.HasPrefix(trimmed, "-d '{") && strings.TrimSpace(strings.TrimPrefix(trimmed, "-d '")) == "{" {
			out = append(out, "  "+trimmed)
			inJSON = true
			jsonIndent = 1
			continue
		}
		if strings.HasPrefix(trimmed, "-H ") || strings.HasPrefix(trimmed, "-d ") {
			trimmed = "  " + trimmed
		}
		out = append(out, trimmed)
	}
	return out
}

func indentBracketedLooseCode(lines []string) []string {
	out := make([]string, 0, len(lines))
	indent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if startsWithClosingDelimiter(trimmed) {
			indent = maxScreen(0, indent-1)
		}
		out = append(out, strings.Repeat("  ", indent)+trimmed)
		indent = maxScreen(0, indent+looseIndentDelta(trimmed))
	}
	return out
}

func startsWithClosingDelimiter(line string) bool {
	return strings.HasPrefix(line, ")") || strings.HasPrefix(line, "]") || strings.HasPrefix(line, "}")
}

func looseIndentDelta(line string) int {
	delta := 0
	for _, r := range line {
		switch r {
		case '(', '[', '{':
			delta++
		case ')', ']', '}':
			delta--
		}
	}
	if delta > 1 {
		return 1
	}
	if delta < -1 {
		return -1
	}
	return delta
}

func looseCodeLanguages(lines []string) []string {
	langs := make([]string, 0, 3)
	for _, line := range lines {
		if lang := normalizeCodeLanguage(line); lang != "" {
			langs = append(langs, lang)
		}
	}
	return langs
}

func normalizeCodeLanguage(line string) string {
	lang := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-")))
	lang = strings.Trim(lang, "`:")
	return normalizeLanguageName(lang)
}

func normalizeLanguageName(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "curl", "shell", "bash", "sh", "zsh", "fish", "terminal", "console":
		return "bash"
	case "python", "py":
		return "python"
	case "nodejs", "node", "javascript", "js", "typescript", "ts":
		return "javascript"
	case "ruby", "rb", "rails":
		return "ruby"
	case "golang", "go":
		return "go"
	case "rust", "rs":
		return "rust"
	case "java":
		return "java"
	case "c", "cpp", "c++", "cc", "h", "hpp":
		return "cpp"
	case "c#", "csharp", "cs":
		return "csharp"
	case "kotlin", "kt":
		return "kotlin"
	case "yaml", "yml":
		return "yaml"
	case "php", "swift", "scala", "sql", "html", "css", "json", "xml", "toml", "dockerfile":
		return lang
	default:
		if lexer := lexers.Get(lang); lexer != nil {
			aliases := lexer.Config().Aliases
			if len(aliases) > 0 {
				return aliases[0]
			}
			return strings.ToLower(lexer.Config().Name)
		}
		return ""
	}
}

func splitShellHeredocFence(marker string, lines []string) []string {
	for i, line := range lines {
		delimiter, lang := heredocDelimiter(line)
		if delimiter == "" || lang == "" {
			continue
		}
		end := heredocEnd(lines, i+1, delimiter)
		if end == -1 {
			continue
		}

		out := make([]string, 0, len(lines)+6)
		if i > 0 {
			out = append(out, fencedCodeBlock(marker, "bash", lines[:i+1])...)
		} else {
			out = append(out, fencedCodeBlock(marker, "bash", lines[:1])...)
		}
		out = append(out, "")
		out = append(out, fencedCodeBlock(marker, lang, lines[i+1:end])...)
		if end < len(lines)-1 {
			out = append(out, "")
			out = append(out, fencedCodeBlock(marker, "bash", lines[end:])...)
		}
		return out
	}
	return nil
}

func heredocDelimiter(line string) (string, string) {
	idx := strings.Index(line, "<<")
	if idx == -1 {
		return "", ""
	}
	delimiter := strings.TrimSpace(line[idx+2:])
	delimiter = strings.TrimPrefix(delimiter, "-")
	delimiter = strings.TrimPrefix(delimiter, "~")
	delimiter = strings.Trim(delimiter, "'\"")
	if delimiter == "" {
		return "", ""
	}
	lang := normalizeLanguageName(delimiter)
	if lang == "" || lang == "text" {
		return delimiter, ""
	}
	return delimiter, lang
}

func heredocEnd(lines []string, start int, delimiter string) int {
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == delimiter {
			return i
		}
	}
	return -1
}

func inferCodeBlockLanguage(lines []string) string {
	text := strings.TrimSpace(strings.Join(lines, "\n"))
	if text == "" {
		return ""
	}
	if lang := inferCodeBlockLanguageFromSignals(lines, false); lang != "" {
		return lang
	}
	if lexer := lexers.Analyse(text); lexer != nil {
		if lang := normalizeLanguageName(lexer.Config().Name); lang != "" && lang != "text" {
			return lang
		}
		for _, alias := range lexer.Config().Aliases {
			if lang := normalizeLanguageName(alias); lang != "" && lang != "text" {
				return lang
			}
		}
	}
	return inferCodeBlockLanguageFromSignals(lines, true)
}

func inferCodeBlockLanguageFromSignals(lines []string, includeShell bool) string {
	var bashScore, rubyScore, goScore, rustScore, cScore int
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		switch {
		case strings.Contains(trimmed, "<<'RUBY'"), strings.Contains(trimmed, "<<\"RUBY\""), strings.Contains(trimmed, "<<RUBY"):
			rubyScore += 4
		case strings.HasPrefix(trimmed, "def "), strings.HasPrefix(trimmed, "class "), strings.HasPrefix(trimmed, "module "), strings.HasPrefix(trimmed, "require "), strings.HasPrefix(trimmed, "puts "):
			rubyScore += 3
		case strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, "attr_"):
			rubyScore += 2
		case strings.HasPrefix(trimmed, "package "), strings.HasPrefix(trimmed, "func "), strings.HasPrefix(trimmed, "type ") && strings.Contains(trimmed, " struct"):
			goScore += 3
		case strings.HasPrefix(trimmed, "use "), strings.HasPrefix(trimmed, "fn "), strings.HasPrefix(trimmed, "impl "), strings.HasPrefix(trimmed, "let mut "):
			rustScore += 3
		case strings.HasPrefix(trimmed, "#include"), strings.HasPrefix(trimmed, "int main"), strings.HasPrefix(trimmed, "static "):
			cScore += 3
		case includeShell && (strings.HasPrefix(lower, "make") || strings.HasPrefix(trimmed, "./") || strings.HasPrefix(lower, "sudo ") || strings.HasPrefix(lower, "cat >") || strings.HasPrefix(trimmed, "$")):
			bashScore += 2
		case includeShell && strings.HasPrefix(trimmed, "#"):
			bashScore++
		}
	}
	if rubyScore >= 4 && rubyScore >= bashScore {
		return "ruby"
	}
	if bashScore >= 2 {
		return "bash"
	}
	if rubyScore >= 3 {
		return "ruby"
	}
	if goScore >= 3 {
		return "go"
	}
	if rustScore >= 3 {
		return "rust"
	}
	if cScore >= 3 {
		return "cpp"
	}
	return ""
}

func isLooseCodeStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") || !strings.HasPrefix(trimmed, "`") {
		return false
	}
	return inferLooseCodeLanguage(trimmed) != ""
}

func inferLooseCodeLanguage(line string) string {
	code := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "`"))
	switch {
	case strings.HasPrefix(code, "curl "), strings.HasPrefix(code, "-H "), strings.HasPrefix(code, "-d "):
		return "bash"
	case strings.HasPrefix(code, "#"), strings.HasPrefix(code, "import "), strings.HasPrefix(code, "from "):
		return "python"
	case strings.HasPrefix(code, "//"), strings.HasPrefix(code, "const "), strings.HasPrefix(code, "let "), strings.HasPrefix(code, "async function"):
		return "javascript"
	case strings.HasPrefix(code, "require "), strings.HasPrefix(code, "module "), strings.HasPrefix(code, "def "), strings.HasPrefix(code, "puts "):
		return "ruby"
	case strings.HasPrefix(code, "class ") && !strings.Contains(code, "{"):
		return "ruby"
	case strings.HasPrefix(code, "package main"), strings.HasPrefix(code, "func "):
		return "go"
	case strings.HasPrefix(code, "use "), strings.HasPrefix(code, "fn "), strings.HasPrefix(code, "impl "):
		return "rust"
	default:
		return ""
	}
}

func cleanLooseCodeStart(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimLeft(line, "`")
	line = strings.TrimRight(line, "`")
	return strings.ReplaceAll(line, "`", "")
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
	} else if url := strings.TrimSpace(article.URL); url != "" {
		sections = append(sections, renderMarkdown(articleFallbackBody(url), width))
	}
	trimmed := trimRenderedSections(sections)
	if hasMeta && hasBody && len(trimmed) >= 2 {
		return strings.Join(trimmed[:len(trimmed)-1], "\n") + "\n\n" + trimmed[len(trimmed)-1]
	}
	return strings.Join(trimmed, "\n")
}

func articleFallbackBody(url string) string {
	return "---\n\n" +
		"Couldn't extract readable content from this page.\n\n" +
		"Press `o` to open in your browser:\n\n" +
		url + "\n"
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
