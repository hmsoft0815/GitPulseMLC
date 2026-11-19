// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
)

// gitStatusTimeout bounds one `git status`; slow network or WSL drives can
// take a while, a hung mount must not block the scan forever.
const gitStatusTimeout = 5 * time.Minute

// gitBinary is the git executable, looked up once; "" if git is missing or
// GITPULSE_NO_GIT is set.
var gitBinary = sync.OnceValue(func() string {
	if os.Getenv("GITPULSE_NO_GIT") != "" {
		return ""
	}
	p, err := exec.LookPath("git")
	if err != nil {
		return ""
	}
	return p
})

// worktreeStatus returns the worktree status of the repository at dir.
// It asks git when it is installed: git uses its index cache, honours
// .gitattributes, core.autocrlf and core.filemode, and is many times faster
// than go-git on large or slow file systems (e.g. /mnt/c in WSL). Without
// git, or if git fails, go-git computes the status.
func worktreeStatus(dir string, wt *git.Worktree, minimalStat bool) (git.Status, error) {
	if dir != "" && gitBinary() != "" {
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
