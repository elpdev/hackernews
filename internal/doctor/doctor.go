package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/elpdev/hackernews/internal/config"
	"github.com/elpdev/hackernews/internal/history"
	"github.com/elpdev/hackernews/internal/saved"
)

type Status int

const (
	OK Status = iota
	Warn
	Fail
)

func (s Status) String() string {
	switch s {
	case OK:
		return "OK"
	case Warn:
		return "WARN"
	default:
		return "FAIL"
	}
}

type Check struct {
	Name    string
	Status  Status
	Message string
}

type Options struct {
	SyncEnabled bool
	SyncRemote  string
	SyncBranch  string
	SyncDir     string
}

func Run(ctx context.Context, options Options) []Check {
	checks := []Check{
		pathCheck("config", config.DefaultPath),
		pathCheck("saved articles", saved.DefaultPath),
		pathCheck("read history", history.DefaultPath),
		pythonCheck(ctx),
		trafilaturaCheck(ctx),
		browserCheck(),
		clipboardCheck(),
	}
	checks = append(checks, syncChecks(options)...)
	return checks
}

func pathCheck(name string, pathFunc func() (string, error)) Check {
	path, err := pathFunc()
	if err != nil {
		return Check{Name: name, Status: Warn, Message: err.Error()}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Check{Name: name, Status: Fail, Message: err.Error()}
	}
	tmp, err := os.CreateTemp(dir, ".doctor-*.tmp")
	if err != nil {
		return Check{Name: name, Status: Fail, Message: err.Error()}
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(tmpPath)
	return Check{Name: name, Status: OK, Message: path}
}

func pythonCheck(ctx context.Context) Check {
	if _, err := exec.LookPath("python3"); err != nil {
		return Check{Name: "python3", Status: Fail, Message: "python3 not found"}
	}
	if err := runCommand(ctx, "python3", "--version"); err != nil {
		return Check{Name: "python3", Status: Warn, Message: err.Error()}
	}
	return Check{Name: "python3", Status: OK, Message: "available"}
}

func trafilaturaCheck(ctx context.Context) Check {
	if err := runCommand(ctx, "python3", "-c", "import trafilatura"); err == nil {
		return Check{Name: "trafilatura", Status: OK, Message: "python package available"}
	}
	if _, err := exec.LookPath("trafilatura"); err == nil {
		return Check{Name: "trafilatura", Status: OK, Message: "CLI available"}
	}
	return Check{Name: "trafilatura", Status: Warn, Message: "install python package or CLI for article extraction"}
}

func browserCheck() Check {
	switch runtime.GOOS {
	case "darwin":
		return lookPathCheck("browser opener", []string{"open"})
	case "windows":
		return Check{Name: "browser opener", Status: OK, Message: "Windows URL handler"}
	default:
		return lookPathCheck("browser opener", []string{"xdg-open"})
	}
}

func clipboardCheck() Check {
	switch runtime.GOOS {
	case "darwin":
		return lookPathCheck("clipboard", []string{"pbcopy"})
	case "windows":
		return lookPathCheck("clipboard", []string{"clip"})
	default:
		return lookPathCheck("clipboard", []string{"wl-copy", "xclip", "xsel"})
	}
}

func syncChecks(options Options) []Check {
	if !options.SyncEnabled {
		return []Check{{Name: "sync", Status: Warn, Message: "not configured"}}
	}
	checks := []Check{lookPathCheck("git", []string{"git"})}
	if strings.TrimSpace(options.SyncRemote) == "" {
		checks = append(checks, Check{Name: "sync remote", Status: Fail, Message: "missing remote; run Setup Sync"})
	} else {
		checks = append(checks, Check{Name: "sync remote", Status: OK, Message: options.SyncRemote})
	}
	if strings.TrimSpace(options.SyncBranch) == "" {
		checks = append(checks, Check{Name: "sync branch", Status: Warn, Message: "empty; main will be used"})
	} else {
		checks = append(checks, Check{Name: "sync branch", Status: OK, Message: options.SyncBranch})
	}
	if strings.TrimSpace(options.SyncDir) == "" {
		checks = append(checks, Check{Name: "sync dir", Status: Fail, Message: "missing directory"})
	} else {
		checks = append(checks, Check{Name: "sync dir", Status: OK, Message: options.SyncDir})
	}
	return checks
}

func lookPathCheck(name string, commands []string) Check {
	for _, command := range commands {
		path, err := exec.LookPath(command)
		if err == nil {
			return Check{Name: name, Status: OK, Message: path}
		}
	}
	return Check{Name: name, Status: Warn, Message: fmt.Sprintf("none found: %s", strings.Join(commands, ", "))}
}

func runCommand(ctx context.Context, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return errors.New(message)
	}
	return nil
}
