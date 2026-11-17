package cli

import (
	"fmt"
	"strings"
)

const rootUsage = `mac-cleanup - preview-first cleanup for macOS caches, logs, and temp files

Usage:
  mac-cleanup                 Open the interactive TUI (needs a terminal)
  mac-cleanup <command> [flags]

Commands:
%s
Global flags:
  --debug       Write debug logs to ~/.config/mac-cleanup-go/debug.log
  -h, --help    Show help
  -v, --version Show version

Typical flow:
  mac-cleanup scan                                # what can be cleaned, by target ID
  mac-cleanup scan xcode-derived --items          # look inside one target
  mac-cleanup clean xcode-derived npm --dry-run   # preview exactly what clean does
  mac-cleanup clean xcode-derived npm --yes       # move the items to Trash

Exit codes: 0 ok, 1 error, 2 usage error or confirmation needed, 3 some items failed.
Add --json to targets, scan, and clean for machine-readable output.
AI agents: run ` + "`mac-cleanup skill`" + ` for the full usage guide.
`

const targetsUsage = `Usage: mac-cleanup targets [<id>...] [flags]

List cleanup targets from the built-in catalog. Nothing is scanned.
Pass IDs to also show paths, blocking processes, and manual guides.

Flags:
  --group <id>     Filter by group: system, browser, dev, app, storage
  --safety <lvl>   Filter by safety: safe, moderate, risky (comma-separated)
  --json           Print JSON: {"targets":[{id,name,group,safety,method,note,...}]}
`

const scanUsage = `Usage: mac-cleanup scan [<id>...] [flags]

Scan targets and report sizes. Read-only. Without IDs, scans every target.

Flags:
  --group <id>     Filter by group
  --safety <lvl>   Filter by safety (comma-separated)
  --items          Include items per target, largest first
  --limit <n>      Max items per target with --items (default 20, 0 = all)
  --json           Print JSON

JSON fields per target:
  status           ok | unavailable | blocked | error
  size_bytes       Everything found
  cleanable_bytes  What ` + "`clean <id>`" + ` would remove: excludes locked and saved-excluded
                   items; 0 for risky (needs --item) and manual targets
  blocked_by       Running process that must quit first (status blocked)
  items[]          path, name, display_name, size_bytes, file_count,
                   modified_at, locked, excluded
`

const cleanUsage = `Usage: mac-cleanup clean <id>... [flags]
       mac-cleanup clean --selected [flags]

Scan the given targets and clean them. Most targets move items to Trash;
check "method" in scan output: permanent deletes, builtin uses target-specific
cleanup. Homebrew moves only its scanned cache to Trash and keeps formula versions.

Flags:
  --dry-run        Show what would be cleaned; change nothing
  --yes            Skip the confirmation prompt (required without a terminal)
  --item <path>    Clean only this scanned item path (repeatable). Required
                   for risky targets. Paths must come from ` + "`scan --items`" + `.
                   An explicit --item also overrides a saved exclusion.
  --selected       Use targets saved with ` + "`mac-cleanup select`" + `
  --json           Print JSON

Never cleaned: manual targets, risky targets without --item, locked items,
targets whose blocking process is running. These appear under "skipped".
JSON dry-runs include item_paths for each planned target. Clean rescans before
execution; a dry-run is not a frozen plan.

Exit codes: 0 ok, 1 error, 2 usage error or confirmation needed, 3 some items failed.
`

const selectUsage = `Usage: mac-cleanup select

Open an interactive picker and save targets for ` + "`mac-cleanup clean --selected`" + `.
Needs a terminal.
`

const skillUsage = `Usage: mac-cleanup skill

Print the agent skill (SKILL.md) matching this binary's version.
Save it where your agent loads skills, for example:
  mkdir -p ~/.claude/skills/mac-cleanup
  mac-cleanup skill > ~/.claude/skills/mac-cleanup/SKILL.md
`

const versionUsage = `Usage: mac-cleanup version
`

const updateUsage = `Usage: mac-cleanup update

Upgrade mac-cleanup-go with Homebrew.
`

const helpUsage = `Usage: mac-cleanup help [<command>]
`

func (a *App) runHelp(args []string) error {
	if len(args) == 0 {
		var b strings.Builder
		for _, cmd := range commandList() {
			fmt.Fprintf(&b, "  %-9s %s\n", cmd.name, cmd.summary)
		}
		fmt.Fprintf(a.Out, rootUsage, b.String())
		return nil
	}
	cmd, ok := findCommand(args[0])
	if !ok {
		return usageErrorf("unknown_command", "run `mac-cleanup help`", "unknown command %q", args[0])
	}
	fmt.Fprint(a.Out, cmd.usage)
	return nil
}
