// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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
		status.Missing = true
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
		status.Missing = errors.Is(err, git.ErrRepositoryNotExists)
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
	status.Remotes = collectRemotes(repo)
	if len(status.Remotes) == 0 {
		status.HasNoRemote = true
	} else {
		status.RemoteURL = status.Remotes[0].URL
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

	// Submodules: a changed pointer alone is reported per submodule, not as
	// a modified file of the parent (see SubmoduleInfo).
	status.Submodules = s.collectSubmodules(status, wt)
	countWorktreeChanges(status, wtStatus)

	// 4. Last fetch and an interrupted merge, read from the Git directory
	//    (".git" may be a file in submodules and worktrees)
	if info, ok := gitDirStat(repo, "FETCH_HEAD"); ok {
		status.LastFetchAt = info.ModTime()
	}
	_, status.InMergeConflict = gitDirStat(repo, "MERGE_HEAD")

	// 5. Collect details for all local branches
	status.LocalBranches, _ = s.collectBranchDetails(repo)
	s.trackHead(repo, head, status)

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

// countWorktreeChanges sets IsClean and the change counters from the
// worktree status. Paths of submodules count only when a new pointer is
// staged; an unstaged pointer change is reported in Submodules instead.
func countWorktreeChanges(status *RepoStatus, wtStatus git.Status) {
	subs := make(map[string]*SubmoduleInfo, len(status.Submodules))
	for i := range status.Submodules {
		subs[status.Submodules[i].Path] = &status.Submodules[i]
	}

	status.IsClean = true
	for file, fs := range wtStatus {
		if sub, ok := subs[file]; ok {
			if fs.Staging == git.Unmodified {
				continue
			}
			sub.PointerStaged = true
			status.IsClean = false
			status.ModifiedCount++
			status.ChangedFiles["(submodule)"]++
			continue
		}
		if fs.Staging == git.Unmodified && fs.Worktree == git.Unmodified {
			continue
		}
		status.IsClean = false
		status.ModifiedCount++
		ext := filepath.Ext(file)
		if ext == "" {
			ext = "(no ext)"
		}
		status.ChangedFiles[ext]++
		if fs.Worktree == git.Untracked {
			status.UntrackedCount++
		}
		if fs.Worktree == git.Deleted || fs.Staging == git.Deleted {
			status.DeletedCount++
		}
	}
}

// Gone explains why path no longer holds a Git repository — "path does not
// exist" or "no Git repository" — or returns "" if it still does.
func Gone(path string) string {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "path does not exist"
	}
	if _, err := git.PlainOpen(path); errors.Is(err, git.ErrRepositoryNotExists) {
		return "no Git repository"
	}
	return ""
}
