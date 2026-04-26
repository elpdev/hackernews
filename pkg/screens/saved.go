package screens

import (
	"charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/hackernews/internal/browser"
	"github.com/elpdev/hackernews/internal/clipboard"
	"github.com/elpdev/hackernews/pkg/articles"
	"github.com/elpdev/hackernews/pkg/saved"
)

type Saved struct {
	store     saved.Store
	extractor articles.Extractor
	opener    func(string) error
	copier    func(string) error
	items     []saved.Article
	selected  int
	listTop   int
	readID    int
	readLine  int
	loading   string
	err       string
	status    string

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

func NewSaved(store saved.Store) Saved {
	return Saved{store: store, extractor: articles.NewTrafilaturaExtractor(), opener: browser.Open, copier: clipboard.Copy, loading: "Loading saved articles..."}
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
	case savedArticleExtractedMsg:
		s.loading = ""
		if msg.err != nil {
			s.status = "Could not extract saved article: " + msg.err.Error()
			return s, nil
		}
		for i := range s.items {
			if s.items[i].ID == msg.id {
				s.items[i].Article = msg.article
				break
			}
		}
		clearArticleRenderCache(msg.id)
		s.readID = msg.id
		s.readLine = 0
		s.status = "Article extracted"
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
