package history

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const fileMode = 0o600

type Entry struct {
	ID        int       `json:"id"`
	FirstRead time.Time `json:"first_read"`
	LastRead  time.Time `json:"last_read"`
}

type Store interface {
	ReadIDs(context.Context) (map[int]bool, error)
	MarkRead(context.Context, int) error
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
	return filepath.Join(home, ".hackernews", "history.json"), nil
}

func (s JSONStore) ReadIDs(ctx context.Context) (map[int]bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := s.read()
	if err != nil {
		return nil, err
	}
	ids := make(map[int]bool, len(entries))
	for _, entry := range entries {
		ids[entry.ID] = true
	}
	return ids, nil
}

func (s JSONStore) MarkRead(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == 0 {
		return nil
	}
	entries, err := s.read()
	if err != nil {
		return err
	}
	now := s.now().UTC()
	for i := range entries {
		if entries[i].ID == id {
			entries[i].LastRead = now
			return s.write(entries)
		}
	}
	entries = append(entries, Entry{ID: id, FirstRead: now, LastRead: now})
	return s.write(entries)
}

func (s JSONStore) read() ([]Entry, error) {
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
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s JSONStore) write(entries []Entry) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".history-*.tmp")
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
