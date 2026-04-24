// Package clipboard copies text to the user's system clipboard.
package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

var (
	execCommand = exec.Command
	lookPath    = exec.LookPath
)

func Copy(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("no text to copy")
	}
	name, args, err := copierFor(runtime.GOOS)
	if err != nil {
		return err
	}
	cmd := execCommand(name, args...)
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copy with %s: %w", name, err)
	}
	return nil
}

func copierFor(goos string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "pbcopy", nil, nil
	case "windows":
		return "clip", nil, nil
	case "linux", "freebsd", "openbsd", "netbsd":
		candidates := []struct {
			name string
			args []string
		}{
			{name: "wl-copy"},
			{name: "xclip", args: []string{"-selection", "clipboard"}},
			{name: "xsel", args: []string{"--clipboard", "--input"}},
		}
		for _, candidate := range candidates {
			if _, err := lookPath(candidate.name); err == nil {
				return candidate.name, candidate.args, nil
			}
		}
		return "", nil, fmt.Errorf("no clipboard command found; install wl-copy, xclip, or xsel")
	default:
		return "", nil, fmt.Errorf("copying to clipboard is not supported on %s", goos)
	}
}
