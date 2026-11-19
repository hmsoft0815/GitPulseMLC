# GitPulseMLC Manual Page

## NAME
gitpulse - Monitor the heartbeat of local Git repositories

## SYNOPSIS
**gitpulse** [*OPTIONS*]

**gitpulse init** [*INIT-OPTIONS*] [*DIR* ...]

## DESCRIPTION
**gitpulse** is a high-performance dashboard designed to monitor a large number of local Git repositories simultaneously. It provides a centralized view of your development environment, ensuring you never lose track of uncommitted changes, unpushed branches, or stale worktrees.

The tool operates on a "Read-Only" principle, meaning it does not modify any files or perform network operations unless explicitly configured to do so.

**Note for Windows Users:**
Windows Defender or other antivirus tools might flag the generated binary as suspicious because it is unsigned and performs concurrent file system scans. If execution is blocked, you may need to add an exclusion for the binary or its directory.

## OPTIONS
*   **-v, --all**: Enable verbose mode. Displays details for all remotes, local branches, submodules and active stashes.
*   **--compact**: Enable compact mode. Only repositories that require action are displayed.
*   **--jobs N**: Scan up to N repositories in parallel (default: one per CPU). Lower it on slow disks.
*   **--public**: Only show repositories with a publicly readable remote.
*   **--offline**: Do not check online whether remotes are public; use cached results only.
*   **--progress**: Show a real-time scanning progress indicator on stderr.
*   **--html**: Output results as a standalone, high-contrast HTML report.
*   **--json**: Output results in JSON format.
*   **--no-color**: Disable ANSI color output.
*   **--config PATH**: Use a specific configuration file.
*   **--version**: Print version information and exit.
*   **--help**: Print a summary of options and exit.

## INIT
**gitpulse init** searches each *DIR* (default: the current directory) for Git repositories and adds the new ones to the configuration file. If no configuration exists yet, it creates `~/.gitpulsemlc/repos.ini` with default settings. Hidden directories, `node_modules` and `vendor` are skipped, and the search does not descend into a repository it has found. Projects already listed in `[projects]` or `[disabled_projects]` are not added again, so `init` can be run repeatedly for several folders.

*   **--depth N**: Number of directory levels to search (default 4).
*   **--dry-run**: List what would be added without writing anything.
*   **--config PATH**: Configuration file to create or extend.

## CONFIGURATION
The tool uses an INI-style configuration file (`repos.ini`).

### [projects]
Maps project names to absolute filesystem paths.
Example: `my-tool = /home/me/code/my-tool`

### [general]
*   **replace_path_prefix**: Prefix to be shortened in the output (e.g., `/home/me`).
*   **replace_with**: The replacement string (e.g., `@`).

### [settings]
*   **column_name_width**: Minimum width for the project name column (default: 25).
*   **show_summary**: Toggle the final summary block (default: true).
*   **compact_mode**: Enable compact mode by default (default: false).
*   **private_hosts**: Comma-separated glob patterns of Git hosts that count as local network and are never checked online.

## REMOTE VISIBILITY
The REMOTE column shows who can read a repository: **pub** (anyone, without credentials), **priv** (external host that refused an anonymous read), **lan** (file path or host on the local network, never checked online), **?** (unknown) or **—** (no remote). With several remotes the most open one is shown. External hosts are checked with an anonymous HTTPS request to `/info/refs?service=git-upload-pack`, without Git configuration or credentials; SSH aliases are resolved with `ssh -G` first. Results are cached for 7 days in the user cache directory (`gitpulse/visibility.json`).

## FILES
*   `config/repos.ini`: Local configuration.
*   `~/.gitpulsemlc/repos.ini`: Global user configuration.

## AUTHOR
Michael Lechner

## COPYRIGHT
Copyright (c) 2026 Michael Lechner. Licensed under the MIT License.
