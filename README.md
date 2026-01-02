# mac-cleanup-go

Interactive TUI for cleaning macOS caches, logs, and temporary files.

![demo](assets/demo.gif)

## Features

- **50+ cleanup targets** - System caches, browsers, dev tools, apps
- **Safety levels** - Each item labeled as Safe / Moderate / Risky
- **File-level control** - Preview and exclude individual files before deletion

## Installation

```bash
brew tap 2ykwang/mac-cleanup-go
brew install mac-cleanup-go
```

Or build from source:

```bash
git clone https://github.com/2ykwang/mac-cleanup-go.git
cd mac-cleanup-go && make build
./bin/mac-cleanup
```

## Usage

```bash
mac-cleanup
```

> **Tip**: Grant Full Disk Access to your terminal for complete cleanup access.

## Inspired by

[mac-cleanup-py](https://github.com/mac-cleanup/mac-cleanup-py)

## License

MIT
