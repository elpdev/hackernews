package hackernews

import (
	"sort"

	"github.com/elpdev/hackernews/pkg/config"
	"github.com/elpdev/hackernews/pkg/history"
	"github.com/elpdev/hackernews/pkg/hn"
	"github.com/elpdev/hackernews/pkg/saved"
	"github.com/elpdev/hackernews/pkg/screens"
	"github.com/elpdev/hackernews/pkg/theme"
)

const DefaultScreenID = "top"

type Options struct {
	SavedStore   saved.Store
	HistoryStore history.Store
	Settings     config.Settings
	Themes       []theme.Theme
}

type Module struct {
	screens  []ScreenSpec
	commands []CommandSpec
	settings config.Settings
}

type ScreenSpec struct {
	ID      string
	Screen  screens.Screen
	Sidebar bool
	Order   int
}

type CommandSpec struct {
	ID          string
	ParentID    string
	Title       string
	Description string
	Keywords    []string
	Order       int
	ScreenID    string
}

func New(options Options) Module {
	settings := normalizeSettings(options.Settings)
	themes := options.Themes
	if len(themes) == 0 {
		themes = theme.BuiltIns()
	}

	specs := []ScreenSpec{
		{ID: "top", Screen: screens.NewStories(options.SavedStore, hn.FeedTop, options.HistoryStore, settings.HideRead, settings.SortMode), Sidebar: true, Order: 10},
		{ID: "new", Screen: screens.NewStories(options.SavedStore, hn.FeedNew, options.HistoryStore, settings.HideRead, settings.SortMode), Sidebar: true, Order: 20},
		{ID: "best", Screen: screens.NewStories(options.SavedStore, hn.FeedBest, options.HistoryStore, settings.HideRead, settings.SortMode), Sidebar: true, Order: 30},
		{ID: "ask", Screen: screens.NewStories(options.SavedStore, hn.FeedAsk, options.HistoryStore, settings.HideRead, settings.SortMode), Sidebar: true, Order: 40},
		{ID: "show", Screen: screens.NewStories(options.SavedStore, hn.FeedShow, options.HistoryStore, settings.HideRead, settings.SortMode), Sidebar: true, Order: 50},
		{ID: "jobs", Screen: screens.NewStories(options.SavedStore, hn.FeedJob, options.HistoryStore, settings.HideRead, settings.SortMode), Sidebar: true, Order: 60},
		{ID: "saved", Screen: screens.NewSaved(options.SavedStore), Sidebar: true, Order: 70},
		{ID: "settings", Screen: screens.NewSettings(settings, themes), Sidebar: true, Order: 80},
		{ID: "comments", Screen: screens.NewComments(hn.NewClient(nil)), Order: 1000},
		{ID: "search", Screen: screens.NewSearch(), Order: 1010},
		{ID: "doctor", Screen: screens.NewDoctor(settings), Order: 1020},
	}

	return Module{screens: specs, commands: commandSpecs(), settings: settings}
}

func normalizeSettings(settings config.Settings) config.Settings {
	defaults := config.Defaults()
	if settings.ThemeName == "" {
		settings.ThemeName = defaults.ThemeName
	}
	if settings.DefaultFeed == "" {
		settings.DefaultFeed = defaults.DefaultFeed
	}
	if settings.SyncBackend == "" {
		settings.SyncBackend = defaults.SyncBackend
	}
	if settings.SyncBranch == "" {
		settings.SyncBranch = defaults.SyncBranch
	}
	if settings.SyncDir == "" {
		settings.SyncDir = defaults.SyncDir
	}
	return settings
}

func (m Module) DefaultScreen() string {
	if _, ok := m.Screens()[m.settings.DefaultFeed]; ok {
		return m.settings.DefaultFeed
	}
	return DefaultScreenID
}

func (m Module) ScreenSpecs() []ScreenSpec {
	return append([]ScreenSpec(nil), m.screens...)
}

func (m Module) Screens() map[string]screens.Screen {
	out := make(map[string]screens.Screen, len(m.screens))
	for _, spec := range m.screens {
		out[spec.ID] = spec.Screen
	}
	return out
}

func (m Module) ScreenOrder() []string {
	specs := append([]ScreenSpec(nil), m.screens...)
	sort.Slice(specs, func(i, j int) bool {
		if specs[i].Order != specs[j].Order {
			return specs[i].Order < specs[j].Order
		}
		return specs[i].ID < specs[j].ID
	})
	out := make([]string, 0, len(specs))
	for _, spec := range specs {
		if spec.Sidebar {
			out = append(out, spec.ID)
		}
	}
	return out
}

func (m Module) Commands() []CommandSpec {
	return append([]CommandSpec(nil), m.commands...)
}

func RefreshSearchScreen(all map[string]screens.Screen, search screens.Search) screens.Search {
	var items []screens.StorySnapshot
	for _, screen := range all {
		if stories, ok := screen.(screens.Top); ok {
			items = append(items, stories.Snapshot()...)
		}
	}
	return search.WithItems(items)
}

func commandSpecs() []CommandSpec {
	return []CommandSpec{
		{ID: "browse", Title: "Browse", Description: "Feeds and story lists", Keywords: []string{"feed", "stories", "hacker news"}, Order: 10},
		{ID: "library", Title: "Library", Description: "Saved articles and search", Keywords: []string{"saved", "search", "articles"}, Order: 20},
		{ID: "settings", Title: "Settings", Description: "App preferences", Keywords: []string{"config", "preferences", "options"}, Order: 35},
		{ID: "system", Title: "System", Description: "App-level actions", Keywords: []string{"diagnostics", "health"}, Order: 60},
		{ID: "go-top", ParentID: "browse", Title: "Top Stories", Description: "Open Hacker News top stories", Keywords: []string{"top", "hacker news", "stories", "news"}, Order: 10, ScreenID: "top"},
		{ID: "go-new", ParentID: "browse", Title: "New", Description: "Open newest Hacker News stories", Keywords: []string{"new", "newest", "recent"}, Order: 20, ScreenID: "new"},
		{ID: "go-best", ParentID: "browse", Title: "Best", Description: "Open best Hacker News stories", Keywords: []string{"best", "popular"}, Order: 30, ScreenID: "best"},
		{ID: "go-ask", ParentID: "browse", Title: "Ask HN", Description: "Open Ask HN stories", Keywords: []string{"ask", "ask hn", "questions"}, Order: 40, ScreenID: "ask"},
		{ID: "go-show", ParentID: "browse", Title: "Show HN", Description: "Open Show HN stories", Keywords: []string{"show", "show hn", "projects"}, Order: 50, ScreenID: "show"},
		{ID: "go-jobs", ParentID: "browse", Title: "Jobs", Description: "Open HN job postings", Keywords: []string{"jobs", "hiring", "careers"}, Order: 60, ScreenID: "jobs"},
		{ID: "go-saved", ParentID: "library", Title: "Saved", Description: "Open saved articles", Keywords: []string{"saved", "articles", "bookmarks", "offline"}, Order: 10, ScreenID: "saved"},
		{ID: "go-search", ParentID: "library", Title: "Search Loaded Stories", Description: "Search stories already loaded in feeds", Keywords: []string{"search", "find", "loaded", "stories"}, Order: 20, ScreenID: "search"},
		{ID: "go-settings", ParentID: "settings", Title: "Open Settings", Description: "Open app preferences", Keywords: []string{"settings", "config", "preferences"}, Order: 10, ScreenID: "settings"},
		{ID: "doctor", ParentID: "system", Title: "Doctor", Description: "Check local setup and dependencies", Keywords: []string{"diagnostics", "health", "python", "trafilatura", "git", "clipboard"}, Order: 5, ScreenID: "doctor"},
	}
}
