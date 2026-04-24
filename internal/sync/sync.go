package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/elpdev/hackernews/internal/history"
	"github.com/elpdev/hackernews/internal/saved"
)

const fileMode = 0o600

type Options struct {
	Remote      string
	Branch      string
	Dir         string
	SavedPath   string
	HistoryPath string
	DeletedPath string
}

type Result struct {
	SavedCount   int
	ReadCount    int
	DeletedCount int
	Committed    bool
}

func SyncNow(ctx context.Context, options Options) (Result, error) {
	options = withDefaults(options)
	if strings.TrimSpace(options.Remote) == "" {
		return Result{}, errors.New("sync remote is not configured")
	}
	if strings.TrimSpace(options.Dir) == "" {
		return Result{}, errors.New("sync directory is not configured")
	}
	if err := ensureRepo(ctx, options); err != nil {
		return Result{}, err
	}
	if err := pull(ctx, options); err != nil {
		return Result{}, err
	}

	localSaved, err := readJSON[[]saved.Article](options.SavedPath)
	if err != nil {
		return Result{}, err
	}
	remoteSaved, err := readJSON[[]saved.Article](filepath.Join(options.Dir, "saved.json"))
	if err != nil {
		return Result{}, err
	}
	localDeleted, err := readJSON[[]saved.DeletedArticle](options.DeletedPath)
	if err != nil {
		return Result{}, err
	}
	remoteDeleted, err := readJSON[[]saved.DeletedArticle](filepath.Join(options.Dir, "deleted_saved.json"))
	if err != nil {
		return Result{}, err
	}
	mergedSaved, mergedDeleted := MergeSaved(localSaved, remoteSaved, localDeleted, remoteDeleted)

	localHistory, err := readJSON[[]history.Entry](options.HistoryPath)
	if err != nil {
		return Result{}, err
	}
	remoteHistory, err := readJSON[[]history.Entry](filepath.Join(options.Dir, "history.json"))
	if err != nil {
		return Result{}, err
	}
	mergedHistory := MergeHistory(localHistory, remoteHistory)

	if err := writeJSON(options.SavedPath, mergedSaved); err != nil {
		return Result{}, err
	}
	if err := writeJSON(filepath.Join(options.Dir, "saved.json"), mergedSaved); err != nil {
		return Result{}, err
	}
	if err := writeJSON(options.DeletedPath, mergedDeleted); err != nil {
		return Result{}, err
	}
	if err := writeJSON(filepath.Join(options.Dir, "deleted_saved.json"), mergedDeleted); err != nil {
		return Result{}, err
	}
	if err := writeJSON(options.HistoryPath, mergedHistory); err != nil {
		return Result{}, err
	}
	if err := writeJSON(filepath.Join(options.Dir, "history.json"), mergedHistory); err != nil {
		return Result{}, err
	}

	committed, err := commitAndPush(ctx, options)
	if err != nil {
		return Result{}, err
	}
	return Result{SavedCount: len(mergedSaved), ReadCount: len(mergedHistory), DeletedCount: len(mergedDeleted), Committed: committed}, nil
}

func MergeHistory(a, b []history.Entry) []history.Entry {
	byID := make(map[int]history.Entry)
	for _, entry := range append(a, b...) {
		if entry.ID == 0 {
			continue
		}
		existing, ok := byID[entry.ID]
		if !ok {
			byID[entry.ID] = entry
			continue
		}
		if entry.FirstRead.Before(existing.FirstRead) || existing.FirstRead.IsZero() {
			existing.FirstRead = entry.FirstRead
		}
		if entry.LastRead.After(existing.LastRead) {
			existing.LastRead = entry.LastRead
		}
		byID[entry.ID] = existing
	}
	entries := make([]history.Entry, 0, len(byID))
	for _, entry := range byID {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastRead.After(entries[j].LastRead)
	})
	return entries
}

func MergeSaved(localSaved, remoteSaved []saved.Article, localDeleted, remoteDeleted []saved.DeletedArticle) ([]saved.Article, []saved.DeletedArticle) {
	articles := make(map[int]saved.Article)
	for _, item := range append(localSaved, remoteSaved...) {
		if item.ID == 0 {
			continue
		}
		if existing, ok := articles[item.ID]; !ok || item.SavedAt.After(existing.SavedAt) || richerArticle(item, existing) {
			articles[item.ID] = item
		}
	}
	deleted := make(map[int]saved.DeletedArticle)
	for _, item := range append(localDeleted, remoteDeleted...) {
		if item.ID == 0 {
			continue
		}
		if existing, ok := deleted[item.ID]; !ok || item.DeletedAt.After(existing.DeletedAt) {
			deleted[item.ID] = item
		}
	}
	for id, tombstone := range deleted {
		article, ok := articles[id]
		if !ok {
			continue
		}
		if tombstone.DeletedAt.IsZero() || !article.SavedAt.After(tombstone.DeletedAt) {
			delete(articles, id)
			continue
		}
		delete(deleted, id)
	}
	mergedSaved := make([]saved.Article, 0, len(articles))
	for _, item := range articles {
		mergedSaved = append(mergedSaved, item)
	}
	sort.SliceStable(mergedSaved, func(i, j int) bool {
		return mergedSaved[i].SavedAt.After(mergedSaved[j].SavedAt)
	})
	mergedDeleted := make([]saved.DeletedArticle, 0, len(deleted))
	for _, item := range deleted {
		mergedDeleted = append(mergedDeleted, item)
	}
	sort.SliceStable(mergedDeleted, func(i, j int) bool {
		return mergedDeleted[i].DeletedAt.After(mergedDeleted[j].DeletedAt)
	})
	return mergedSaved, mergedDeleted
}

func richerArticle(candidate, existing saved.Article) bool {
	return candidate.SavedAt.Equal(existing.SavedAt) && len(candidate.Article.Markdown) > len(existing.Article.Markdown)
}

func withDefaults(options Options) Options {
	if strings.TrimSpace(options.Branch) == "" {
		options.Branch = "main"
	}
	if strings.TrimSpace(options.SavedPath) == "" {
		options.SavedPath, _ = saved.DefaultPath()
	}
	if strings.TrimSpace(options.HistoryPath) == "" {
		options.HistoryPath, _ = history.DefaultPath()
	}
	if strings.TrimSpace(options.DeletedPath) == "" {
		options.DeletedPath, _ = saved.DefaultDeletedPath()
	}
	options.Dir = expandHome(options.Dir)
	return options
}

func ensureRepo(ctx context.Context, options Options) error {
	if _, err := os.Stat(filepath.Join(options.Dir, ".git")); err == nil {
		return runGit(ctx, options.Dir, "checkout", "-B", options.Branch)
	}
	if err := os.MkdirAll(filepath.Dir(options.Dir), 0o700); err != nil {
		return err
	}
	if err := runGit(ctx, "", "clone", options.Remote, options.Dir); err != nil {
		return err
	}
	return runGit(ctx, options.Dir, "checkout", "-B", options.Branch)
}

func pull(ctx context.Context, options Options) error {
	err := runGit(ctx, options.Dir, "pull", "--rebase", "origin", options.Branch)
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "couldn't find remote ref") || strings.Contains(message, "could not find remote ref") {
		return nil
	}
	return err
}

func commitAndPush(ctx context.Context, options Options) (bool, error) {
	if err := runGit(ctx, options.Dir, "add", "saved.json", "history.json", "deleted_saved.json"); err != nil {
		return false, err
	}
	changed, err := hasStagedChanges(ctx, options.Dir)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	message := fmt.Sprintf("Sync Hackernews state %s", time.Now().UTC().Format(time.RFC3339))
	if err := runGit(ctx, options.Dir, "commit", "-m", message); err != nil {
		return false, err
	}
	if err := runGit(ctx, options.Dir, "push", "-u", "origin", options.Branch); err != nil {
		return false, err
	}
	return true, nil
}

func hasStagedChanges(ctx context.Context, dir string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return true, nil
	}
	if stderr.Len() > 0 {
		return false, fmt.Errorf("git diff failed: %s", strings.TrimSpace(stderr.String()))
	}
	return false, err
}

func runGit(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if output, err := cmd.Output(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(string(output))
		}
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Errorf("git %s failed: %s", strings.Join(args, " "), detail)
	}
	return nil
}

func readJSON[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return zero, nil
	}
	if err != nil {
		return zero, err
	}
	if len(data) == 0 {
		return zero, nil
	}
	if err := json.Unmarshal(data, &zero); err != nil {
		return zero, err
	}
	return zero, nil
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".sync-*.tmp")
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
	return os.Rename(tmpPath, path)
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
