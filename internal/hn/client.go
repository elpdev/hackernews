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

func (c Client) TopStoryIDs(ctx context.Context) ([]int, error) {
	var ids []int
	if err := c.getJSON(ctx, "topstories.json", &ids); err != nil {
		return nil, err
	}
	return ids, nil
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
