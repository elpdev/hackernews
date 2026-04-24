package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/hackernews/internal/browser"
	"github.com/elpdev/hackernews/internal/clipboard"
	"github.com/elpdev/hackernews/internal/hn"
)

const (
	commentMaxDepth = 8
	commentMaxCount = 500
)

type Comments struct {
	client hn.Client
	opener func(string) error
	copier func(string) error

	story     hn.Item
	tree      map[int]hn.Item
	order     []commentLine
	collapsed map[int]bool
	parentOf  map[int]int

	selected int
	loading  string
	err      string
	status   string
	returnTo string

	searching    bool
	searchQuery  string
	allCollapsed bool
}

type commentLine struct {
	id    int
	depth int
}

func NewComments(client hn.Client) Comments {
	return Comments{
		client:    client,
		opener:    browser.Open,
		copier:    clipboard.Copy,
		tree:      make(map[int]hn.Item),
		collapsed: make(map[int]bool),
		parentOf:  make(map[int]int),
	}
}

// Open resets state for a new story and returns a command that fetches its
// comment tree. Called by the app dispatcher before switching to this screen.
func (c Comments) Open(story hn.Item, returnTo string) (Comments, tea.Cmd) {
	c.story = story
	c.returnTo = returnTo
	c.tree = map[int]hn.Item{story.ID: story}
	c.order = nil
	c.collapsed = make(map[int]bool)
	c.parentOf = make(map[int]int)
	c.selected = 0
	c.err = ""
	c.status = ""
	c.loading = "Loading comments..."
	return c, c.loadTree()
}

func (c Comments) Init() tea.Cmd { return nil }

func (c Comments) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case commentsTreeLoadedMsg:
		c.loading = ""
		if msg.storyID != c.story.ID {
			return c, nil
		}
		if msg.err != nil {
			c.err = msg.err.Error()
		} else {
			c.err = ""
		}
		if msg.tree != nil {
			c.tree = msg.tree
			if root, ok := msg.tree[c.story.ID]; ok {
				c.story = root
			}
			c.parentOf = buildParentMap(msg.tree)
		}
		c.order = c.buildOrder()
		c.selected = clampIndex(c.selected, len(c.order))
		return c, nil
	case tea.KeyPressMsg:
		return c.handleKey(msg)
	}
	return c, nil
}

func (c Comments) Title() string {
	if strings.TrimSpace(c.story.Title) != "" {
		return "Comments · " + c.story.Title
	}
	return "Comments"
}

func (c Comments) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "prev")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "next")),
		key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
		key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
		key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		key.NewBinding(key.WithKeys("left", "p"), key.WithHelp("left/p", "prev thread/match")),
		key.NewBinding(key.WithKeys("right", "n"), key.WithHelp("right/n", "next thread/match")),
		key.NewBinding(key.WithKeys("space", "enter"), key.WithHelp("space/enter", "collapse")),
		key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "parent")),
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "collapse all")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "clear search")),
		key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
		key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy url")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func (c Comments) CapturesKey(msg tea.KeyPressMsg) bool {
	return c.searching && msg.String() != "ctrl+c"
}
