# GitPulseMLC

**Monitor the heartbeat of your local Git repositories.**

Keep track of all your projects – fast, safe, and directly in your terminal.

---

## Overview

GitPulseMLC is a high-performance Go-based dashboard designed to monitor a large number of local Git repositories. It provides a centralized view of your development environment, ensuring you never lose track of uncommitted changes or unpushed branches across multiple projects.

### Key Features
*   **Concurrent Scanning**: Leverages Go routines to scan hundreds of directories in a heartbeat.
*   **Passive & Safe**: Operates on a "Read-Only" principle. No file modifications or merge risks.
*   **Deep Insights**: Tracks dirty worktrees with **file-type summaries** (e.g., `+3 .go`), ahead/behind counts, and branch staleness.
*   **Visual Dashboard**: Uses Lip Gloss for a modern, color-coded terminal output with dynamic alignment.
*   **Multiple Outputs**: Supports TUI, **JSON** data export, and **high-contrast HTML** reports suitable for automated emails.
*   **Compact Mode**: Option to hide clean repositories and focus only on those requiring action.
*   **Stale Branch Detection**: Automatically flags local branches that haven't been touched in over 30 days.
*   **Path Shortening**: Configurable path replacement to keep the dashboard tidy even with deep directory structures.
*   **Flexible Config**: Intelligent configuration lookup (local, home directory, or custom path).

---

## 📦 Go Library Usage

GitPulseMLC is built as a modular Go library. You can use the scanning engine in your own Go projects:

```bash
go get github.com/hmsoft0815/GitPulseMLC
```

For detailed integration examples, see [Go Library Documentation](docs/LIBRARY.md).

---

## Installation & Usage

### Prerequisites
*   Go 1.25+
*   Git (installed and configured)

### Setup
1.  Clone the repository.
2.  Build the binary:
    ```bash
    go build -o gitpulse ./cmd/tui
    ```
3.  Create your `config/repos.ini` (see `config/repos.ini.example`):
    ```ini
    [projects]
    my-tool = /mnt/data2tb/dev/my-tool
    web-app = /mnt/data2tb/dev/another/repo

    [general]
    replace_path_prefix = /mnt/data2tb
    replace_with = "@"

    [settings]
    column_name_width = 30
    show_summary = true
    compact_mode = false
    ```

### Running
```bash
./gitpulse          # Standard view
./gitpulse -v       # Verbose mode: lists all local branches, sync status, and stashes
./gitpulse --progress # Show scanning progress (useful for large repository lists)
./gitpulse --compact # Only show repositories that need attention (Dirty, Ahead, etc.)
./gitpulse --html > report.html # Generate a high-contrast HTML report
./gitpulse --json > data.json   # Export status data as JSON (See docs/json_schema.md)
./gitpulse --config /path/to/my-repos.ini # Use a specific config file
./gitpulse --version # Show version info
./gitpulse --help    # Show all available flags
```

### Config Lookup Order
1.  `--config` flag
2.  `GITPULSE_CONFIG` environment variable
3.  `config/repos.ini` (current directory)
4.  `~/.gitpulsemlc/repos.ini` (user home)

---

## Motivation & Credits

### Why GitPulseMLC?
Managing 20+ active repositories on a local server or dev machine often leads to "forgotten" commits or outdated local branches. Existing GUI tools are often too heavy or require manual clicking. GitPulseMLC was built to provide a "Single Source of Truth" that is as fast as `ls` but as informative as `git status`.

### Credits
*   **Engine**: Built with the excellent [go-git](https://github.com/go-git/go-git) library.
*   **UI**: Styled with [Lip Gloss](https://github.com/charmbracelet/lipgloss).
*   **Inspiration**: Developed as a practical tool for high-performance development environments.

---

## ⚖️ License

Copyright (c) 2026 Michael Lechner.
This project is licensed under the MIT License - see the `LICENSE` file for details.
