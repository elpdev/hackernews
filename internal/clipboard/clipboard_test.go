package clipboard

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestCopierForByGOOS(t *testing.T) {
	orig := lookPath
	lookPath = func(name string) (string, error) {
		if name == "wl-copy" {
			return "/usr/bin/wl-copy", nil
		}
		return "", errors.New("not found")
	}
	defer func() { lookPath = orig }()

	cases := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{"darwin", "pbcopy", nil},
		{"windows", "clip", nil},
		{"linux", "wl-copy", nil},
		{"freebsd", "wl-copy", nil},
	}
	for _, c := range cases {
		name, args, err := copierFor(c.goos)
		if err != nil {
			t.Fatalf("%s: unexpected error %v", c.goos, err)
		}
		if name != c.wantName {
			t.Fatalf("%s: name = %q, want %q", c.goos, name, c.wantName)
		}
		if strings.Join(args, " ") != strings.Join(c.wantArgs, " ") {
			t.Fatalf("%s: args = %v, want %v", c.goos, args, c.wantArgs)
		}
	}

	if _, _, err := copierFor("plan9"); err == nil {
		t.Fatal("expected error for unsupported GOOS")
	}
}

func TestCopierForFallsBackToXclip(t *testing.T) {
	orig := lookPath
	lookPath = func(name string) (string, error) {
		if name == "xclip" {
			return "/usr/bin/xclip", nil
		}
		return "", errors.New("not found")
	}
	defer func() { lookPath = orig }()

	name, args, err := copierFor("linux")
	if err != nil {
		t.Fatalf("copierFor: %v", err)
	}
	if name != "xclip" {
		t.Fatalf("name = %q, want xclip", name)
	}
	if strings.Join(args, " ") != "-selection clipboard" {
		t.Fatalf("args = %v, want xclip clipboard selection", args)
	}
}

func TestCopyRejectsEmptyText(t *testing.T) {
	if err := Copy("   "); err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestCopyRunsCommandWithStdin(t *testing.T) {
	origCommand := execCommand
	origLookPath := lookPath
	lookPath = func(name string) (string, error) { return name, nil }
	var gotName string
	execCommand = func(name string, args ...string) *exec.Cmd {
		gotName = name
		return exec.Command("true")
	}
	defer func() {
		execCommand = origCommand
		lookPath = origLookPath
	}()

	if err := Copy("https://example.com"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if gotName == "" {
		t.Fatal("expected copy command to run")
	}
}
