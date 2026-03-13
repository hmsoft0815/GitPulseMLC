# GitPulseMLC Manual Page

## NAME
gitpulse - Monitor the heartbeat of local Git repositories

## SYNOPSIS
**gitpulse** [*OPTIONS*]

## DESCRIPTION
**gitpulse** is a high-performance dashboard designed to monitor a large number of local Git repositories simultaneously. It provides a centralized view of your development environment, ensuring you never lose track of uncommitted changes, unpushed branches, or stale worktrees.

The tool operates on a "Read-Only" principle, meaning it does not modify any files or perform network operations unless explicitly configured to do so.

## OPTIONS
*   **-v, --all**: Enable verbose mode. Displays details for all local branches and active stashes.
*   **--compact**: Enable compact mode. Only repositories that require action are displayed.
*   **--progress**: Show a real-time scanning progress indicator on stderr.
*   **--html**: Output results as a standalone, high-contrast HTML report.
*   **--json**: Output results in JSON format.
*   **--no-color**: Disable ANSI color output.
*   **--config PATH**: Use a specific configuration file.
*   **--version**: Print version information and exit.
*   **--help**: Print a summary of options and exit.

## CONFIGURATION
The tool uses an INI-style configuration file (`repos.ini`).

### [projects]
Maps project names to absolute filesystem paths.
Example: `my-tool = /mnt/data2tb/dev/my-tool`

### [general]
*   **replace_path_prefix**: Prefix to be shortened in the output (e.g., `/mnt/data2tb`).
*   **replace_with**: The replacement string (e.g., `@`).

### [settings]
*   **column_name_width**: Minimum width for the project name column (default: 25).
*   **show_summary**: Toggle the final summary block (default: true).
*   **compact_mode**: Enable compact mode by default (default: false).

## FILES
*   `config/repos.ini`: Local configuration.
*   `~/.gitpulsemlc/repos.ini`: Global user configuration.

## AUTHOR
Michael Lechner

## COPYRIGHT
Copyright (c) 2026 Michael Lechner. Licensed under the MIT License.
