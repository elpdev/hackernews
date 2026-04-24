package screens

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
)

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
