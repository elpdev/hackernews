package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestDetectProtocol(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	t.Setenv("TERM", "xterm-256color")
	if got := DetectProtocol(); got != ProtocolITerm2 {
		t.Fatalf("expected iTerm2 protocol, got %v", got)
	}

	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "xterm-kitty")
	if got := DetectProtocol(); got != ProtocolKitty {
		t.Fatalf("expected kitty protocol, got %v", got)
	}

	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "foot")
	if got := DetectProtocol(); got != ProtocolSixel {
		t.Fatalf("expected sixel protocol, got %v", got)
	}
}

func TestRenderBytesNoProtocol(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "xterm-256color")
	rendered, rows, err := RenderBytes(tinyPNG(t), 20)
	if err != nil {
		t.Fatalf("render bytes: %v", err)
	}
	if rendered != "" || rows != 0 {
		t.Fatalf("expected empty render for unsupported protocol, got %q rows=%d", rendered, rows)
	}
}

func TestRenderBytesITerm2(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "WezTerm")
	t.Setenv("TERM", "wezterm")
	rendered, rows, err := RenderBytes(tinyPNG(t), 20)
	if err != nil {
		t.Fatalf("render bytes: %v", err)
	}
	if rows < 1 {
		t.Fatalf("expected rows > 0, got %d", rows)
	}
	if !strings.Contains(rendered, "]1337;") || !strings.Contains(rendered, "inline=1") {
		t.Fatalf("expected OSC 1337 image sequence, got %q", rendered)
	}
}

func TestRenderBytesKittyInsideTmux(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "xterm-kitty")
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")
	rendered, _, err := RenderBytes(tinyPNG(t), 24)
	if err != nil {
		t.Fatalf("render bytes: %v", err)
	}
	if !strings.Contains(rendered, "\x1bPtmux;") {
		t.Fatalf("expected tmux passthrough, got %q", rendered)
	}
	if !strings.Contains(rendered, "\x1b_G") {
		t.Fatalf("expected kitty graphics payload, got %q", rendered)
	}
	if strings.Contains(rendered, "z=-1") {
		t.Fatalf("expected kitty placement to render above background, got %q", rendered)
	}
}

func TestRenderBytesSixel(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "foot")
	rendered, rows, err := RenderBytes(tinyPNG(t), 16)
	if err != nil {
		t.Fatalf("render bytes: %v", err)
	}
	if rows < 1 {
		t.Fatalf("expected rows > 0, got %d", rows)
	}
	if !strings.Contains(rendered, "\x1bP1;1q") || !strings.Contains(rendered, "\x1b\\") {
		t.Fatalf("expected sixel DCS sequence, got %q", rendered)
	}
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 255, A: 255})
	img.Set(1, 0, color.NRGBA{G: 255, A: 255})
	img.Set(0, 1, color.NRGBA{B: 255, A: 255})
	img.Set(1, 1, color.NRGBA{R: 255, G: 255, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatalf("encode png fixture: %v", err)
	}
	return b.Bytes()
}
