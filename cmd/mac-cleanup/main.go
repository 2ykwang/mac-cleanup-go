package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"mac-cleanup-go/internal/config"
	"mac-cleanup-go/internal/tui"
)

// version is set via ldflags: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("mac-cleanup %s\n", version)
		return
	}

	cfg, err := config.Load("configs/targets.yaml")
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
