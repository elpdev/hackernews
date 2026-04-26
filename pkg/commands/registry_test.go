package commands

import "testing"

func TestRegistryRegisterAndFind(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Command{ID: "go-home", Title: "Go to Home"})

	command, ok := registry.Find("go-home")
	if !ok {
		t.Fatal("expected command to be found")
	}
	if command.Title != "Go to Home" {
		t.Fatalf("unexpected title: %q", command.Title)
	}
}

func TestRegistryFilter(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Command{ID: "go-home", Title: "Go to Home", Description: "Open home", Keywords: []string{"start"}})
	registry.Register(Command{ID: "toggle-theme", Title: "Toggle Theme", Description: "Switch colors", Keywords: []string{"dark"}})

	matches := registry.Filter("dark")
	if len(matches) != 1 {
		t.Fatalf("expected one match, got %d", len(matches))
	}
	if matches[0].ID != "toggle-theme" {
		t.Fatalf("unexpected match: %q", matches[0].ID)
	}
}

func TestRegistryChildren(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Command{ID: "browse", Title: "Browse", Order: 10})
	registry.Register(Command{ID: "settings", Title: "Settings", Order: 20})
	registry.Register(Command{ID: "go-top", ParentID: "browse", Title: "Top Stories", Order: 20})
	registry.Register(Command{ID: "go-new", ParentID: "browse", Title: "New", Order: 10})

	children := registry.Children("browse")
	if len(children) != 2 {
		t.Fatalf("expected two children, got %d", len(children))
	}
	if children[0].ID != "go-new" || children[1].ID != "go-top" {
		t.Fatalf("unexpected children order: %#v", children)
	}
	if !registry.HasChildren("browse") {
		t.Fatal("expected browse to have children")
	}
	if registry.HasChildren("settings") {
		t.Fatal("did not expect settings to have children")
	}
}
