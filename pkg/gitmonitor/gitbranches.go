// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// gitBranch is one local branch as `git for-each-ref` reports it.
type gitBranch struct {
	info     BranchInfo
	upstream string // e.g. "origin/main"; "" without a fetched upstream
}

// gitOutput runs a read-only git command in dir and returns its output.
func gitOutput(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitStatusTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBinary(), append([]string{"-C", dir, "--no-optional-locks"}, args...)...)
	cmd.WaitDelay = 2 * time.Second
	cmd.Env = append(cmd.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s in %s: %w", args[0], dir, err)
	}
	return out, nil
}

// gitBranches lists the local branches with sync state and age in one call;
// go-git would walk the history once per branch.
func gitBranches(dir string) ([]gitBranch, error) {
	out, err := gitOutput(dir, "for-each-ref",
		"--format=%(refname:short)%00%(upstream:short)%00%(upstream:track,nobracket)%00%(committerdate:unix)%00%(HEAD)",
		"refs/heads")
	if err != nil {
		return nil, err
	}
	var branches []gitBranch
	for _, line := range bytes.Split(bytes.TrimRight(out, "\n"), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		f := strings.Split(string(line), "\x00")
		if len(f) != 5 {
			return nil, fmt.Errorf("unexpected for-each-ref line %q", line)
		}
		b := gitBranch{info: BranchInfo{Name: f[0], IsHead: f[4] == "*"}, upstream: f[1]}
		if f[2] == "gone" {
			b.upstream = ""
		}
		b.info.Ahead, b.info.Behind = parseTrack(f[2])
		if ts, err := strconv.ParseInt(f[3], 10, 64); err == nil {
			b.info.LastCommitAt = time.Unix(ts, 0)
		}
		b.info.IsStale = isStale(b.info)
		branches = append(branches, b)
	}
	return branches, nil
}

// parseTrack reads "ahead 2, behind 1" (either part optional).
func parseTrack(track string) (ahead, behind int) {
	for _, part := range strings.Split(track, ", ") {
		if n, ok := strings.CutPrefix(part, "ahead "); ok {
			ahead, _ = strconv.Atoi(n)
		}
		if n, ok := strings.CutPrefix(part, "behind "); ok {
			behind, _ = strconv.Atoi(n)
		}
	}
	return ahead, behind
}

// isStale: not the checked-out branch, not main/master, no commit for 30 days.
func isStale(b BranchInfo) bool {
	return !b.IsHead && b.Name != "main" && b.Name != "master" &&
		!b.LastCommitAt.IsZero() && time.Since(b.LastCommitAt) > 30*24*time.Hour
}

// gitTrackHead fills the repository-level sync fields from the checked-out
// branch, like trackHead does with go-git.
func gitTrackHead(dir string, branches []gitBranch, status *RepoStatus) {
	for _, b := range branches {
		if !b.info.IsHead || b.upstream == "" {
			continue
		}
		status.IsTracking = true
		status.RemoteBranchName = b.upstream
		status.Ahead, status.Behind = b.info.Ahead, b.info.Behind
		status.IsDiverged = status.Ahead > 0 && status.Behind > 0
		status.HasRemoteUpdates = status.Behind > 0
		if status.Behind == 0 {
			return
		}
		out, err := gitOutput(dir, "log", "--format=%s", "-n", strconv.Itoa(maxIncoming), "HEAD..@{upstream}")
		if err != nil {
			return
		}
		status.IncomingCommits = strings.Split(strings.TrimRight(string(out), "\n"), "\n")
		return
	}
}
