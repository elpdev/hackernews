package articles

import (
	"errors"
	"strings"
	"testing"
)

func TestTrafilaturaExtractorRejectsEmptyURL(t *testing.T) {
	extractor := NewTrafilaturaExtractor()
	if _, err := extractor.Extract(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewTrafilaturaExtractorUsesEmbeddedScript(t *testing.T) {
	extractor := NewTrafilaturaExtractor()
	if extractor.Script != "" {
		t.Fatalf("expected default extractor to use embedded script, got %q", extractor.Script)
	}
	if extractor.Trafilatura != "trafilatura" {
		t.Fatalf("expected default CLI fallback command, got %q", extractor.Trafilatura)
	}

	args := trafilaturaCommandArgs(extractor.Script, "https://example.com/article")
	if len(args) != 3 {
		t.Fatalf("expected python -c invocation args, got %#v", args)
	}
	if args[0] != "-c" {
		t.Fatalf("expected embedded script passed with -c, got %#v", args)
	}
	if !strings.Contains(args[1], "import trafilatura") {
		t.Fatalf("expected embedded helper script, got %q", args[1])
	}
	if args[2] != "https://example.com/article" {
		t.Fatalf("expected URL arg preserved, got %q", args[2])
	}
}

func TestTrafilaturaCommandArgsUsesExplicitScript(t *testing.T) {
	args := trafilaturaCommandArgs("/tmp/trafilatura_extract.py", "https://example.com/article")
	if len(args) != 2 {
		t.Fatalf("expected script path invocation args, got %#v", args)
	}
	if args[0] != "/tmp/trafilatura_extract.py" || args[1] != "https://example.com/article" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestTrafilaturaCLIArgsExtractMarkdown(t *testing.T) {
	args := trafilaturaCLIArgs("https://example.com/article")
	want := []string{"--markdown", "--no-comments", "--no-tables", "-u", "https://example.com/article"}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("trafilaturaCLIArgs() = %#v, want %#v", args, want)
	}
}

func TestMissingPythonTrafilatura(t *testing.T) {
	for _, msg := range []string{
		"Python package 'trafilatura' is not installed. Run: python3 -m pip install trafilatura",
		"ModuleNotFoundError: No module named 'trafilatura'",
	} {
		if !missingPythonTrafilatura(errors.New(msg)) {
			t.Fatalf("expected missing package error detected for %q", msg)
		}
	}
}
