package screens

import (
	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/hackernews/internal/hn"
)

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
		c.toggleSelectedComment()
	case "left", "p":
		c.selected = c.previousTopLevel(c.selected)
	case "right", "n":
		c.selected = c.nextTopLevel(c.selected)
	case "P":
		c.selectParent()
	case "a":
		c.toggleAllCollapsed()
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

func (c *Comments) toggleSelectedComment() {
	id := c.order[c.selected].id
	c.collapsed[id] = !c.collapsed[id]
	c.order = c.buildOrder()
	for i, line := range c.order {
		if line.id == id {
			c.selected = i
			break
		}
	}
}

func (c *Comments) selectParent() {
	cur := c.order[c.selected].id
	if parent, ok := c.parentOf[cur]; ok && parent != c.story.ID {
		for i, line := range c.order {
			if line.id == parent {
				c.selected = i
				break
			}
		}
	}
}

func (c *Comments) toggleAllCollapsed() {
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
