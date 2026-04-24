package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const DefaultBaseURL = "https://hacker-news.firebaseio.com/v0"

type Feed string

const (
	FeedTop  Feed = "topstories"
	FeedNew  Feed = "newstories"
	FeedBest Feed = "beststories"
	FeedAsk  Feed = "askstories"
	FeedShow Feed = "showstories"
	FeedJob  Feed = "jobstories"
)

func (f Feed) Title() string {
	switch f {
	case FeedNew:
		return "New"
	case FeedBest:
		return "Best"
	case FeedAsk:
		return "Ask HN"
	case FeedShow:
		return "Show HN"
	case FeedJob:
		return "Jobs"
	default:
		return "Top Stories"
	}
}

func (f Feed) ScreenID() string {
	switch f {
	case FeedNew:
		return "new"
	case FeedBest:
		return "best"
	case FeedAsk:
		return "ask"
	case FeedShow:
		return "show"
	case FeedJob:
		return "jobs"
	default:
		return "top"
	}
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(httpClient *http.Client) Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return Client{baseURL: DefaultBaseURL, http: httpClient}
}

func NewClientWithBaseURL(baseURL string, httpClient *http.Client) Client {
	c := NewClient(httpClient)
	c.baseURL = strings.TrimRight(baseURL, "/")
	return c
}

func (c Client) StoryIDs(ctx context.Context, feed Feed) ([]int, error) {
	if feed == "" {
		feed = FeedTop
	}
	var ids []int
	if err := c.getJSON(ctx, string(feed)+".json", &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (c Client) TopStoryIDs(ctx context.Context) ([]int, error) {
	return c.StoryIDs(ctx, FeedTop)
}

func (c Client) Item(ctx context.Context, id int) (Item, error) {
	var item Item
	if err := c.getJSON(ctx, fmt.Sprintf("item/%d.json", id), &item); err != nil {
		return Item{}, err
	}
	return item, nil
}

func (c Client) TopStories(ctx context.Context, limit int) ([]Item, error) {
	ids, err := c.TopStoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(ids) {
		limit = len(ids)
	}
	ids = ids[:limit]

	return c.Stories(ctx, ids)
}

func (c Client) Stories(ctx context.Context, ids []int) ([]Item, error) {
	stories := make([]Item, len(ids))
	errs := make(chan error, len(ids))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for idx, id := range ids {
		idx, id := idx, id
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			}
			item, err := c.Item(ctx, id)
			if err != nil {
				errs <- err
				return
			}
			stories[idx] = item
		}()
	}
	wg.Wait()
	close(errs)
	if err := <-errs; err != nil {
		return nil, err
	}

	filtered := stories[:0]
	for _, story := range stories {
		if story.Readable() {
			filtered = append(filtered, story)
		}
	}
	return filtered, nil
}

// CommentTree fetches the root item and its descendant comments level-by-level
// up to maxDepth. Stops early once totalCap items have been collected. Returns
// partial results on error. Keyed by item ID; tree shape is reconstructed via
// each returned Item's Kids slice.
func (c Client) CommentTree(ctx context.Context, rootID, maxDepth, totalCap int) (map[int]Item, error) {
	if maxDepth <= 0 {
		maxDepth = 8
	}
	if totalCap <= 0 {
		totalCap = 500
	}
	result := make(map[int]Item)
	root, err := c.Item(ctx, rootID)
	if err != nil {
		return nil, err
	}
	result[rootID] = root

	sem := make(chan struct{}, 8)
	current := []int{rootID}
	for depth := 0; depth < maxDepth && len(current) > 0; depth++ {
		var nextIDs []int
		for _, id := range current {
			for _, kid := range result[id].Kids {
				if _, seen := result[kid]; !seen {
					nextIDs = append(nextIDs, kid)
				}
			}
		}
		if len(nextIDs) == 0 {
			break
		}
		remaining := totalCap - len(result)
		if remaining <= 0 {
			break
		}
		if len(nextIDs) > remaining {
			nextIDs = nextIDs[:remaining]
		}

		fetched := make([]Item, len(nextIDs))
		errs := make(chan error, len(nextIDs))
		var wg sync.WaitGroup
		for i, id := range nextIDs {
			i, id := i, id
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					errs <- ctx.Err()
					return
				}
				item, err := c.Item(ctx, id)
				if err != nil {
					errs <- err
					return
				}
				fetched[i] = item
			}()
		}
		wg.Wait()
		close(errs)
		if err := <-errs; err != nil {
			return result, err
		}
		for _, item := range fetched {
			result[item.ID] = item
		}
		current = nextIDs
	}
	return result, nil
}

func (c Client) getJSON(ctx context.Context, path string, dest any) error {
	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("hacker news api returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return err
	}
	return nil
}
