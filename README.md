# mac-cleanup-go

A terminal utility to clean up macOS caches, logs, and temporary files.

Inspired by [mac-cleanup-py](https://github.com/mac-cleanup/mac-cleanup-py).

![demo](assets/demo.gif)

## Installation

### Build from Source

```bash
git clone https://github.com/2ykwang/mac-cleanup-go.git
cd mac-cleanup-go
go build -o bin/mac-cleanup ./cmd/mac-cleanup
./bin/mac-cleanup
```

### Homebrew

```bash
brew tap 2ykwang/mac-cleanup-go
brew install mac-cleanup-go
```

## Usage

```bash
mac-cleanup                       # TUI mode (interactive)
mac-cleanup --dangerously-delete  # Permanent deletion
mac-cleanup --version             # Show version
```

### CLI Mode

After running TUI mode once, your selection is saved. Use `--clean` for quick cleanup:

```bash
mac-cleanup --clean               # Clean with saved profile
mac-cleanup --clean --dry-run     # Preview only (no deletion)
mac-cleanup --clean --dangerously-delete  # Permanent deletion
```

> **Full Disk Access (Optional)**: Some items (Trash, Mail, etc.) require Full Disk Access.
> System Settings → Privacy & Security → Full Disk Access → Add your terminal app

## How It Works

1. **Scan** - Searches for cleanable files in system, browsers, dev tools, and apps
2. **Select** - Choose items to delete (sorted by size, safety levels displayed)
3. **Preview** - Review detailed file lists and exclude individual items (exclusions are remembered)
4. **Clean** - Move to Trash (default) or permanently delete with `--dangerously-delete`

### Safety Levels

| Level    | Description                              |
|----------|------------------------------------------|
| Safe     | Auto-regenerated caches                  |
| Moderate | May require re-download or re-login      |
| Risky    | May contain important data - review first |

## Keyboard Shortcuts

| Screen  | Keys |
|---------|------|
| Main    | `↑↓` Navigate / `Space` Select / `a` Select All / `d` Deselect All / `Enter` Preview |
| Preview | `←→` Switch tabs / `Space` Exclude item / `a` Include All / `d` Exclude All / `y` Delete |

## Cleanup Targets

- **System**: Trash, App Caches, Logs, QuickLook
- **Browsers**: Chrome, Safari, Firefox, Arc, Edge, Brave
- **Dev Tools**: Xcode, npm, Yarn, pip, Docker, Homebrew, Go, Gradle, etc.
- **Apps**: Discord, Slack, Spotify, VS Code, JetBrains, etc.

## License

MIT