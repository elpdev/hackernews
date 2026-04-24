package hn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTopStoriesFetchesItemsInOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/topstories.json":
			_, _ = w.Write([]byte(`[3,2,1]`))
		case "/item/3.json":
			_, _ = w.Write([]byte(`{"id":3,"type":"story","title":"third","score":30}`))
		case "/item/2.json":
			_, _ = w.Write([]byte(`{"id":2,"type":"comment","title":"skip"}`))
		case "/item/1.json":
			_, _ = w.Write([]byte(`{"id":1,"type":"story","title":"first","score":10}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClientWithBaseURL(server.URL, server.Client())
	stories, err := client.TopStories(context.Background(), 3)
	if err != nil {
		t.Fatalf("TopStories returned error: %v", err)
	}
	if len(stories) != 2 {
		t.Fatalf("expected 2 readable stories, got %d", len(stories))
	}
	if stories[0].ID != 3 || stories[1].ID != 1 {
		t.Fatalf("stories not kept in ranking order: %#v", stories)
	}
}

func TestTopStoryIDsReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClientWithBaseURL(server.URL, server.Client())
	if _, err := client.TopStoryIDs(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
