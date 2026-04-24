package saved

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/elpdev/hackernews/internal/articles"
	"github.com/elpdev/hackernews/internal/hn"
)

const fileMode = 0o600

type Article struct {
	ID      int              `json:"id"`
	SavedAt time.Time        `json:"saved_at"`
	Story   hn.Item          `json:"story"`
	Article articles.Article `json:"article"`
}

type Store interface {
	List(context.Context) ([]Article, error)
	Get(context.Context, int) (Article, bool, error)
	Save(context.Context, hn.Item, articles.Article) error
	Delete(context.Context, int) error
	IsSaved(context.Context, int) (bool, error)
}

type JSONStore struct {
	path string
	now  func() time.Time
}

func NewJSONStore(path string) JSONStore {
	return JSONStore{path: path, now: time.Now}
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hackernews", "saved.json"), nil
}

func (s JSONStore) List(ctx context.Context) ([]Article, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items, err := s.read()
	if err != nil {
		return nil, err
	}
	sortSaved(items)
	return items, nil
}

func (s JSONStore) Get(ctx context.Context, id int) (Article, bool, error) {
	items, err := s.List(ctx)
	if err != nil {
		return Article{}, false, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, true, nil
		}
	}
	return Article{}, false, nil
}

func (s JSONStore) Save(ctx context.Context, story hn.Item, article articles.Article) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	items, err := s.read()
	if err != nil {
		return err
	}
	now := s.now().UTC()
	for i := range items {
		if items[i].ID == story.ID {
			items[i].Story = story
			items[i].Article = article
			sortSaved(items)
			return s.write(items)
		}
	}
	items = append(items, Article{ID: story.ID, SavedAt: now, Story: story, Article: article})
	sortSaved(items)
	return s.write(items)
}

func (s JSONStore) Delete(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	items, err := s.read()
	if err != nil {
		return err
	}
	filtered := items[:0]
	for _, item := range items {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	return s.write(filtered)
}

func (s JSONStore) IsSaved(ctx context.Context, id int) (bool, error) {
	_, ok, err := s.Get(ctx, id)
	return ok, err
}

func (s JSONStore) read() ([]Article, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var items []Article
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s JSONStore) write(items []Article) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".saved-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(fileMode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}

func sortSaved(items []Article) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].SavedAt.After(items[j].SavedAt)
	})
}
