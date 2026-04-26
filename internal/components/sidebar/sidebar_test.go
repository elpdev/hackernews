package sidebar

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/elpdev/hackernews/pkg/theme"
)

func TestInactiveTopStoriesDoesNotWrap(t *testing.T) {
	view := View(Model{
		Items: []Item{
			{ID: "top", Title: "Top Stories"},
			{ID: "saved", Title: "Saved"},
		},
		ActiveID: "saved",
	}, 18, 10, theme.Phosphor())

	lines := strings.Split(ansi.Strip(view), "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "Top Stories") {
			found = true
		}
		if strings.TrimSpace(line) == "Top" {
			t.Fatal("expected Top Stories to stay on one sidebar row")
		}
		if strings.TrimSpace(line) == "Stories" {
			t.Fatal("expected Top Stories to stay on one sidebar row")
		}
	}
	if !found {
		t.Fatal("expected rendered sidebar to contain Top Stories")
	}
}

func TestSidebarItemTextColumnStaysStable(t *testing.T) {
	theme := theme.Phosphor()
	items := []Item{
		{ID: "top", Title: "Top Stories"},
		{ID: "saved", Title: "Saved"},
	}

	topActive := sidebarTitleColumn(View(Model{Items: items, ActiveID: "top"}, 18, 10, theme), "Top Stories")
	topInactive := sidebarTitleColumn(View(Model{Items: items, ActiveID: "saved"}, 18, 10, theme), "Top Stories")
	if topActive != topInactive {
		t.Fatalf("expected Top Stories column to stay stable, active=%d inactive=%d", topActive, topInactive)
	}

	savedInactive := sidebarTitleColumn(View(Model{Items: items, ActiveID: "top"}, 18, 10, theme), "Saved")
	savedActive := sidebarTitleColumn(View(Model{Items: items, ActiveID: "saved"}, 18, 10, theme), "Saved")
	if savedActive != savedInactive {
		t.Fatalf("expected Saved column to stay stable, active=%d inactive=%d", savedActive, savedInactive)
	}
}

func TestSidebarItemTextStaysLeftAligned(t *testing.T) {
	view := View(Model{
		Items: []Item{
			{ID: "top", Title: "Top Stories"},
			{ID: "saved", Title: "Saved"},
		},
		ActiveID: "saved",
	}, 18, 10, theme.Phosphor())

	column := sidebarTitleColumn(view, "Top Stories")
	if column != 1 {
		t.Fatalf("expected Top Stories to be left aligned at column 1, got %d", column)
	}
}

func sidebarTitleColumn(view, title string) int {
	for _, line := range strings.Split(ansi.Strip(view), "\n") {
		if column := strings.Index(line, title); column >= 0 {
			return column
		}
	}
	return -1
}
