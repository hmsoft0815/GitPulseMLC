# Changelog

All notable changes to GitPulseMLC. Details and reasoning live in the project documentation.

## Unreleased

### Added
- `backend = auto | git | go-git` (and `--backend`) chooses how repositories are read; `go-git` never calls git.
- `--progress` shows which hosts from `[hosts]` are still running and for how long, instead of a silent wait.

### Fixed
- Without git, ahead/behind and incoming commits were 0 in shallow clones.

## 1.6.0 — 2026-11-19

### Added
- `[hosts]` in the config adds other machines (servers, WSL on Windows) to the overview, each as its own block.
- `EOL (n)` marks files that differ only in line endings, before a commit would rewrite every line.
- `stat_check = minimal` makes scans of checkouts shared with Windows git about ten times faster.

### Changed
- gitpulse asks git for the worktree and branch state when git is installed: about ten times faster, up to 40 times on slow drives.
- The status now honours `.gitattributes`, `core.autocrlf` and `core.filemode` exactly like `git status`.

### Fixed
- Behind counts (`↓n`, `behind`, `incoming_commits`) were too low after merges upstream.
- The `install.sh` hint points to `gitpulse add` instead of `gitpulse init`.

## 1.5.0 — 2026-11-19

### Added
- The terminal table fits narrow terminals: without PATH, then two lines per repository; `--layout` and `layout =` choose one.

### Changed
- The hint after `gitpulse add` and the third-party heading wrap to fit an 80-column terminal.

### Fixed
- `gitpulse -v` no longer draws the stash line as the last branch of the tree.
- Commits younger than an hour show `12m ago` or `just now` instead of `0h ago`.

## 1.4.0 — 2026-11-19

### Added
- Submodules are scanned: `SUB <path>` marks one with work of its own, `--all` lists each with its pointer state.
- A REMOTE column shows whether a repository is public, private, on the local network or has no remote.
- `--public` lists only publicly readable repositories; `--offline` skips the online check and uses cached results.
- `private_hosts` in the config marks Git hosts that are private although they look external.
- `gitpulse add`, `ignore`, `watch`, `mark`, `remove` and `list` manage projects without editing the config by hand.
- `gitpulse remove --missing` forgets every project whose folder is gone or holds no repository.
- Projects can be marked as third party; their local work is shown as hints in a separate block.
- `task install` builds gitpulse and installs it with its man page to `~/.local` without root; `task uninstall` removes it.
- `own_remotes` sorts new repositories into own and third party; `gitpulse mark --auto` sorts existing ones.

### Changed
- `gitpulse init` is now `gitpulse add` and also adds a single repository; `init` still works.
- The `(No Remote)` marker moved from the status column to the REMOTE column (`—`).
- A submodule whose only change is a newer commit than the parent records shows as `sub-ptr`, not DIRTY.
- GitHub releases no longer carry macOS archives; `install.sh` fetches the signed macOS binaries from mlcgo.eu.

### Fixed
- JSON: `ahead`, `behind`, `is_tracking`, `is_diverged`, `incoming_commits` and the other sync fields of the checked-out branch are filled; they were always empty.
- JSON: lists are `[]` instead of `null`, so the output matches `docs/schema.json`; the never-filled `main_language` is gone.
- The last fetch time and an interrupted merge are found in submodules and worktrees too.
- A project name used twice in the config is no longer dropped when gitpulse writes the config; gitpulse warns about it.

## 1.3.0 — 2026-10-11

### Added
- Repositories are scanned in parallel: 73 repositories take 4 s instead of 33 s; `--jobs N` limits it.
- The macOS downloads on mlcgo.eu are signed with a Developer ID and notarized by Apple.

### Fixed
- The HTML report honours `--compact` and shows the same status column as the terminal.
- The HTML report escapes repository names, branches and paths and uses inline styles for mail clients.

## 1.2.0 — 2026-10-10

### Added
- `gitpulse init [DIR]` finds Git repositories below a folder and writes or extends the config.
- First run without a config explains what gitpulse needs and how to start, instead of a bare error.
- Release builds now include deb, rpm and apk packages and an `install.sh` for Linux and macOS.
- `VERSION` file is the single source of the version number.

### Changed
- The status column shows unpushed (`↑2`), unpulled (`↓1`), stashed and stale work instead of plain CLEAN.
- The binary is built from `cmd/gitpulse` and is named `gitpulse` everywhere (was `tui`).
- Module path is now `github.com/hmsoft0815/GitPulseMLC`, so `go install` and `go get` work.
- Requires Go 1.27.2; go-git, go-billy and other dependencies are updated.
- The Makefile is removed; all tasks run through `Taskfile.yml`.

### Fixed
- Eleven known vulnerabilities in go-git, go-billy, circl and the Go standard library are fixed by the updates.
- The compact-mode summary no longer pushes the "Dirty" line far to the right.
- CI and release workflows run on `main` again and use GoReleaser v2.

## 1.1.0 — 2026-03-13

- First public release with TUI, JSON and HTML output, compact mode and stale branch detection.
