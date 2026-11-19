// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// maxIncoming limits how many incoming commit subjects are listed.
const maxIncoming = 10

// errStop ends a commit walk early.
var errStop = errors.New("stop")

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

// mergeBase returns the first common ancestor of a and b.
func mergeBase(repo *git.Repository, a, b plumbing.Hash) (plumbing.Hash, error) {
	ca, err := repo.CommitObject(a)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	cb, err := repo.CommitObject(b)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	bases, err := ca.MergeBase(cb)
	if err != nil || len(bases) == 0 {
		return plumbing.ZeroHash, fmt.Errorf("no merge base: %w", err)
	}
	return bases[0].Hash, nil
}

// aheadBehind counts the commits local has that remote lacks (ahead) and the
// other way round (behind), measured from their merge base.
func (s *scanner) aheadBehind(repo *git.Repository, local, remote plumbing.Hash) (int, int, error) {
	base, err := mergeBase(repo, local, remote)
	if err != nil {
		return 0, 0, nil
	}
	ahead, _ := s.countCommitsBetween(repo, base, local)
	behind, _ := s.countCommitsBetween(repo, base, remote)
	return ahead, behind, nil
}

// incoming lists the subjects of up to maxIncoming commits on remote that
// local does not have yet, newest first.
func (s *scanner) incoming(repo *git.Repository, local, remote plumbing.Hash) []string {
	base, err := mergeBase(repo, local, remote)
	if err != nil {
		return nil
	}
	iter, err := repo.Log(&git.LogOptions{From: remote})
	if err != nil {
		return nil
	}
	defer iter.Close()
	var subjects []string
	_ = iter.ForEach(func(c *object.Commit) error {
		if c.Hash == base || len(subjects) == maxIncoming {
			return errStop
		}
		subject, _, _ := strings.Cut(strings.TrimSpace(c.Message), "\n")
		subjects = append(subjects, subject)
		return nil
	})
	return subjects
}

// countCommitsBetween returns the number of commits from the 'from' hash to
// the 'to' hash.
func (s *scanner) countCommitsBetween(repo *git.Repository, from, to plumbing.Hash) (int, error) {
	if from == to {
		return 0, nil
	}
	iter, err := repo.Log(&git.LogOptions{From: to})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	count := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if c.Hash == from {
			return errStop
		}
		count++
		return nil
	})
	if err != nil && !errors.Is(err, errStop) {
		return count, err
	}
	return count, nil
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
