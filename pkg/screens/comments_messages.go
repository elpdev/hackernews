package screens

import "github.com/elpdev/hackernews/pkg/hn"

type commentsTreeLoadedMsg struct {
	screenID string
	storyID  int
	tree     map[int]hn.Item
	err      error
}

func (m commentsTreeLoadedMsg) TargetScreenID() string { return m.screenID }
