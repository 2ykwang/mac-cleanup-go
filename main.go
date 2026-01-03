package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/2ykwang/mac-cleanup-go/internal/config"
	"github.com/2ykwang/mac-cleanup-go/internal/tui"
)

// version is set via ldflags: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	// Flags
	showVersion := flag.Bool("version", false, "Show version")
	shortVersion := flag.Bool("v", false, "Show version (short)")
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

	p := tea.NewProgram(
		tui.NewModel(cfg),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
