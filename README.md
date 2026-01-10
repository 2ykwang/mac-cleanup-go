<div align="center">
  <h1>mac-cleanup-go</h1>
  <p>A TUI tool for cleaning up macOS caches, logs, and temporary files.</p>
</div>

<p align="center">
  <a href="https://github.com/2ykwang/mac-cleanup-go/releases"><img src="https://img.shields.io/github/v/release/2ykwang/mac-cleanup-go" alt="GitHub Release"></a>
  <a href="https://goreportcard.com/report/github.com/2ykwang/mac-cleanup-go"><img src="https://goreportcard.com/badge/github.com/2ykwang/mac-cleanup-go" alt="Go Report Card"></a>
  <a href="https://github.com/2ykwang/mac-cleanup-go/actions/workflows/test.yml"><img src="https://github.com/2ykwang/mac-cleanup-go/actions/workflows/test.yml/badge.svg" alt="CI"></a>
  <a href="https://golangci-lint.run/"><img src="https://img.shields.io/badge/linted%20by-golangci--lint-brightgreen" alt="golangci-lint"></a>
</p>

![demo](assets/demo2.gif)

## Features

- **Preview before delete** - See the full list and select only what you want
- **Safety levels** - Safe (auto-regenerated) / Moderate (re-login required) / Risky (important data)
- **90 targets** - 10 browsers, 32 dev tools, 38 apps, and more
- **Trash by default** - Recoverable if you make a mistake

## Installation

```bash
# Homebrew
brew tap 2ykwang/2ykwang && brew install mac-cleanup-go
```

## Usage

```bash
mac-cleanup
```

> **Tip**: Grant Full Disk Access to your terminal to clean Trash as well.
> System Settings → Privacy & Security → Full Disk Access

## Alternatives

- [mac-cleanup-py](https://github.com/mac-cleanup/mac-cleanup-py) - Python cleanup script for macOS
- [Mole](https://github.com/tw93/Mole) - Deep clean and optimize your Mac

## License

MIT
