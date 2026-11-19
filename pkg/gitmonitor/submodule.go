// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"errors"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// SubmoduleInfo describes one submodule as seen from its parent repository.
//
// A submodule is recorded in the parent as a single commit hash (the gitlink).
// When work is committed inside the submodule, its HEAD moves on while the
// parent still records the old commit: git reports the submodule path as
// modified although nothing is lost. PointerAhead/PointerBehind say how far
// the checkout has moved; Status holds the submodule's own scan.
type SubmoduleInfo struct {
	Path           string      `json:"path"`                      // Path relative to the parent's worktree.
	RecordedCommit string      `json:"recorded_commit"`           // Commit the parent's index records.
	CurrentCommit  string      `json:"current_commit,omitempty"`  // HEAD of the checked-out submodule.
	PointerAhead   int         `json:"pointer_ahead"`             // Submodule HEAD is this many commits past the recorded commit.
	PointerBehind  int         `json:"pointer_behind"`            // Recorded commit is this many commits past the submodule HEAD.
	PointerStaged  bool        `json:"pointer_staged"`            // A new pointer is staged in the parent but not committed.
	NotInitialized bool        `json:"not_initialized,omitempty"` // Declared in .gitmodules but not checked out.
	Status         *RepoStatus `json:"status,omitempty"`          // The submodule's own working tree and branch state.
}

// PointerChanged reports whether the submodule's HEAD differs from the commit
// its parent records.
func (s SubmoduleInfo) PointerChanged() bool {
	return s.CurrentCommit != "" && s.CurrentCommit != s.RecordedCommit
}

// NeedsAction reports whether the submodule itself holds work that is not
// safe yet: uncommitted changes, unpushed or unpulled commits, stashes, or a
// scan error. A mere pointer change in the parent does not count.
func (s SubmoduleInfo) NeedsAction() bool {
	st := s.Status
	if st == nil {
		return false
	}
	if st.ErrorMsg != "" || !st.IsClean || st.StashCount > 0 {
		return true
	}
	for _, b := range st.LocalBranches {
		if b.Ahead > 0 || b.Behind > 0 {
			return true
		}
	}
	for _, sub := range st.Submodules {
		if sub.NeedsAction() {
			return true
		}
	}
	return false
}

// collectSubmodules scans every submodule declared in the worktree's
// .gitmodules. Repositories without submodules return nil.
func (s *scanner) collectSubmodules(parent *RepoStatus, wt *git.Worktree) []SubmoduleInfo {
	subs, err := wt.Submodules()
	if err != nil || len(subs) == 0 {
		return nil
	}
	infos := make([]SubmoduleInfo, 0, len(subs))
	for _, sm := range subs {
		infos = append(infos, s.scanSubmodule(parent, sm))
	}
	return infos
}

// scanSubmodule fills one SubmoduleInfo; errors end up in its Status.
func (s *scanner) scanSubmodule(parent *RepoStatus, sm *git.Submodule) SubmoduleInfo {
	path := sm.Config().Path
	info := SubmoduleInfo{Path: path}

	smStatus, err := sm.Status()
	if err != nil {
		info.Status = errorStatus(path, parent.Path, err)
		return info
	}
	info.RecordedCommit = hashString(smStatus.Expected)
	info.CurrentCommit = hashString(smStatus.Current)

	repo, err := sm.Repository()
	if errors.Is(err, git.ErrSubmoduleNotInitialized) {
		info.NotInitialized = true
		return info
	}
	if err != nil {
		info.Status = errorStatus(path, parent.Path, err)
		return info
	}

	if info.PointerChanged() && !smStatus.Expected.IsZero() {
		info.PointerAhead, info.PointerBehind, _ = s.aheadBehind(repo, smStatus.Current, smStatus.Expected)
	}

	sub := &RepoStatus{Name: path, ChangedFiles: make(map[string]int)}
	if parent.Path != "" {
		sub.Path = filepath.Join(parent.Path, path)
	}
	info.Status, _ = s.scanRepo(sub, repo)
	return info
}

// errorStatus builds a RepoStatus that only carries an error.
func errorStatus(name, parentPath string, err error) *RepoStatus {
	st := &RepoStatus{Name: name, Error: err, ErrorMsg: err.Error()}
	if parentPath != "" {
		st.Path = filepath.Join(parentPath, name)
	}
	return st
}

func hashString(h plumbing.Hash) string {
	if h.IsZero() {
		return ""
	}
	return h.String()
}

// SubmoduleAttention splits the repository's submodules into those that need
// action (their paths) and the number whose only difference is a pointer the
// parent has not committed yet. A staged pointer counts as a change of the
// parent itself and appears in neither.
func (r *RepoStatus) SubmoduleAttention() (needsAction []string, pointerOnly int) {
	for _, sub := range r.Submodules {
		switch {
		case sub.NeedsAction():
			needsAction = append(needsAction, sub.Path)
		case sub.PointerChanged() && !sub.PointerStaged:
			pointerOnly++
		}
	}
	return needsAction, pointerOnly
}
