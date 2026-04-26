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

func TestStoryIDsDispatchesByFeed(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[1,2]`))
	}))
	defer server.Close()

	client := NewClientWithBaseURL(server.URL, server.Client())
	if _, err := client.StoryIDs(context.Background(), FeedAsk); err != nil {
		t.Fatalf("StoryIDs returned error: %v", err)
	}
	if gotPath != "/askstories.json" {
		t.Fatalf("expected /askstories.json, got %q", gotPath)
	}
}

func TestCommentTreeFetchesNestedReplies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/item/1.json":
			_, _ = w.Write([]byte(`{"id":1,"type":"story","kids":[2,3]}`))
		case "/item/2.json":
			_, _ = w.Write([]byte(`{"id":2,"type":"comment","by":"alice","kids":[4]}`))
		case "/item/3.json":
			_, _ = w.Write([]byte(`{"id":3,"type":"comment","by":"bob"}`))
		case "/item/4.json":
			_, _ = w.Write([]byte(`{"id":4,"type":"comment","by":"carol"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClientWithBaseURL(server.URL, server.Client())
	tree, err := client.CommentTree(context.Background(), 1, 8, 100)
	if err != nil {
		t.Fatalf("CommentTree returned error: %v", err)
	}
	for _, id := range []int{1, 2, 3, 4} {
		if _, ok := tree[id]; !ok {
			t.Fatalf("expected id %d in tree, got %v", id, tree)
		}
	}
	if tree[2].By != "alice" {
		t.Fatalf("expected alice as author for id 2, got %q", tree[2].By)
	}
}

func TestCommentTreeRespectsTotalCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/item/1.json":
			_, _ = w.Write([]byte(`{"id":1,"type":"story","kids":[2,3,4,5]}`))
		default:
			id := r.URL.Path[len("/item/") : len(r.URL.Path)-len(".json")]
			_, _ = w.Write([]byte(`{"id":` + id + `,"type":"comment"}`))
		}
	}))
	defer server.Close()

	client := NewClientWithBaseURL(server.URL, server.Client())
	tree, err := client.CommentTree(context.Background(), 1, 8, 3)
	if err != nil {
		t.Fatalf("CommentTree returned error: %v", err)
	}
	if len(tree) != 3 {
		t.Fatalf("expected tree size capped at 3, got %d: %v", len(tree), tree)
	}
}
