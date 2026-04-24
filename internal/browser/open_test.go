package browser

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestOpenerForByGOOS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{"linux", "xdg-open", []string{"https://example.com"}},
		{"freebsd", "xdg-open", []string{"https://example.com"}},
		{"darwin", "open", []string{"https://example.com"}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", "https://example.com"}},
	}
	for _, c := range cases {
		name, args, err := openerFor(c.goos, "https://example.com")
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

	if _, _, err := openerFor("plan9", "https://example.com"); err == nil {
		t.Fatal("expected error for unsupported GOOS")
	}
}

func TestOpenRejectsEmptyURL(t *testing.T) {
	if err := Open("   "); err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestOpenLaunchesCommand(t *testing.T) {
	if _, _, err := openerFor(runtime.GOOS, "https://example.com"); err != nil {
		t.Skipf("no opener on %s: %v", runtime.GOOS, err)
	}

	var gotName string
	var gotArgs []string
	orig := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = args
		// Run a harmless command that exits successfully so cmd.Start/Wait succeed.
		return exec.Command("true")
	}
	defer func() { execCommand = orig }()

	if err := Open("https://example.com/x"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd", "netbsd":
		if gotName != "xdg-open" {
			t.Fatalf("name = %q, want xdg-open", gotName)
		}
	case "darwin":
		if gotName != "open" {
			t.Fatalf("name = %q, want open", gotName)
		}
	case "windows":
		if gotName != "rundll32" {
			t.Fatalf("name = %q, want rundll32", gotName)
		}
	}
	if len(gotArgs) == 0 || gotArgs[len(gotArgs)-1] != "https://example.com/x" {
		t.Fatalf("expected URL as final arg, got %v", gotArgs)
	}
}
