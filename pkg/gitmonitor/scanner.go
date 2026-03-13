// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitScanner is the interface for gathering information about Git repositories.
type GitScanner interface {
	// Scan takes a project name and an absolute path, and returns the repository status.
	Scan(name, path string) (*RepoStatus, error)
}

type scanner struct{}

// NewScanner creates and returns a new instance of a GitScanner.
func NewScanner() GitScanner {
	return &scanner{}
}

// Scan performs a comprehensive analysis of a Git repository at the given path.
// It checks for existence, permissions, dirty state, branch sync status, and stashes.
func (s *scanner) Scan(name, path string) (*RepoStatus, error) {
	status := &RepoStatus{
		Name:         name,
		Path:         path,
		ChangedFiles: make(map[string]int),
	}

	// 0. Preliminary Check: Does the path exist and is it accessible?
	if _, err := os.Stat(path); os.IsNotExist(err) {
		status.Error = fmt.Errorf("path does not exist: %s", path)
		status.ErrorMsg = status.Error.Error()
		return status, nil
	} else if err != nil {
		status.Error = fmt.Errorf("error accessing path: %w", err)
		status.ErrorMsg = status.Error.Error()
		return status, nil
	}

	// Check read permissions
	if _, err := os.ReadDir(path); os.IsPermission(err) {
		status.Error = fmt.Errorf("permission denied")
		status.ErrorMsg = status.Error.Error()
		return status, nil
	}

	// 1. Open the repository
	repo, err := git.PlainOpen(path)
	if err != nil {
		status.Error = fmt.Errorf("failed to open repo: %w", err)
		status.ErrorMsg = status.Error.Error()
		return status, nil
	}

	return s.scanRepo(status, repo)
}

// scanRepo carries out the analysis on an already opened repository.
// This allowing for easier unit testing with in-memory repos.
func (s *scanner) scanRepo(status *RepoStatus, repo *git.Repository) (*RepoStatus, error) {
	if status.ChangedFiles == nil {
		status.ChangedFiles = make(map[string]int)
	}
	// 1.1 Check if any remotes are configured
	remotes, err := repo.Remotes()
	if err == nil && len(remotes) == 0 {
		status.HasNoRemote = true
	}

	// 2. Get HEAD and current branch information
	head, err := repo.Head()
	if err != nil {
		status.Error = fmt.Errorf("failed to get head: %w", err)
		status.ErrorMsg = status.Error.Error()
		return status, nil
	}

	status.CurrentBranch = head.Name().Short()
	status.LastCommitID = head.Hash().String()

	// Retrieve details of the last commit
	commit, err := repo.CommitObject(head.Hash())
	if err == nil {
		status.LastCommitMsg = commit.Message
		status.LastCommitAt = commit.Author.When
	}

	// 3. Perform a "Dirty Check" on the worktree
	wt, err := repo.Worktree()
	if err != nil {
		status.Error = fmt.Errorf("failed to get worktree: %w", err)
		status.ErrorMsg = status.Error.Error()
		return status, nil
	}

	wtStatus, err := wt.Status()
	if err != nil {
		status.Error = fmt.Errorf("failed to get status: %w", err)
		status.ErrorMsg = status.Error.Error()
		return status, nil
	}

	status.IsClean = wtStatus.IsClean()
	for file, s := range wtStatus {
		if s.Staging != git.Unmodified || s.Worktree != git.Unmodified {
			status.ModifiedCount++
			ext := filepath.Ext(file)
			if ext == "" {
				ext = "(no ext)"
			}
			status.ChangedFiles[ext]++
		}
		if s.Worktree == git.Untracked {
			status.UntrackedCount++
		}
		if s.Worktree == git.Deleted || s.Staging == git.Deleted {
			status.DeletedCount++
		}
	}

	// 4. Retrieve last fetch time from .git/FETCH_HEAD
	if status.Path != "" {
		fetchHeadPath := filepath.Join(status.Path, ".git", "FETCH_HEAD")
		if info, err := os.Stat(fetchHeadPath); err == nil {
			status.LastFetchAt = info.ModTime()
		}
	}

	// 5. Collect details for all local branches
	status.LocalBranches, _ = s.collectBranchDetails(repo)

	// 6. Count active stashes
	status.StashCount = s.countStashes(repo)

	return status, nil
}

// collectBranchDetails iterates through all local branches and gathers their sync status.
func (s *scanner) collectBranchDetails(repo *git.Repository) ([]BranchInfo, error) {
	iter, err := repo.Branches()
	if err != nil {
		return nil, err
	}
	var infos []BranchInfo

	head, _ := repo.Head()

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		isHead := head != nil && ref.Hash() == head.Hash() && ref.Name() == head.Name()

		ahead, behind, _ := s.getAheadBehindForBranch(repo, ref)

		// Get the last commit timestamp for this specific branch
		var lastCommitAt time.Time
		commit, err := repo.CommitObject(ref.Hash())
		if err == nil {
			lastCommitAt = commit.Author.When
		}

		// Check if the branch is "stale" (> 30 days old and not a primary branch)
		isStale := false
		if !isHead && name != "main" && name != "master" && !lastCommitAt.IsZero() && time.Since(lastCommitAt).Hours() > 24*30 {
			isStale = true
		}

		infos = append(infos, BranchInfo{
			Name:         name,
			IsHead:       isHead,
			Ahead:        ahead,
			Behind:       behind,
			LastCommitAt: lastCommitAt,
			IsStale:      isStale,
		})
		return nil
	})
	return infos, err
}

// getAheadBehindForBranch calculates how many commits a branch is ahead of or behind its remote tracking branch.
func (s *scanner) getAheadBehindForBranch(repo *git.Repository, ref *plumbing.Reference) (int, int, error) {
	localCommit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return 0, 0, err
	}

	cfg, err := repo.Config()
	if err != nil {
		return 0, 0, err
	}

	// Identify the remote tracking branch
	branchCfg := cfg.Branches[ref.Name().Short()]
	if branchCfg == nil || branchCfg.Remote == "" || branchCfg.Merge == "" {
		return 0, 0, nil
	}

	remoteRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", branchCfg.Remote, branchCfg.Merge.Short()))
	remoteRef, err := repo.Reference(remoteRefName, true)
	if err != nil {
		return 0, 0, nil
	}

	remoteCommit, err := repo.CommitObject(remoteRef.Hash())
	if err != nil {
		return 0, 0, err
	}

	// Find the common ancestor (merge base)
	bases, err := localCommit.MergeBase(remoteCommit)
	if err != nil || len(bases) == 0 {
		return 0, 0, nil
	}
	base := bases[0]

	ahead, _ := s.countCommitsBetween(repo, base.Hash, localCommit.Hash)
	behind, _ := s.countCommitsBetween(repo, base.Hash, remoteCommit.Hash)

	return ahead, behind, nil
}

// countCommitsBetween returns the number of commits from the 'from' hash to the 'to' hash.
func (s *scanner) countCommitsBetween(repo *git.Repository, from, to plumbing.Hash) (int, error) {
	if from == to {
		return 0, nil
	}

	cIter, err := repo.Log(&git.LogOptions{From: to})
	if err != nil {
		return 0, err
	}
	defer cIter.Close()

	count := 0
	err = cIter.ForEach(func(c *object.Commit) error {
		// fmt.Printf("Debug: visit %s (looking for %s)\n", c.Hash, from)
		if c.Hash == from {
			return fmt.Errorf("reached from")
		}
		count++
		return nil
	})

	if err != nil && err.Error() == "reached from" {
		return count, nil
	}

	return count, nil
}

// countStashes returns the number of stashed changes by checking the refs/stash reference.
func (s *scanner) countStashes(repo *git.Repository) int {
	refName := plumbing.ReferenceName("refs/stash")
	_, err := repo.Reference(refName, true)
	if err != nil {
		return 0
	}

	// Currently returns 1 if any stashes exist. 
	// Full counting would require low-level reflog traversal.
	return 1
}
