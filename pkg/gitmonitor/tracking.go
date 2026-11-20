// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// maxIncoming limits how many incoming commit subjects are listed.
const maxIncoming = 10

// upstream returns the remote-tracking reference of a local branch and its
// short name ("origin/main"), or nil if the branch tracks nothing that was
// fetched.
func upstream(repo *git.Repository, branch string) (*plumbing.Reference, string) {
	cfg, err := repo.Config()
	if err != nil {
		return nil, ""
	}
	bc := cfg.Branches[branch]
	if bc == nil || bc.Remote == "" || bc.Merge == "" {
		return nil, ""
	}
	short := fmt.Sprintf("%s/%s", bc.Remote, bc.Merge.Short())
	ref, err := repo.Reference(plumbing.ReferenceName("refs/remotes/"+short), true)
	if err != nil {
		return nil, ""
	}
	return ref, short
}

// trackHead fills the repository-level sync fields from the checked-out
// branch: tracking branch, ahead/behind, divergence and incoming commits.
func (s *scanner) trackHead(repo *git.Repository, head *plumbing.Reference, status *RepoStatus) {
	if !head.Name().IsBranch() {
		return
	}
	remote, short := upstream(repo, head.Name().Short())
	if remote == nil {
		return
	}
	status.IsTracking = true
	status.RemoteBranchName = short
	status.Ahead, status.Behind, _ = s.aheadBehind(repo, head.Hash(), remote.Hash())
	status.IsDiverged = status.Ahead > 0 && status.Behind > 0
	status.HasRemoteUpdates = status.Behind > 0
	if status.Behind > 0 {
		status.IncomingCommits = s.incoming(repo, head.Hash(), remote.Hash())
	}
}

// getAheadBehindForBranch calculates how many commits a branch is ahead of
// or behind its remote tracking branch.
func (s *scanner) getAheadBehindForBranch(repo *git.Repository, ref *plumbing.Reference) (int, int, error) {
	remote, _ := upstream(repo, ref.Name().Short())
	if remote == nil {
		return 0, 0, nil
	}
	return s.aheadBehind(repo, ref.Hash(), remote.Hash())
}

// aheadBehind counts the commits local has that remote lacks (ahead) and the
// other way round (behind) — what `git rev-list --count` reports. Comparing
// the two ancestor sets needs no merge base, so it also works after merges
// and in shallow clones.
func (s *scanner) aheadBehind(repo *git.Repository, local, remote plumbing.Hash) (int, int, error) {
	if local == remote {
		return 0, 0, nil
	}
	mine, err := ancestors(repo, local)
	if err != nil {
		return 0, 0, err
	}
	theirs, err := ancestors(repo, remote)
	if err != nil {
		return 0, 0, err
	}
	return countMissing(mine, theirs), countMissing(theirs, mine), nil
}

// countCommitsBetween returns how many commits are reachable from to but
// not from from.
func (s *scanner) countCommitsBetween(repo *git.Repository, from, to plumbing.Hash) (int, error) {
	_, n, err := s.aheadBehind(repo, from, to)
	return n, err
}

// countMissing counts the commits of a that b does not contain.
func countMissing(a, b map[plumbing.Hash]*object.Commit) int {
	n := 0
	for h := range a {
		if _, ok := b[h]; !ok {
			n++
		}
	}
	return n
}

// incoming lists the subjects of up to maxIncoming commits on remote that
// local does not have yet, newest first.
func (s *scanner) incoming(repo *git.Repository, local, remote plumbing.Hash) []string {
	have, err := ancestors(repo, local)
	if err != nil {
		return nil
	}
	theirs, err := ancestors(repo, remote)
	if err != nil {
		return nil
	}
	var missing []*object.Commit
	for h, c := range theirs {
		if _, ok := have[h]; !ok {
			missing = append(missing, c)
		}
	}
	// Newest first; for commits made in the same second, children before
	// their parents, as git log shows them.
	depth := map[plumbing.Hash]int{}
	var depthOf func(c *object.Commit) int
	depthOf = func(c *object.Commit) int {
		if d, ok := depth[c.Hash]; ok {
			return d
		}
		depth[c.Hash] = 0 // guards against cycles in a broken repository
		d := 0
		for _, p := range c.ParentHashes {
			if pc, ok := theirs[p]; ok {
				d = max(d, depthOf(pc)+1)
			}
		}
		depth[c.Hash] = d
		return d
	}
	sort.Slice(missing, func(i, j int) bool {
		a, b := missing[i], missing[j]
		if !a.Committer.When.Equal(b.Committer.When) {
			return a.Committer.When.After(b.Committer.When)
		}
		return depthOf(a) > depthOf(b)
	})
	var subjects []string
	for _, c := range missing[:min(len(missing), maxIncoming)] {
		subject, _, _ := strings.Cut(strings.TrimSpace(c.Message), "\n")
		subjects = append(subjects, subject)
	}
	return subjects
}

// ancestors returns h and every commit reachable from it. Parents missing
// from the repository — the boundary of a shallow clone — end the walk
// there, as they do for git; only a missing h itself is an error.
func ancestors(repo *git.Repository, h plumbing.Hash) (map[plumbing.Hash]*object.Commit, error) {
	seen := map[plumbing.Hash]*object.Commit{}
	stack := []plumbing.Hash{h}
	for len(stack) > 0 {
		next := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := seen[next]; ok {
			continue
		}
		c, err := repo.CommitObject(next)
		if errors.Is(err, plumbing.ErrObjectNotFound) && next != h {
			continue
		}
		if err != nil {
			return nil, err
		}
		seen[next] = c
		stack = append(stack, c.ParentHashes...)
	}
	return seen, nil
}

// gitDirStat looks up a file in the repository's Git directory, which works
// for normal repositories, submodules and worktrees alike.
func gitDirStat(repo *git.Repository, name string) (fs.FileInfo, bool) {
	storage, ok := repo.Storer.(*filesystem.Storage)
	if !ok {
		return nil, false
	}
	info, err := storage.Filesystem().Stat(name)
	if err != nil {
		return nil, false
	}
	return info, true
}
