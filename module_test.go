package hackernews

import (
	"reflect"
	"testing"

	"github.com/elpdev/hackernews/pkg/config"
	"github.com/elpdev/hackernews/pkg/screens"
)

func TestModuleScreens(t *testing.T) {
	module := New(Options{Settings: config.Defaults()})
	screensByID := module.Screens()

	for _, id := range []string{"top", "new", "best", "ask", "show", "jobs", "saved", "settings", "comments", "search", "doctor"} {
		if screensByID[id] == nil {
			t.Fatalf("expected screen %q to be registered", id)
		}
	}
}

func TestModuleScreenOrderIncludesSidebarScreensOnly(t *testing.T) {
	module := New(Options{Settings: config.Defaults()})
	want := []string{"top", "new", "best", "ask", "show", "jobs", "saved", "settings"}

	if got := module.ScreenOrder(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected screen order: got %#v want %#v", got, want)
	}
}

func TestModuleDefaultScreen(t *testing.T) {
	settings := config.Defaults()
	settings.DefaultFeed = "best"
	module := New(Options{Settings: settings})

	if got := module.DefaultScreen(); got != "best" {
		t.Fatalf("expected default screen best, got %q", got)
	}

	settings.DefaultFeed = "missing"
	module = New(Options{Settings: settings})
	if got := module.DefaultScreen(); got != DefaultScreenID {
		t.Fatalf("expected fallback default screen %q, got %q", DefaultScreenID, got)
	}
}

func TestModuleCommandsExposeRouteTargets(t *testing.T) {
	module := New(Options{Settings: config.Defaults()})
	commands := module.Commands()
	byID := make(map[string]CommandSpec, len(commands))
	for _, command := range commands {
		byID[command.ID] = command
	}

	for id, screenID := range map[string]string{"go-top": "top", "go-new": "new", "go-best": "best", "go-ask": "ask", "go-show": "show", "go-jobs": "jobs", "go-saved": "saved", "go-search": "search", "go-settings": "settings", "doctor": "doctor"} {
		command, ok := byID[id]
		if !ok {
			t.Fatalf("expected command %q", id)
		}
		if command.ScreenID != screenID {
			t.Fatalf("expected command %q to target %q, got %q", id, screenID, command.ScreenID)
		}
	}
}

func TestRefreshSearchScreenUsesLoadedStories(t *testing.T) {
	search := screens.NewSearch()
	updated := RefreshSearchScreen(map[string]screens.Screen{}, search)

	if updated.Title() != "Search" {
		t.Fatalf("expected search screen to remain usable, got title %q", updated.Title())
	}
}
