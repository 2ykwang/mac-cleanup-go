# Go code

- Use `os.Lstat` to check that a path exists or to measure it, and don't follow symlinks. `os.Stat` reports a broken link as a missing file, and following links counts files outside the cleanup target.
- Wrap external commands in `context.WithTimeout`. Every target's `IsAvailable()` runs in sequence at startup, so one command that doesn't respond hangs the whole program.

# Tests

- Keep tests away from the real Trash, the installed `brew` and `docker`, and the real home directory. Stub external commands and OS calls by replacing the package-level function variables, and create files under `t.TempDir()`.
- To test cleanup, pass paths that don't exist or use `MethodPermanent` with `t.TempDir()`. Cleaning real files through the Trash path fills the user's Trash on every test run. Tests that exercise the Trash API itself must run only when an environment variable enables them.
- Restore replaced package variables with `defer` or `t.Cleanup`, and don't use `t.Parallel()`. Tests replace the same package variables, so parallel runs overwrite each other's stubs.
- Write one test function per scenario, named `TestSubject_Scenario`. Don't write table-driven tests.

# Cleanup targets (`internal/config/targets.yaml`)

- Don't rename or remove an `id` that has shipped. The user config stores selected targets and excluded paths by `id`, so changing one silently drops those settings.
- Before adding a path, check that no file can be matched by two items. Look for a pattern in another category, or another glob in the same category, that equals the new path, contains it, or sits inside it. Overlap counts the same size twice.
- Don't expect `system-cache` to filter out overlap. It drops one of its own items only when that item equals or sits under another category's YAML path. If the other category's path is deeper, the parent directory item that contains it stays. Paths that aren't in the YAML, such as one read from a command's output, must be excluded in code.
- Don't treat a passing `make targets-validate` as proof of safety. It doesn't catch duplicate `id`s or overlapping paths, and a duplicate `id` silently replaces the earlier entry.
- Give new targets `method: trash`. If data that can't be rebuilt (chat history, databases, volumes) may be mixed in, don't put it in the same category as regenerable caches. Split it into its own category with `safety: risky`.
- When a target needs its own implementation, add it to `builtinIDs` and `registerAllBuiltins` in `internal/target/registry.go` and to the YAML together. The `id` picks the scanner and `method` picks how it's cleaned. Unless `method` is `builtin`, the target's own `Clean` is never called.

# Scanning and cleaning

- A target's own `Clean` deletes only the items shown in the preview. Match the items it receives by exact path or ID, and don't substitute a broader command such as `prune` or `cleanup`.
- Lock state and running processes are only what the scan saw. Check again right before deleting.
- A selection or exclusion is identified by the category `id` and the `item.Path` string. An item's `Path` must be unique within a scan and identical across scans. Canonicalizing a path with `filepath.EvalSymlinks` or changing its format breaks the match with saved exclusions.
- Apply saved exclusions and the `risky` policy on every path that builds clean jobs (TUI, rescan, CLI). The policy doesn't live in one place. The TUI excludes every item the moment a `risky` category is selected, and the CLI skips the target unless `--item` is given.
- Make confirmation text and result reports match the category's actual `method`. `permanent` and Docker cleanup can't be undone, so don't say they move files to the Trash.

# TUI

- Scan results arrive per target, and a target has no items until its result arrives. Don't compute a safety action such as the `risky` auto-exclude from an empty result. Don't run side effects such as saving exclusions for a selection that was refused.
- Clamp every cursor, scroll offset and width derived from the terminal size or a filtered list before using it. A list with zero items and a width under 2 are normal states, and a negative width panics in `strings.Repeat`.
- When you add or change a key, update three places together. The `msg.String()` cases in `internal/tui/handlers.go`, the footer help text in `internal/tui/shortcuts.go` and the help table in `internal/tui/view_help.go` aren't connected.

# CLI and docs

- When you change a command, flag, error code, exit code or JSON field, update `internal/cli/help.go`, `internal/cli/skills/mac-cleanup/SKILL.md`, `README.md` and `assets/README_KO.md` in the same change. No test checks that they agree.
- When you edit `README.md`, edit the same section of `assets/README_KO.md`.

# Build and verification

- Before you finish, make `make fmt && make lint && go test ./...` pass. Format with `make fmt`, not `gofmt` directly. The lint config's formatters are stricter than `gofmt`.
- If you changed `internal/config/targets.yaml`, also run `make targets-validate`.
- Don't run `mac-cleanup clean ... --yes` to check a change. The tool deletes files in the real home directory. Use `--dry-run` to inspect the plan, and verify deletion with tests that use `t.TempDir()`.
- Bare `mac-cleanup`, `mac-cleanup select` and `make run` need a terminal. Without one, verify behavior with `scan` and `clean --dry-run`. A full scan is slow, so pass target IDs to narrow it.

# Commits and pull requests

- Write commit subjects and PR titles as capitalized imperative sentences. Don't add a prefix such as `feat:` or `fix:`, or a trailing period. Example: `Limit Homebrew cleanup to the scanned cache`
- Follow `.github/PULL_REQUEST_TEMPLATE.md` for the PR body and keep it short.
- Fix every review finding, or leave the reason for not fixing it in that thread. A finding that goes unanswered stays in the code.

# Rules hygiene

Agents read this whole file at the start of every task. The longer it gets, the harder the important rules are to see, so keep only the rules that are needed.

## High bar for new rules

Editing or clarifying an existing rule is always welcome. A new rule must meet **all four** criteria:

1. **Non-obvious.** Someone who knows the codebase would still get it wrong without the rule.
2. **Repeatedly encountered.** It came up more than once. Multiple hits in one task count.
3. **Specific enough to act on.** A concrete instruction, not a vague principle.
4. **Independent of context.** It doesn't need a particular task, conversation, PR or point in time to be understood or to stay true. It makes sense from this file and the repository alone.

## What not to put in this file

Don't add architecture descriptions such as package layout, data flow or key types. They go stale fast, and an agent can learn them by reading the code. Rules are **traps to avoid**, not **maps to follow**.

## Adding a rule

1. When you hit a non-obvious trap during a task, add a **"Suggested AGENTS.md additions"** heading to the PR description with the proposed text. Don't edit this file during feature or fix work.
2. With the proposal, cite evidence that the problem happened before: commits, PRs, issues or review findings. If you find none, say it's a single observation.
3. Once it meets the criteria above, add it in a dedicated commit and say **why** the rule exists.
