package screens

import (
	"strings"

	"github.com/elpdev/hackernews/pkg/hn"
)

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
