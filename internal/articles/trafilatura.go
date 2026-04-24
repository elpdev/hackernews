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
	Python      string
	Script      string
	Trafilatura string
	Limit       time.Duration
}

func NewTrafilaturaExtractor() TrafilaturaExtractor {
	return TrafilaturaExtractor{
		Python:      "python3",
		Trafilatura: "trafilatura",
		Limit:       30 * time.Second,
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
	trafilatura := e.Trafilatura
	if trafilatura == "" {
		trafilatura = "trafilatura"
	}
	script := e.Script
	limit := e.Limit
	if limit <= 0 {
		limit = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	article, err := extractWithPython(ctx, python, script, url)
	if err != nil && ctx.Err() == nil && missingPythonTrafilatura(err) {
		pythonErr := err
		article, err = extractWithTrafilaturaCLI(ctx, trafilatura, url)
		if err != nil {
			err = fmt.Errorf("python package unavailable (%s); trafilatura CLI fallback failed (%s)", pythonErr, err)
		}
	}
	if err != nil {
		msg := err.Error()
		if ctx.Err() == context.DeadlineExceeded {
			msg = "article extraction timed out"
		}
		return Article{}, fmt.Errorf("trafilatura extraction failed: %s", msg)
	}
	if strings.TrimSpace(article.Markdown) == "" {
		return Article{}, errors.New("trafilatura did not find readable article content")
	}
	return article, nil
}

func extractWithPython(ctx context.Context, python, script, url string) (Article, error) {
	args := trafilaturaCommandArgs(script, url)
	cmd := exec.CommandContext(ctx, python, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Article{}, commandError(out, err)
	}

	var article Article
	if err := json.Unmarshal(out, &article); err != nil {
		return Article{}, fmt.Errorf("invalid trafilatura output: %w", err)
	}
	return article, nil
}

func extractWithTrafilaturaCLI(ctx context.Context, trafilatura, url string) (Article, error) {
	cmd := exec.CommandContext(ctx, trafilatura, trafilaturaCLIArgs(url)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Article{}, commandError(out, err)
	}
	return Article{URL: url, Markdown: strings.TrimSpace(string(out))}, nil
}

func commandError(out []byte, err error) error {
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		msg = err.Error()
	}
	return errors.New(msg)
}

func trafilaturaCommandArgs(script, url string) []string {
	if strings.TrimSpace(script) != "" {
		return []string{script, url}
	}
	return []string{"-c", embeddedTrafilaturaScript, url}
}

func trafilaturaCLIArgs(url string) []string {
	return []string{"--markdown", "--images", "--no-comments", "--no-tables", "-u", url}
}

func missingPythonTrafilatura(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "Python package 'trafilatura' is not installed") ||
		strings.Contains(msg, "No module named 'trafilatura'")
}
