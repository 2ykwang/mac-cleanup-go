package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"mac-cleanup-go/internal/cli"
	"mac-cleanup-go/internal/config"
	"mac-cleanup-go/internal/tui"
)

// version is set via ldflags: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	// Flags
	showVersion := flag.Bool("version", false, "Show version")
	shortVersion := flag.Bool("v", false, "Show version (short)")
	dangerouslyDelete := flag.Bool("dangerously-delete", false, "Enable permanent deletion (default: move to Trash)")
	cleanMode := flag.Bool("clean", false, "Run cleanup with saved profile (no TUI)")
	dryRun := flag.Bool("dry-run", false, "Show what would be cleaned without actually cleaning")
	flag.Parse()

	if *showVersion || *shortVersion {
		fmt.Printf("mac-cleanup %s\n", version)
		return
	}

	cfg, err := config.LoadEmbedded()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// CLI clean mode
	if *cleanMode {
		if err := cli.Run(cfg, *dangerouslyDelete, *dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// TUI mode
	p := tea.NewProgram(
		tui.NewModel(cfg, *dangerouslyDelete),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
