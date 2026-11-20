// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
)

// gitStatusTimeout bounds one `git status`; slow network or WSL drives can
// take a while, a hung mount must not block the scan forever.
const gitStatusTimeout = 5 * time.Minute

// Backend selects how a scanner reads the worktree and branch state.
type Backend string

const (
	// BackendAuto uses git when it is installed, otherwise go-git.
	BackendAuto Backend = "auto"
	// BackendGit uses the installed git: fast, and the result matches
	// `git status` with the user's own git configuration.
	BackendGit Backend = "git"
	// BackendGoGit uses only the built-in go-git library: no dependency on
	// an installed git, but slower, and it differs from git in corners such
	// as repositories nested in a worktree or the global ignore file.
	BackendGoGit Backend = "go-git"
)

// ParseBackend reads a backend name; "" means auto.
func ParseBackend(s string) (Backend, error) {
	switch b := Backend(strings.ToLower(strings.TrimSpace(s))); b {
	case "":
		return BackendAuto, nil
	case BackendAuto, BackendGit, BackendGoGit:
		return b, nil
	}
	return "", fmt.Errorf("unknown backend %q — use auto, git or go-git", s)
}

// GitInstalled reports whether a git executable is on the PATH.
func GitInstalled() bool {
	return gitBinary() != ""
}

// gitBinary is the git executable, looked up once; "" if git is missing.
var gitBinary = sync.OnceValue(func() string {
	p, err := exec.LookPath("git")
	if err != nil {
		return ""
	}
	return p
})

// useGit reports whether this scanner asks git for the repository state.
func (s *scanner) useGit() bool {
	return s.opts.Backend != BackendGoGit && gitBinary() != ""
}

// worktreeStatus returns the worktree status of the repository at dir.
// With useGit it asks git: git uses its index cache, honours
// .gitattributes, core.autocrlf and core.filemode, and is many times faster
// than go-git on large or slow file systems (e.g. /mnt/c in WSL). Otherwise,
// or if git fails, go-git computes the status.
func worktreeStatus(dir string, wt *git.Worktree, useGit, minimalStat bool) (git.Status, error) {
	if dir != "" && useGit {
		if st, err := gitCLIStatus(dir, minimalStat); err == nil {
			return st, nil
		}
	}
	return wt.Status()
}

// gitCLIStatus runs `git status --porcelain -z` read-only: no index
// refresh, no fsmonitor hook from the repository's configuration.
//
// With minimalStat git compares only size and mtime. A checkout shared by
// Windows and WSL never matches the inode/ctime data of an index Windows git
// wrote, and since gitpulse does not rewrite the index, git re-reads every
// file on every scan (solidcraft on /mnt/c: 108 s → 11 s). The price: git
// then trusts the sizes Windows git recorded and does not see line endings a
// Windows tool changed — what Windows git reports, not what a commit from
// WSL would record.
func gitCLIStatus(dir string, minimalStat bool) (git.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitStatusTimeout)
	defer cancel()
	args := []string{"-C", dir, "--no-optional-locks", "-c", "core.fsmonitor=false"}
	if minimalStat {
		args = append(args, "-c", "core.checkStat=minimal", "-c", "core.trustctime=false")
	}
	args = append(args, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=none")
	cmd := exec.CommandContext(ctx, gitBinary(), args...)
	cmd.WaitDelay = 2 * time.Second
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status in %s: %w", dir, err)
	}
	return parsePorcelain(out)
}

// parsePorcelain reads `git status --porcelain=v1 -z` output. Each entry is
// "XY path"; renames and copies are followed by an entry with the old path.
func parsePorcelain(out []byte) (git.Status, error) {
	st := git.Status{}
	entries := bytes.Split(out, []byte{0})
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if len(e) == 0 {
			continue
		}
		if len(e) < 4 || e[2] != ' ' {
			return nil, fmt.Errorf("unexpected git status entry %q", e)
		}
		x, y := statusCode(e[0]), statusCode(e[1])
		st[string(e[3:])] = &git.FileStatus{Staging: x, Worktree: y}
		if x == git.Renamed || x == git.Copied {
			i++ // the old path follows
		}
	}
	return st, nil
}

// statusCode maps a porcelain letter to go-git's status code; git's type
// change ('T') counts as a modification.
func statusCode(c byte) git.StatusCode {
	if c == 'T' {
		return git.Modified
	}
	return git.StatusCode(c)
}
