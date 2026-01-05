# mac-cleanup-go

[![GitHub Release](https://img.shields.io/github/v/release/2ykwang/mac-cleanup-go)](https://github.com/2ykwang/mac-cleanup-go/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/2ykwang/mac-cleanup-go)](https://goreportcard.com/report/github.com/2ykwang/mac-cleanup-go)
[![codecov](https://codecov.io/gh/2ykwang/mac-cleanup-go/graph/badge.svg)](https://codecov.io/gh/2ykwang/mac-cleanup-go)
[![CI](https://github.com/2ykwang/mac-cleanup-go/actions/workflows/test.yml/badge.svg)](https://github.com/2ykwang/mac-cleanup-go/actions/workflows/test.yml)
[![golangci-lint](https://img.shields.io/badge/linted%20by-golangci--lint-brightgreen)](https://golangci-lint.run/)

A TUI tool for cleaning up macOS caches, logs, and temporary files.

![demo](assets/demo2.gif)

## Features

- **Preview before delete** - See the full list and select only what you want
- **Safety levels** - Safe (auto-regenerated) / Moderate (re-login required) / Risky (important data)
- **90 targets** - 10 browsers, 32 dev tools, 38 apps, and more
- **Trash by default** - Recoverable if you make a mistake

## Installation

```bash
# Homebrew
brew tap 2ykwang/mac-cleanup-go && brew install mac-cleanup-go

# Go install
go install github.com/2ykwang/mac-cleanup-go@latest

# Build from source
git clone https://github.com/2ykwang/mac-cleanup-go.git
cd mac-cleanup-go && make build
```

## Usage

```bash
mac-cleanup          # Run TUI
mac-cleanup -v       # Show version
```

> **Tip**: Grant Full Disk Access to your terminal to clean Trash as well.
> System Settings → Privacy & Security → Full Disk Access

## Alternatives

- [mac-cleanup-py](https://github.com/mac-cleanup/mac-cleanup-py) - Python cleanup script for macOS
- [Mole](https://github.com/tw93/Mole) - Deep clean and optimize your Mac

## License

MIT
