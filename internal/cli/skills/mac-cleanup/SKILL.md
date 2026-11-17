---
name: mac-cleanup
description: Find and safely free disk space on macOS with the mac-cleanup CLI (caches, logs, build artifacts, Docker, Homebrew, browser and app caches). Use when the user asks what is taking space, wants to clean caches or free storage, or mentions mac-cleanup. Do not use for deleting the user's own documents or projects, for uninstalling apps, or on non-macOS systems.
compatibility: macOS with the mac-cleanup binary on PATH (brew install mac-cleanup-go)
---

# mac-cleanup

mac-cleanup scans known cache and temp locations ("targets") and cleans them.
Most targets move files to the Trash. This skill is printed by
`mac-cleanup skill`, so it matches the installed binary. If a flag here
disagrees with `mac-cleanup help <command>`, trust the binary.

## Scope

`scan` measures only known cleanup targets (caches, logs, build artifacts,
backups, old downloads). It doesn't see documents, photos, apps, or other
user files, and it doesn't report total disk usage. When the user asks what
is using their disk, say this up front. Use `df -h /` if they want the total.

## Rules

- Never run bare `mac-cleanup` or `mac-cleanup select`. They open an
  interactive TUI and fail with `not_a_terminal` without a terminal.
- Pass `--json` to `targets`, `scan`, and `clean`. Data goes to stdout; errors go to stderr as
  `{"error":{"code","message","hint"}}`. Follow the `hint`.
- Never run `clean ... --yes` until the user has approved the exact targets
  and items from a `--dry-run` you showed them.
- Sizes are integer scan estimates. They are not measured free disk space.
  Convert to GB/MB when talking to the user. Trash retains data until emptied.

## Workflow

1. Scan. A full scan takes 30s or more; narrow it when you can.
   ```sh
   mac-cleanup scan --json                    # every target
   mac-cleanup scan --group dev --json        # system, browser, dev, app, storage
   mac-cleanup scan docker npm --json         # specific target IDs
   ```
   Each target has `status`, `safety`, `method`, `disposal`, `size_bytes`, and
   `cleanable_bytes` (eligible scan estimate). Ignore
   targets with `status` `unavailable` or `item_count` 0.
2. Look inside targets before proposing them, especially large ones.
   ```sh
   mac-cleanup scan system-cache --items --json   # top 20 items by size
   mac-cleanup scan system-cache --items --limit 0 --json
   ```
3. Propose a plan to the user: target IDs, estimated sizes, safety, and disposal.
   Point out anything risky or permanent (see Safety).
4. Preview exactly what will happen.
   ```sh
   mac-cleanup clean docker npm --dry-run --json
   ```
   Check `targets` (what will be cleaned) and `skipped` (what will not, and why).
   Show `item_paths` to the user. For targets without `side_effect`, pass
   approved paths with `--item` in both preview and execution when the user
   wants to limit cleanup to those items.
5. After the user approves, run the same command with `--yes` instead of
   `--dry-run`.
   ```sh
   mac-cleanup clean docker npm --yes --json
   ```
6. Report `trashed_bytes` as moved to Trash and `permanently_deleted_bytes`
   as permanent deletion estimates. Report `processed_items`, `already_absent_items`,
   `unconfirmed_items`, `failed_items`, and `skipped` entries separately.

`planned_bytes` is the scan size of prepared items and can include later skips or
failures. A dry-run fills only this size field; both successful size fields stay 0.
All three size fields appear in totals and each target. Additional builtin cleanup
outside scanned items has unmeasured size.

`freed_bytes` is deprecated. It retains its legacy value: planned scan bytes in a
dry-run, legacy successful-item scan bytes in a clean, including paths already
absent or gone without a confirmed Trash movement. It is not free disk space.
`cleaned_items` and per-target `items` also retain legacy success/planned counts.
Use `processed_items` for confirmed cleanup outcomes.
Per-target `sip_skipped_items` is a legacy alias of `skipped_items`. Both include
all policy skips, not only SIP-protected paths.

Each clean invocation rescans; a dry-run is not a frozen plan. Contents of
listed directories may change too. Explain this before approval, and re-preview
if the selected scope or conditions change.

To clean only some items of a target, repeat `--item` with exact `path`
values from `scan <id> --items --json`:

```sh
mac-cleanup clean slack --item "/Users/me/Library/Application Support/Slack/Cache/Cache_Data" --dry-run --json
```

## Safety

- `safety`:
  - `safe`: regenerated automatically.
  - `moderate`: regenerated, but costs a re-download or rebuild. Mention it.
  - `risky`: may hold user data such as chat history. Plain `clean <id>`
    skips these (`risky_needs_item`). Show the items, and pass `--item` only
    for paths the user explicitly approved.
- `method`:
  - `trash`: moved to the Trash, recoverable until it is emptied.
  - `permanent`: deleted for good. The `trash` target empties the Trash.
    Ask about it separately.
  - `builtin`: dispatches to a target-specific cleaner. It does not promise
    Trash or permanent deletion. Docker items are images, volumes, and the build cache.
    Targets with `side_effect` in scan output (`homebrew` runs `brew cleanup`,
    which also removes old formula versions) only clean as a whole: tell the
    user the side effect before asking. `--item` on them fails with
    `item_not_supported`.
  - `manual`: never cleaned by the CLI. Relay `guide` from
    `mac-cleanup targets <id> --json` to the user.
- `disposal` describes the actual action: `trash`, `permanent`, `mixed`,
  `manual`, or `unknown`. Docker is permanent. Project caches, old downloads,
  and old releases go to Trash. Homebrew uses mixed cleanup in this version.
  Never infer disposal from `builtin`, or promise recovery for `unknown`.
- `status` `blocked` with `blocked_by`: that app must be quit first. Ask the
  user to quit it, then scan again. Do not kill processes yourself.
- Items with `locked: true` are in use and are skipped. Items with
  `excluded: true` were excluded by the user and are skipped unless passed
  with `--item`.

## Reference

| Command | Purpose |
| --- | --- |
| `targets [<id>...] [--group g] [--safety s] --json` | Catalog only, no scan. IDs add paths and guides |
| `scan [<id>...] [--group g] [--safety s] [--items] [--limit n] --json` | Read-only sizes |
| `clean <id>... (--dry-run \| --yes) [--item path]... --json` | Clean |
| `clean --selected ...` | Use targets the user saved with `mac-cleanup select` |
| `help <command>` | Exact flags for this version |

Exit codes:

| Code | Meaning | What to do |
| --- | --- | --- |
| 0 | Success. `skipped` may still be non-empty | Report results |
| 1 | Runtime error | Read `error.message` |
| 2 | Usage error or confirmation needed | Read `error.hint` and fix the command |
| 3 | Clean ran, but some items failed | Report `errors`; scan again before any retry |

Error codes you may see: `confirmation_required` (add `--dry-run` or, after
approval, `--yes`), `unknown_target`, `unknown_item` (path not in the scan;
copy it exactly and include its target ID), `missing_target`,
`item_not_supported` (target has a `side_effect`; clean it whole or not at
all), `no_selection`, `invalid_flag`, `not_a_terminal`, `moved_command`.
