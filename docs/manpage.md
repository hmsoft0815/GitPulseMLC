# GitPulseMLC Manual Page

## NAME
gitpulse - Monitor the heartbeat of local Git repositories

## SYNOPSIS
**gitpulse** [*OPTIONS*]

**gitpulse add** [*ADD-OPTIONS*] [*PATH* ...] (alias: **init**)

**gitpulse ignore** | **watch** *NAME*|*PATH* ...

**gitpulse mark** *NAME*|*PATH* ... **own**|**3rd** · **gitpulse mark --auto** [**--dry-run**]

**gitpulse remove** *NAME*|*PATH* ... · **gitpulse remove --missing** [**--dry-run**]

**gitpulse list**

## DESCRIPTION
**gitpulse** is a high-performance dashboard designed to monitor a large number of local Git repositories simultaneously. It provides a centralized view of your development environment, ensuring you never lose track of uncommitted changes, unpushed branches, or stale worktrees.

gitpulse only reads your repositories: it never commits, fetches, pulls or pushes. It writes only its own configuration (through the project commands) and a small cache. Its one network request is an anonymous HTTPS check whether a remote is public (see REMOTE VISIBILITY); **--offline** skips it. Third-party projects and remotes on the local network are never checked.

**Note for Windows Users:**
Windows Defender or other antivirus tools might flag the generated binary as suspicious because it is unsigned and performs concurrent file system scans. If execution is blocked, you may need to add an exclusion for the binary or its directory.

## OPTIONS
*   **-v, --all**: Enable verbose mode. Displays details for all remotes, local branches, submodules and active stashes.
*   **--compact**: Enable compact mode. Only repositories that require action are displayed.
*   **--jobs N**: Scan up to N repositories in parallel (default: one per CPU). Lower it on slow disks.
*   **--public**: Only show repositories with a publicly readable remote.
*   **--layout auto|wide|narrow|stacked**: Table layout. **auto** (default) fits the terminal: **wide** shows every column, **narrow** drops PATH and shortens long branch names, **stacked** shows two lines per repository (name and status; branch, remote and path). A long status wraps within its column. When the output is not a terminal it is always wide and never cut. **$COLUMNS** overrides the detected terminal width.
*   **--offline**: Do not check online whether remotes are public; use cached results only.
*   **--progress**: Show a real-time scanning progress indicator on stderr.
*   **--html**: Output results as a standalone, high-contrast HTML report.
*   **--json**: Output results in JSON format.
*   **--no-color**: Disable ANSI color output.
*   **--config PATH**: Use a specific configuration file.
*   **--version**: Print version information and exit.
*   **--help**: Print a summary of options and exit.

## MANAGING PROJECTS
**gitpulse add** adds the repository at each *PATH* (default: the current directory), or every repository below *PATH* if it is a plain folder. If no configuration exists yet, it creates `~/.gitpulsemlc/repos.ini` with default settings. Hidden directories, `node_modules` and `vendor` are skipped, and the search does not descend into a repository it has found. Projects already in the configuration, watched or ignored, are not added again. A repository whose remotes are not listed in `own_remotes` is added as third party; without `own_remotes` everything counts as your own and **add** lists the remote owners it found. **init** is the former name of **add**.

*   **--depth N**: Number of directory levels to search (default 4).
*   **--dry-run**: List what would be added without writing anything.
*   **--own**, **--3rd**: Add as your own or as third party, whatever the remote.

**gitpulse ignore** stops watching projects, given by name or path, and moves them to `[disabled_projects]`; **add** will not add them again. A repository that is not in the configuration yet is recorded as ignored. **gitpulse watch** watches ignored projects again, sorted by `own_remotes`.

**gitpulse mark** *NAME* ... **own**|**3rd** moves projects between `[projects]` and `[third_party]`. Third-party projects are listed in their own block; local changes, commits, stashes and stale branches there are hints, they are not counted as Dirty, Ahead, Behind or Public, and only new upstream commits make them need action. **gitpulse mark --auto** moves own projects whose remotes are not in `own_remotes` to third party; **--dry-run** only lists them.

**gitpulse remove** (alias **rm**) forgets projects: they leave the configuration completely, and nothing on disk is touched. Unlike **ignore**, a later **add** of a parent folder finds them again. **--missing** forgets every project whose folder is gone or holds no Git repository; **--dry-run** only lists them.

**gitpulse list** prints every project with its kind (own, 3rd) and state (watched, ignored, missing, dup). A name that occurs twice is watched only with its last entry; the others are listed as dup.

Flags of these commands may also follow the names and paths. Each accepts **--config PATH**.
*   **--config PATH**: Configuration file to create or extend.

## CONFIGURATION
The tool uses an INI-style configuration file (`repos.ini`), looked up in this order: **--config**, `$GITPULSE_CONFIG`, `config/repos.ini` in the current directory, `~/.gitpulsemlc/repos.ini`. **gitpulse add** creates the last one if none exists.

### [projects]
Your own projects: names mapped to absolute filesystem paths.
Example: `my-tool = /home/me/code/my-tool`

### [third_party]
Other people's repositories, checked out to look at or test; same format. See **gitpulse mark**.

### [disabled_projects]
Ignored projects: not scanned, and **gitpulse add** does not add them again. See **gitpulse ignore**.

Each project name must be unique across all sections; of a name that occurs twice only the last entry is used.

### [general]
*   **replace_path_prefix**: Prefix to be shortened in the output (e.g., `/home/me`).
*   **replace_with**: The replacement string (e.g., `@`).

### [settings]
*   **column_name_width**: Minimum width for the project name column (default: 25).
*   **show_summary**: Toggle the final summary block (default: true).
*   **compact_mode**: Enable compact mode by default (default: false).
*   **layout**: Default for **--layout** (default: auto).
*   **own_remotes**: Comma-separated remotes that are yours, as host (`git.example.com`) or host and owner (`github.com/me`); `*` matches within a segment. Repositories elsewhere are added as third party.
*   **private_hosts**: Comma-separated glob patterns of Git hosts that count as local network and are never checked online.

## REMOTE VISIBILITY
The REMOTE column shows who can read a repository: **pub** (anyone, without credentials), **priv** (external host that refused an anonymous read), **lan** (file path or host on the local network, never checked online), **?** (unknown) or **—** (no remote). With several remotes the most open one is shown. External hosts are checked with an anonymous HTTPS request to `/info/refs?service=git-upload-pack`, without Git configuration or credentials; SSH aliases are resolved with `ssh -G` first. Results are cached for 7 days in the user cache directory (`gitpulse/visibility.json`).

## FILES
*   `~/.gitpulsemlc/repos.ini`: User configuration; created by **gitpulse add**.
*   `config/repos.ini`: Configuration in the current directory; takes precedence over the user configuration.
*   `$XDG_CACHE_HOME/gitpulse/visibility.json` (Linux: `~/.cache/…`, macOS: `~/Library/Caches/…`): Cached visibility results; safe to delete.

## ENVIRONMENT
*   **GITPULSE_CONFIG**: Path of the configuration file, if **--config** is not given.

## EXIT STATUS
*   **0**: Success.
*   **1**: A fatal error, e.g. no configuration, or a project command failed.

## AUTHOR
Michael Lechner

## COPYRIGHT
Copyright (c) 2026 Michael Lechner. Licensed under the MIT License.
