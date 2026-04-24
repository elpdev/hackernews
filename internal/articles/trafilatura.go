package articles

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

//go:embed trafilatura_extract.py
var embeddedTrafilaturaScript string

type TrafilaturaExtractor struct {
	Python string
	Script string
	Limit  time.Duration
}

func NewTrafilaturaExtractor() TrafilaturaExtractor {
	return TrafilaturaExtractor{
		Python: "python3",
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
	limit := e.Limit
	if limit <= 0 {
		limit = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	args := trafilaturaCommandArgs(script, url)
	cmd := exec.CommandContext(ctx, python, args...)
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

func trafilaturaCommandArgs(script, url string) []string {
	if strings.TrimSpace(script) != "" {
		return []string{script, url}
	}
	return []string{"-c", embeddedTrafilaturaScript, url}
}
