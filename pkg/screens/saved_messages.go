package screens

import (
	"github.com/elpdev/hackernews/pkg/articles"
	"github.com/elpdev/hackernews/pkg/saved"
)

type savedArticlesLoadedMsg struct {
	screenID string
	items    []saved.Article
	err      error
}

func (m savedArticlesLoadedMsg) TargetScreenID() string { return m.screenID }

type savedArticleDeletedMsg struct {
	screenID string
	id       int
	err      error
}

func (m savedArticleDeletedMsg) TargetScreenID() string { return m.screenID }

type savedArticleTagsUpdatedMsg struct {
	screenID string
	id       int
	tags     []string
	err      error
}

func (m savedArticleTagsUpdatedMsg) TargetScreenID() string { return m.screenID }

type savedArticleExtractedMsg struct {
	screenID string
	id       int
	article  articles.Article
	err      error
}

func (m savedArticleExtractedMsg) TargetScreenID() string { return m.screenID }
