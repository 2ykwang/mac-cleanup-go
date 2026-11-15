package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/2ykwang/mac-cleanup-go/internal/config"
	"github.com/2ykwang/mac-cleanup-go/internal/logger"
	"github.com/2ykwang/mac-cleanup-go/internal/target"
	"github.com/2ykwang/mac-cleanup-go/internal/types"
	"github.com/2ykwang/mac-cleanup-go/internal/userconfig"
	pkgversion "github.com/2ykwang/mac-cleanup-go/internal/version"
)

// Exit codes are part of the CLI contract. Scripts and agents branch on them.
const (
	ExitOK      = 0
	ExitError   = 1
	ExitUsage   = 2
	ExitPartial = 3
)

// Function variables for testing
var (
	initLogger     = logger.Init
	loadConfig     = config.LoadEmbedded
	loadUserConfig = userconfig.Load
	newRegistry    = target.DefaultRegistry
	runUpdate      = pkgversion.RunUpdate
)

// App is the command-line entry point. Interactive screens are injected so this
// package does not depend on the TUI.
type App struct {
	Version   string
	Skill     string
	In        io.Reader
	Out       io.Writer
	Err       io.Writer
	RunTUI    func(cfg *types.Config) error
	RunSelect func(cfg *types.Config) error
}

type command struct {
	name    string
	summary string
	usage   string
	run     func(a *App, args []string) error
}

// commandList is a function rather than a package var because help refers back
// to it, which would make an initialization cycle.
func commandList() []command {
	return []command{
		{"targets", "List cleanup targets (no scan)", targetsUsage, (*App).runTargets},
		{"scan", "Measure what each target would clean (read-only)", scanUsage, (*App).runScan},
		{"clean", "Clean the given targets", cleanUsage, (*App).runClean},
		{"select", "Choose targets for `clean --selected` (interactive)", selectUsage, (*App).runSelect},
		{"skill", "Print the agent skill (SKILL.md) for this version", skillUsage, (*App).runSkill},
		{"version", "Print the version", versionUsage, (*App).runVersion},
		{"update", "Update via Homebrew", updateUsage, (*App).runUpdate},
		{"help", "Show help for a command", helpUsage, (*App).runHelp},
	}
}

// Moved flags from the flag-based CLI, kept only to point at the replacement.
var legacyFlags = map[string]string{
	"--clean":   "mac-cleanup clean --selected",
	"--select":  "mac-cleanup select",
	"--dry-run": "mac-cleanup clean <id>... --dry-run",
	"--update":  "mac-cleanup update",
}

// Run executes the command line and returns the process exit code.
func (a *App) Run(args []string) int {
	debug := os.Getenv("DEBUG") == "true"
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--debug" {
			debug = true
			continue
		}
		rest = append(rest, arg)
	}

	if err := initLogger(debug); err != nil {
		fmt.Fprintf(a.Err, "warning: logging disabled: %v\n", err)
	}
	defer logger.Close()

	err := a.dispatch(rest)
	return a.exit(err, hasJSONFlag(rest))
}

func (a *App) dispatch(args []string) error {
	if len(args) == 0 {
		return a.runInteractive(a.RunTUI, "mac-cleanup")
	}

	name := args[0]
	switch name {
	case "-h", "--help":
		return a.runHelp(nil)
	case "-v", "--version":
		return a.runVersion(nil)
	}
	if replacement, ok := legacyFlags[name]; ok {
		return usageErrorf("moved_command", "use `"+replacement+"`", "%s was replaced by a subcommand", name)
	}
	if cmd, ok := findCommand(name); ok {
		return cmd.run(a, args[1:])
	}
	return usageErrorf("unknown_command", "run `mac-cleanup help`", "unknown command %q", name)
}

func findCommand(name string) (command, bool) {
	for _, cmd := range commandList() {
		if cmd.name == name {
			return cmd, true
		}
	}
	return command{}, false
}

func (a *App) exit(err error, jsonOutput bool) int {
	if err == nil {
		return ExitOK
	}
	var partial *partialError
	if errors.As(err, &partial) {
		return ExitPartial
	}

	var cerr *cliError
	if !errors.As(err, &cerr) {
		cerr = &cliError{Code: "failed", Message: err.Error(), Exit: ExitError}
	}
	if jsonOutput {
		writeJSONError(a.Err, cerr)
	} else {
		fmt.Fprintf(a.Err, "error: %s\n", cerr.Message)
		if cerr.Hint != "" {
			fmt.Fprintf(a.Err, "hint: %s\n", cerr.Hint)
		}
	}
	return cerr.Exit
}

// runInteractive guards TUI entry points so a non-terminal caller gets help
// instead of a screen that waits for keys forever.
func (a *App) runInteractive(run func(*types.Config) error, name string) error {
	if !isTerminal(a.In) || !isTerminal(a.Out) {
		return usageErrorf("not_a_terminal", "run `mac-cleanup help` for non-interactive commands",
			"`%s` is interactive and needs a terminal", name)
	}
	if run == nil {
		return fmt.Errorf("`%s` is not available in this build", name)
	}
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading targets: %w", err)
	}
	return run(cfg)
}

func (a *App) runSelect(args []string) error {
	if err := noArgs("select", args); err != nil {
		return a.flagError("select", err)
	}
	return a.runInteractive(a.RunSelect, "mac-cleanup select")
}

func (a *App) runSkill(args []string) error {
	if err := noArgs("skill", args); err != nil {
		return a.flagError("skill", err)
	}
	fmt.Fprint(a.Out, a.Skill)
	return nil
}

func (a *App) runVersion(args []string) error {
	if err := noArgs("version", args); err != nil {
		return a.flagError("version", err)
	}
	fmt.Fprintf(a.Out, "mac-cleanup %s\n", a.Version)
	return nil
}

func (a *App) runUpdate(args []string) error {
	if err := noArgs("update", args); err != nil {
		return a.flagError("update", err)
	}
	fmt.Fprintln(a.Err, "Updating mac-cleanup-go via Homebrew...")
	if err := runUpdate(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	fmt.Fprintln(a.Err, "Update complete!")
	return nil
}

// session holds what scan and clean need. Loaded per command so help, version,
// and skill never fail on a broken user config.
type session struct {
	cfg      *types.Config
	registry *target.Registry
	userCfg  *userconfig.UserConfig
}

func openSession() (*session, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading targets: %w", err)
	}
	registry, err := newRegistry(cfg)
	if err != nil {
		return nil, fmt.Errorf("building target registry: %w", err)
	}
	userCfg, err := loadUserConfig()
	if err != nil {
		return nil, fmt.Errorf("loading user config: %w", err)
	}
	return &session{cfg: cfg, registry: registry, userCfg: userCfg}, nil
}
