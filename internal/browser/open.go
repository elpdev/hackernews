// Package browser launches a URL in the user's default web browser.
package browser

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

var execCommand = exec.Command

func Open(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("no URL to open")
	}
	name, args, err := openerFor(runtime.GOOS, url)
	if err != nil {
		return err
	}
	cmd := execCommand(name, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch %s: %w", name, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func openerFor(goos, url string) (string, []string, error) {
	switch goos {
	case "linux", "freebsd", "openbsd", "netbsd":
		return "xdg-open", []string{url}, nil
	case "darwin":
		return "open", []string{url}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}, nil
	default:
		return "", nil, fmt.Errorf("opening a browser is not supported on %s", goos)
	}
}
