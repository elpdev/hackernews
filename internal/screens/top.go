package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/hackernews/internal/articles"
	"github.com/elpdev/hackernews/internal/browser"
	"github.com/elpdev/hackernews/internal/clipboard"
	"github.com/elpdev/hackernews/internal/history"
	"github.com/elpdev/hackernews/internal/hn"
	"github.com/elpdev/hackernews/internal/saved"
)

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

type StorySnapshot struct {
	ScreenID string
	FeedName string
	Rank     int
	Story    hn.Item
}

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
