package screens

import (
	"github.com/elpdev/hackernews/internal/articles"
	"github.com/elpdev/hackernews/internal/hn"
)

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

type articleImageLoadedMsg struct {
	screenID string
	id       int
	url      string
	bytes    []byte
	err      error
}

func (m articleImageLoadedMsg) TargetScreenID() string { return m.screenID }
