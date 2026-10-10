# Changelog

All notable changes to GitPulseMLC. Details and reasoning live in the project documentation.

## 1.2.0 — unreleased

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
