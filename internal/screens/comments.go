package screens

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/x/ansi"
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

type commentsTreeLoadedMsg struct {
	screenID string
	storyID  int
	tree     map[int]hn.Item
	err      error
}

func (m commentsTreeLoadedMsg) TargetScreenID() string { return m.screenID }

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

func (c Comments) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	var head strings.Builder
	for _, line := range commentsHeaderBlock(c.story, width) {
		head.WriteString(line + "\n")
	}
	if c.loading != "" {
		head.WriteString(truncateScreen(c.loading, width) + "\n")
	}
	if c.err != "" {
		head.WriteString(truncateScreen(c.err, width) + "\n")
	}
	if c.status != "" {
		head.WriteString(truncateScreen(c.status, width) + "\n")
	}
	if c.searching || c.searchQuery != "" {
		label := "Filter"
		if c.searching {
			label = "Search"
		}
		query := c.searchQuery
		if query == "" {
			query = lipgloss.NewStyle().Faint(true).Render("type to search comments...")
		}
		head.WriteString(truncateScreen(label+": "+query, width) + "\n")
	}
	head.WriteString("\n")

	headerText := head.String()
	headerLines := strings.Count(headerText, "\n")

	if len(c.order) == 0 {
		if c.loading == "" && c.err == "" {
			headerText += "No comments yet. Press esc to go back.\n"
		}
		return headerText
	}

	contentHeight := maxScreen(1, height-headerLines)
	lines, starts := c.renderComments(width)
	if len(lines) == 0 {
		return headerText
	}
	sel := clampIndex(c.selected, len(starts))
	cursor := starts[sel]
	maxTop := maxScreen(0, len(lines)-contentHeight)
	top := cursor - contentHeight/2
	if top < 0 {
		top = 0
	} else if top > maxTop {
		top = maxTop
	}
	end := minScreen(len(lines), top+contentHeight)

	var b strings.Builder
	b.WriteString(headerText)
	for i := top; i < end; i++ {
		line := lines[i]
		if i == cursor {
			line = articleLineHighlight(width).Render(padLine(ansi.Strip(line), width))
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
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

func (c Comments) handleKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	if c.searching {
		return c.handleSearchKey(msg)
	}
	switch msg.String() {
	case "esc":
		dest := c.returnTo
		if dest == "" {
			dest = hn.FeedTop.ScreenID()
		}
		return c, func() tea.Msg { return NavigateMsg{ScreenID: dest} }
	case "r":
		c.loading = "Loading comments..."
		c.err = ""
		return c, c.loadTree()
	case "o":
		c.status = commentsOpenURL(c.opener, c.hnURL())
		return c, nil
	case "y":
		c.status = commentsCopyURL(c.copier, c.hnURL())
		return c, nil
	case "/":
		c.searching = true
		return c, nil
	case "ctrl+u":
		c.searchQuery = ""
		return c, nil
	}
	if len(c.order) == 0 {
		return c, nil
	}
	c.selected = clampIndex(c.selected, len(c.order))
	switch msg.String() {
	case "up", "k":
		if c.selected > 0 {
			c.selected--
		}
	case "down", "j":
		if c.selected < len(c.order)-1 {
			c.selected++
		}
	case "pgup":
		c.selected -= 10
		if c.selected < 0 {
			c.selected = 0
		}
	case "pgdown":
		c.selected += 10
		if c.selected >= len(c.order) {
			c.selected = len(c.order) - 1
		}
	case "g":
		c.selected = 0
	case "G":
		c.selected = len(c.order) - 1
	case "space", "enter":
		id := c.order[c.selected].id
		c.collapsed[id] = !c.collapsed[id]
		c.order = c.buildOrder()
		for i, line := range c.order {
			if line.id == id {
				c.selected = i
				break
			}
		}
	case "left", "p":
		c.selected = c.previousTopLevel(c.selected)
	case "right", "n":
		c.selected = c.nextTopLevel(c.selected)
	case "P":
		cur := c.order[c.selected].id
		if parent, ok := c.parentOf[cur]; ok && parent != c.story.ID {
			for i, line := range c.order {
				if line.id == parent {
					c.selected = i
					break
				}
			}
		}
	case "a":
		c.allCollapsed = !c.allCollapsed
		c.collapsed = make(map[int]bool)
		if c.allCollapsed {
			for _, line := range c.order {
				if countDescendants(c.tree, line.id) > 0 {
					c.collapsed[line.id] = true
				}
			}
		}
		selectedID := c.order[c.selected].id
		c.order = c.buildOrder()
		c.selected = c.indexOfVisibleComment(selectedID)
	}
	return c, nil
}

func (c Comments) handleSearchKey(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		c.searching = false
		return c, nil
	case "ctrl+u":
		c.searchQuery = ""
		return c, nil
	case "backspace", "ctrl+h":
		if len(c.searchQuery) > 0 {
			c.searchQuery = c.searchQuery[:len(c.searchQuery)-1]
		}
		return c, nil
	case "space":
		c.searchQuery += " "
		return c, nil
	case "right", "n", "enter":
		c.selected = c.nextSearchMatch(c.selected)
		return c, nil
	case "left", "p":
		c.selected = c.previousSearchMatch(c.selected)
		return c, nil
	}
	if len(msg.String()) == 1 {
		c.searchQuery += msg.String()
		c.selected = c.nextSearchMatch(c.selected - 1)
	}
	return c, nil
}

func (c Comments) hnURL() string {
	if c.story.ID == 0 {
		return ""
	}
	return fmt.Sprintf("https://news.ycombinator.com/item?id=%d", c.story.ID)
}

func (c Comments) loadTree() tea.Cmd {
	client := c.client
	storyID := c.story.ID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tree, err := client.CommentTree(ctx, storyID, commentMaxDepth, commentMaxCount)
		return commentsTreeLoadedMsg{screenID: "comments", storyID: storyID, tree: tree, err: err}
	}
}

func (c Comments) buildOrder() []commentLine {
	root, ok := c.tree[c.story.ID]
	if !ok {
		return nil
	}
	var order []commentLine
	var walk func(id, depth int)
	walk = func(id, depth int) {
		if _, ok := c.tree[id]; !ok {
			return
		}
		order = append(order, commentLine{id: id, depth: depth})
		if c.collapsed[id] {
			return
		}
		for _, kid := range c.tree[id].Kids {
			walk(kid, depth+1)
		}
	}
	for _, kid := range root.Kids {
		walk(kid, 0)
	}
	return order
}

func (c Comments) renderComments(width int) ([]string, []int) {
	lines := make([]string, 0, len(c.order)*4)
	starts := make([]int, 0, len(c.order))
	muted := lipgloss.NewStyle().Faint(true)
	title := lipgloss.NewStyle().Bold(true)
	for _, line := range c.order {
		item := c.tree[line.id]
		indent := strings.Repeat("│ ", line.depth)
		starts = append(starts, len(lines))

		var headerParts []string
		if item.Deleted || item.Dead {
			headerParts = append(headerParts, title.Render("[deleted]"))
		} else {
			author := item.By
			if author == "" {
				author = "anonymous"
			}
			headerParts = append(headerParts, title.Render("@"+author))
			if ts := relativeAge(item.Time); ts != "" {
				headerParts = append(headerParts, muted.Render(ts))
			}
		}
		replyCount := countDescendants(c.tree, line.id)
		if c.collapsed[line.id] && replyCount > 0 {
			headerParts = append(headerParts, muted.Render(fmt.Sprintf("[+%d hidden]", replyCount)))
		}
		lines = append(lines, truncateScreen(muted.Render(indent)+strings.Join(headerParts, " · "), width))

		if !c.collapsed[line.id] {
			body := commentBodyMarkdown(item)
			bodyWidth := maxScreen(20, width-lipgloss.Width(indent))
			rendered := renderMarkdown(body, bodyWidth)
			for _, bline := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
				lines = append(lines, truncateScreen(muted.Render(indent)+bline, width))
			}
		}
		lines = append(lines, "")
	}
	return lines, starts
}

func commentsHeaderBlock(story hn.Item, width int) []string {
	var out []string
	titleStyle := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Faint(true)
	if strings.TrimSpace(story.Title) != "" {
		out = append(out, truncateScreen(titleStyle.Render(story.Title), width))
	}
	var metaParts []string
	if story.By != "" {
		metaParts = append(metaParts, "by "+story.By)
	}
	if story.Score > 0 {
		metaParts = append(metaParts, fmt.Sprintf("%d points", story.Score))
	}
	if story.Descendants > 0 {
		metaParts = append(metaParts, fmt.Sprintf("%d comments", story.Descendants))
	}
	if ts := relativeAge(story.Time); ts != "" {
		metaParts = append(metaParts, ts)
	}
	if story.URL != "" {
		if domain := storyDomain(story.URL); domain != "" {
			metaParts = append(metaParts, domain)
		}
	}
	if len(metaParts) > 0 {
		out = append(out, truncateScreen(muted.Render(strings.Join(metaParts, " · ")), width))
	}
	if strings.TrimSpace(story.Text) != "" {
		rendered := renderMarkdown(commentHTMLToMarkdown(story.Text), width)
		for _, line := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
			out = append(out, truncateScreen(line, width))
		}
	}
	out = append(out, truncateScreen(muted.Render("esc back · j/k move · left/right or p/n prev/next · / search · P parent · space collapse · o open"), width))
	return out
}

func (c Comments) nextTopLevel(from int) int {
	for i := from + 1; i < len(c.order); i++ {
		if c.order[i].depth == 0 {
			return i
		}
	}
	return from
}

func (c Comments) previousTopLevel(from int) int {
	for i := from - 1; i >= 0; i-- {
		if c.order[i].depth == 0 {
			return i
		}
	}
	return from
}

func (c Comments) nextSearchMatch(from int) int {
	if strings.TrimSpace(c.searchQuery) == "" || len(c.order) == 0 {
		return clampIndex(from, len(c.order))
	}
	for step := 1; step <= len(c.order); step++ {
		idx := (from + step + len(c.order)) % len(c.order)
		if c.commentMatches(c.order[idx].id) {
			return idx
		}
	}
	return clampIndex(from, len(c.order))
}

func (c Comments) previousSearchMatch(from int) int {
	if strings.TrimSpace(c.searchQuery) == "" || len(c.order) == 0 {
		return clampIndex(from, len(c.order))
	}
	for step := 1; step <= len(c.order); step++ {
		idx := (from - step + len(c.order)) % len(c.order)
		if c.commentMatches(c.order[idx].id) {
			return idx
		}
	}
	return clampIndex(from, len(c.order))
}

func (c Comments) commentMatches(id int) bool {
	query := strings.ToLower(strings.TrimSpace(c.searchQuery))
	if query == "" {
		return false
	}
	item := c.tree[id]
	fields := []string{item.By, commentHTMLToMarkdown(item.Text)}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func (c Comments) indexOfVisibleComment(id int) int {
	for i, line := range c.order {
		if line.id == id {
			return i
		}
	}
	return clampIndex(c.selected, len(c.order))
}

func commentBodyMarkdown(item hn.Item) string {
	if item.Deleted || item.Dead {
		return "*[deleted]*"
	}
	return commentHTMLToMarkdown(item.Text)
}

var commentLinkRE = regexp.MustCompile(`(?is)<a\s+[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
var commentTagRE = regexp.MustCompile(`(?is)</?[a-z][^>]*>`)

func commentHTMLToMarkdown(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	t := text
	t = strings.ReplaceAll(t, "<p>", "\n\n")
	t = strings.ReplaceAll(t, "</p>", "")
	t = strings.ReplaceAll(t, "<i>", "*")
	t = strings.ReplaceAll(t, "</i>", "*")
	t = strings.ReplaceAll(t, "<pre><code>", "\n\n```\n")
	t = strings.ReplaceAll(t, "</code></pre>", "\n```\n\n")
	t = strings.ReplaceAll(t, "<code>", "`")
	t = strings.ReplaceAll(t, "</code>", "`")
	t = commentLinkRE.ReplaceAllString(t, "[$2]($1)")
	t = commentTagRE.ReplaceAllString(t, "")
	return html.UnescapeString(t)
}

func countDescendants(tree map[int]hn.Item, id int) int {
	item, ok := tree[id]
	if !ok {
		return 0
	}
	count := 0
	for _, kid := range item.Kids {
		if _, present := tree[kid]; present {
			count += 1 + countDescendants(tree, kid)
		}
	}
	return count
}

func buildParentMap(tree map[int]hn.Item) map[int]int {
	parent := make(map[int]int, len(tree))
	for id, item := range tree {
		for _, kid := range item.Kids {
			parent[kid] = id
		}
	}
	return parent
}

func relativeAge(unix int64) string {
	if unix <= 0 {
		return ""
	}
	d := time.Since(time.Unix(unix, 0))
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/24/365))
	}
}

func commentsOpenURL(opener func(string) error, url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to open"
	}
	if opener == nil {
		return "Could not open browser: no opener configured"
	}
	if err := opener(url); err != nil {
		return "Could not open browser: " + err.Error()
	}
	return "Opening in browser..."
}

func commentsCopyURL(copier func(string) error, url string) string {
	if strings.TrimSpace(url) == "" {
		return "No URL to copy"
	}
	if copier == nil {
		return "Could not copy URL: no clipboard configured"
	}
	if err := copier(url); err != nil {
		return "Could not copy URL: " + err.Error()
	}
	return "Copied URL to clipboard"
}
