package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/elpdev/hackernews/internal/app"
	tea "charm.land/bubbletea/v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "print version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("hackernews %s (%s, %s)\n", version, commit, date)
		return
	}

	meta := app.BuildInfo{Version: version, Commit: commit, Date: date}
	program := tea.NewProgram(app.New(meta))
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "hackernews: %v\n", err)
		os.Exit(1)
	}
}
