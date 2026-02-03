package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/2ykwang/mac-cleanup-go/internal/cli"
	"github.com/2ykwang/mac-cleanup-go/internal/config"
	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/tui"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
	pkgversion "github.com/2ykwang/mac-cleanup-go/internal/version"
)

// version is set via ldflags: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	// Flags
	showVersion := flag.Bool("version", false, "Show version")
	shortVersion := flag.Bool("v", false, "Show version (short)")
	doUpdate := flag.Bool("update", false, "Update to latest version via Homebrew")
	debugMode := flag.Bool("debug", false, "Enable debug logging to ~/.config/mac-cleanup-go/debug.log")
	selectTargets := flag.Bool("select", false, "Select cleanup targets")
	doClean := flag.Bool("clean", false, "Clean selected targets")
	dryRun := flag.Bool("dry-run", false, "Show report without deleting (requires --clean)")
	flag.Parse()

	// Initialize logger: --debug flag or DEBUG env var
	debug := *debugMode || os.Getenv("DEBUG") == "true"
	if err := logger.Init(debug); err != nil {
		fmt.Fprintf(os.Stderr, "warning: logging disabled: %v\n", err)
	}
	defer logger.Close()

	if *showVersion || *shortVersion {
		fmt.Printf("mac-cleanup %s\n", version)
		return
	}

	if *doUpdate {
		fmt.Println("Updating mac-cleanup-go via Homebrew...")
		if err := pkgversion.RunUpdate(); err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Update complete!")
		return
	}

	cfg, err := config.LoadEmbedded()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if *selectTargets && *doClean {
		fmt.Fprintln(os.Stderr, "error: --select cannot be combined with --clean")
		os.Exit(1)
	}
	if *dryRun && !*doClean {
		fmt.Fprintln(os.Stderr, "error: --dry-run requires --clean")
		os.Exit(1)
	}

	if *selectTargets {
		p := tea.NewProgram(
			tui.NewConfigModel(cfg),
			tea.WithAltScreen(),
		)

		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *doClean {
		userCfg, err := userconfig.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load user config: %v\n", err)
			os.Exit(1)
		}

		runner, err := cli.NewRunner(cfg, userCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize cli runner: %v\n", err)
			os.Exit(1)
		}

		report, warnings, err := runner.Run(*dryRun)
		if err != nil {
			switch {
			case errors.Is(err, cli.ErrNoSelection):
				fmt.Fprintln(os.Stderr, "no targets selected. run `mac-cleanup --select` to configure.")
			case errors.Is(err, cli.ErrNoEligibleTargets):
				fmt.Fprintln(os.Stderr, "no eligible targets selected. run `mac-cleanup --select` to configure.")
			default:
				fmt.Fprintf(os.Stderr, "clean failed: %v\n", err)
			}
			os.Exit(1)
		}

		if len(warnings) > 0 {
			fmt.Fprintln(os.Stderr, "Warnings:")
			for _, warning := range warnings {
				fmt.Fprintln(os.Stderr, "  - "+warning)
			}
		}

		fmt.Print(cli.FormatReport(report, *dryRun))
		return
	}

	p := tea.NewProgram(
		tui.NewModel(cfg, version),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
