package articles

import (
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
