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
  <a href="README.md">English</a> | <a href="README_KO.md">한국어</a>
</p>

## Overview

- Preview items before delete and exclude what you want.
- By default, items go to Trash; only the Trash category empties it permanently.
- Risky items start excluded; manual categories show guides only.
- Scope: caches/logs/temp and selected app data (no system optimization or uninstaller).

![demo](assets/result_view.png)


## Quick Start

Install via Homebrew:

```bash
brew install 2ykwang/2ykwang/mac-cleanup-go
```

Run:

```bash
mac-cleanup
mac-cleanup --update   # update via Homebrew
```

> Tip: Grant Full Disk Access to your terminal to clean Trash and restricted locations.  
> System Settings -> Privacy & Security -> Full Disk Access

![demo](assets/demo.gif)

## What it does

- Scans known cache/log/temp paths across apps and tools in parallel.
- Lets you preview items and exclude what you want to keep.
- Labels targets by impact level (safe, moderate, risky).
- Built-in scans for Homebrew, Docker, and old downloads (brew/docker output or last-modified time filtering).

> Note: Risky categories start selected with all items excluded by default. You must
> explicitly include items in the preview page to delete them.

## Impact levels

- safe: auto-regenerated caches/logs.
- moderate: may require re-download or re-login.
- risky: user data possible; items start excluded.
- manual: no automatic deletion; shows an app guide.

## Targets

- Total targets: 107.
- Groups: System 7, Browsers 10, Development 35, Applications 52, Storage 3.
- Cleanup methods: trash 101, permanent 1, builtin 3, manual 2.
- Builtins: homebrew, docker, old-downloads (built-in scanners using brew/docker output or last-modified time filtering).
- Manual: telegram, kakaotalk (no automatic deletion; surfaces large data like chat caches).

## Usage notes

- Full Disk Access helps scan/clean restricted locations.
- Version check: `mac-cleanup --version`.

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

## Alternatives

- [mac-cleanup-py](https://github.com/mac-cleanup/mac-cleanup-py) - Python cleanup script for macOS
- [Mole](https://github.com/tw93/Mole) - Deep clean and optimize your Mac

## License

MIT
