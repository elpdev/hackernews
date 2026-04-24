package articles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type TrafilaturaExtractor struct {
	Python string
	Script string
	Limit  time.Duration
}

func NewTrafilaturaExtractor() TrafilaturaExtractor {
	_, file, _, ok := runtime.Caller(0)
	script := filepath.Join("internal", "articles", "trafilatura_extract.py")
	if ok {
		script = filepath.Join(filepath.Dir(file), "trafilatura_extract.py")
	}
	return TrafilaturaExtractor{
		Python: "python3",
		Script: script,
		Limit:  30 * time.Second,
	}
}

func (e TrafilaturaExtractor) Extract(url string) (Article, error) {
	if strings.TrimSpace(url) == "" {
		return Article{}, errors.New("story has no article URL")
	}
	python := e.Python
	if python == "" {
		python = "python3"
	}
	script := e.Script
	if script == "" {
		script = filepath.Join("internal", "articles", "trafilatura_extract.py")
	}
	limit := e.Limit
	if limit <= 0 {
		limit = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, script, url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		if ctx.Err() == context.DeadlineExceeded {
			msg = "article extraction timed out"
		}
		return Article{}, fmt.Errorf("trafilatura extraction failed: %s", msg)
	}

	var article Article
	if err := json.Unmarshal(out, &article); err != nil {
		return Article{}, fmt.Errorf("invalid trafilatura output: %w", err)
	}
	if strings.TrimSpace(article.Markdown) == "" {
		return Article{}, errors.New("trafilatura did not find readable article content")
	}
	return article, nil
}
