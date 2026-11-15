package main

import (
	_ "embed"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/2ykwang/mac-cleanup-go/internal/cli"
	"github.com/2ykwang/mac-cleanup-go/internal/tui"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
)

// version is set via ldflags: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

// skill ships with the binary so agents always read instructions that match it.
//
//go:embed internal/cli/skills/mac-cleanup/SKILL.md
var skill string

func main() {
	app := &cli.App{
		Version: version,
		Skill:   skill,
		In:      os.Stdin,
		Out:     os.Stdout,
		Err:     os.Stderr,
		RunTUI: func(cfg *types.Config) error {
			_, err := tea.NewProgram(tui.NewModel(cfg, version)).Run()
			return err
		},
		RunSelect: func(cfg *types.Config) error {
			_, err := tea.NewProgram(tui.NewConfigModel(cfg)).Run()
			return err
		},
	}
	os.Exit(app.Run(os.Args[1:]))
}
