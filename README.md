<div align="center">
  <h1>mac-cleanup-go</h1>
  <p>Preview-first TUI for cleaning macOS caches, logs, and temporary files.</p>
</div>

<p align="center">
  <a href="https://github.com/2ykwang/mac-cleanup-go/releases"><img src="https://img.shields.io/github/v/release/2ykwang/mac-cleanup-go" alt="GitHub Release"></a>
  <a href="https://goreportcard.com/report/github.com/2ykwang/mac-cleanup-go"><img src="https://goreportcard.com/badge/github.com/2ykwang/mac-cleanup-go" alt="Go Report Card"></a>
  <a href="https://github.com/2ykwang/mac-cleanup-go/actions/workflows/test.yml"><img src="https://github.com/2ykwang/mac-cleanup-go/actions/workflows/test.yml/badge.svg" alt="CI"></a>
  <a href="https://codecov.io/gh/2ykwang/mac-cleanup-go"><img src="https://codecov.io/gh/2ykwang/mac-cleanup-go/graph/badge.svg?token=ecH3KP0piI" alt="codecov"/></a>
  <a href="https://golangci-lint.run/"><img src="https://img.shields.io/badge/linted%20by-golangci--lint-brightgreen" alt="golangci-lint"></a>
</p>

<p align="center">
  <a href="README.md">English</a> | <a href="assets/README_KO.md">한국어</a>
</p>

## Overview

- Select items that take up space and delete them yourself.
- By default, items go to Trash; only the Trash category empties it permanently.
- Risky categories are unselected by default; even when selected, items are auto-excluded. (Include them in Preview to delete.)
- Manual categories show guides only.
- Scope: caches/logs/temp and selected app data (no system optimization or uninstaller).

![demo](assets/result_view.png)


## Quick Start

**1) Install**

```bash
brew install mac-cleanup-go
```

Or download the archive from [GitHub Releases](https://github.com/2ykwang/mac-cleanup-go/releases).

**2) Optional: Full Disk Access (needed for Trash/restricted locations)**
System Settings -> Privacy & Security -> Full Disk Access -> add Terminal

**3) Run**

```bash
mac-cleanup
```

Tip: Use Enter to preview, then y to proceed with deletion. Press ? to see key bindings.

![demo](assets/demo.gif)

- Update: `brew upgrade mac-cleanup-go` or `mac-cleanup --update`.
- Uninstall: `brew uninstall mac-cleanup-go`.
- Debug: `mac-cleanup --debug` saves log to `~/.config/mac-cleanup-go/debug.log`.

<details>
<summary><strong>Key bindings</strong></summary>

List view:

- `Up`/`Down` or `k`/`j`: move
- `Space`: select category
- `a`: select all, `d`: deselect all
- `Enter` or `p`: preview selection
- `?`: help, `q`: quit

Preview view:

- `Up`/`Down` or `k`/`j`: move
- `h`/`l`: previous/next category
- `Space`: toggle exclude
- `Enter`: drill into directory
- `/`: search, `s`: sort, `o`: open in Finder
- `a`: include all, `d`: exclude all
- `y`: delete (confirm), `esc`: back

Confirm view:

- `y` or `Enter`: confirm
- `n` or `esc`: cancel

</details>

## CLI mode

Configure targets and clean from the command line.

```bash
mac-cleanup --select                   # Configure cleanup targets
mac-cleanup --clean --dry-run          # Preview cleanup report
mac-cleanup --clean                    # Execute cleanup
```

For command-line cleanup, see the examples below.

<details>
<summary><strong>Example output</strong></summary>

**1) Select targets**

```
$ mac-cleanup --select

Select cleanup targets                  ● safe  ○ moderate
─────────────────────────────────────────────────────────
        Name                                      Size
  [ ] ● Trash                                      0 B
  [✓] ○ App Caches                              3.2 GB
  [✓] ○ System Logs                           259.7 MB
▸ [✓] ● Go Build Cache                        845.0 MB
  [✓] ○ Docker                                  2.8 GB
  [✓] ● Homebrew Cache                          1.5 GB
  [ ] ● Chrome Cache                               0 B
─────────────────────────────────────────────────────────
Selected: 5
↑/↓ Move  space Select  s Save  ? Help  q Cancel
```

**2) Preview / Clean**

```
$ mac-cleanup --clean --dry-run

Dry Run Report
--------------
Mode: Dry Run

Summary                             Highlights
Freed (dry-run): 8.6 GB             1. App Caches - 3.2 GB (523 items)
                                    2. Docker - 2.8 GB (12 items)
                                    3. Homebrew Cache - 1.5 GB (34 items)

Details
STATUS  CATEGORY              ITEMS        SIZE
OK      App Caches              523      3.2 GB
OK      Docker                   12      2.8 GB
OK      Homebrew Cache           34      1.5 GB
OK      Go Build Cache           89    845.0 MB
OK      System Logs              67    259.7 MB
```

</details>

## How it works & safety

- Scans known cache/log/temp paths across apps and tools in parallel.
- Lets you preview items and exclude what you want to keep.
- Labels targets by impact level (safe, moderate, risky, manual).
- SIP-protected paths are excluded from scan/cleanup.
- Built-in scans for Homebrew, Docker, and old downloads (brew/docker output or last-modified time filtering).

## Impact levels

- safe: auto-regenerated caches/logs.
- moderate: may require re-download or re-login.
- risky: user data possible; items start excluded.
- manual: no automatic deletion; shows an app guide only.

## Targets (as of v1.3.6)

- Total targets: 107.
- Groups: System 7, Browsers 10, Development 35, Applications 52, Storage 3.
- Cleanup methods: trash 101, permanent 1, builtin 3, manual 2.
- Builtins: homebrew, docker, old-downloads (built-in scanners using brew/docker output or last-modified time filtering).
- Manual: telegram, kakaotalk (no automatic deletion; surfaces large data like chat caches).
- Counts are release-based and may change over time.

## Alternatives

- [mac-cleanup-py](https://github.com/mac-cleanup/mac-cleanup-py) - Python cleanup script for macOS
- [Mole](https://github.com/tw93/Mole) - Deep clean and optimize your Mac

## License

MIT
