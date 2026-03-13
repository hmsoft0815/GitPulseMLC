// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import "time"

// Summary holds aggregate statistics for multiple repositories.
type Summary struct {
	Total  int `json:"total"`
	Dirty  int `json:"dirty"`
	Ahead  int `json:"ahead"`
	Behind int `json:"behind"`
	Errors int `json:"errors"`
}

// BranchInfo repräsentiert Informationen über einen lokalen Git-Branch.
type BranchInfo struct {
	Name         string    `json:"name"`           // Name of the branch.
	IsHead       bool      `json:"is_head"`        // True if this is the current active branch (HEAD).
	Ahead        int       `json:"ahead"`          // Number of commits ahead of the remote tracking branch.
	Behind       int       `json:"behind"`         // Number of commits behind the remote tracking branch.
	LastCommitAt time.Time `json:"last_commit_at"` // Timestamp of the last commit on this branch.
	IsStale      bool      `json:"is_stale"`       // True if the branch is old (e.g., > 30 days) and not main/master.
}

// RepoStatus repräsentiert den vollständigen Zustand eines Git-Repositories.
type RepoStatus struct {
	// --- Identität ---
	Name string `json:"name"` // Ordnername oder Projektname aus INI
	Path string `json:"path"` // Absoluter Pfad auf der Festplatte

	// --- Branch & Head ---
	CurrentBranch string       `json:"current_branch"`  // The currently checked-out branch name.
	LastCommitID  string       `json:"last_commit_id"`  // SHA-1 hash of the last commit.
	LastCommitMsg string       `json:"last_commit_msg"` // Message of the last commit.
	LastCommitAt  time.Time    `json:"last_commit_at"`  // Timestamp of the last commit on HEAD.
	LocalBranches []BranchInfo `json:"local_branches"`  // List of all local branches with their sync status.

	// --- Lokaler Arbeitsstand (Dirty Check) ---
	IsClean        bool           `json:"is_clean"`        // True if there are no uncommitted changes.
	ModifiedCount  int            `json:"modified_count"`  // Count of modified files (staged & unstaged).
	UntrackedCount int            `json:"untracked_count"` // Count of new untracked files.
	DeletedCount   int            `json:"deleted_count"`   // Count of deleted files.
	StashCount     int            `json:"stash_count"`     // Number of stashed changes.
	ChangedFiles   map[string]int `json:"changed_files"`   // Grouping of changes by file extension (e.g., ".go": 3).

	// --- Synchronisation (Remote) ---
	RemoteURL        string    `json:"remote_url"`         // URL of the primary remote (usually origin).
	RemoteBranchName string    `json:"remote_branch_name"` // The remote branch being tracked (e.g., "origin/main").
	IsTracking       bool      `json:"is_tracking"`        // True if the current branch tracks a remote branch.
	Ahead            int       `json:"ahead"`              // Number of local commits to push.
	Behind           int       `json:"behind"`             // Number of remote commits to pull.
	IncomingCommits  []string  `json:"incoming_commits"`   // List of recent commit messages from remote.
	IsDiverged       bool      `json:"is_diverged"`        // True if both ahead and behind (requires merge).
	LastFetchAt      time.Time `json:"last_fetch_at"`      // Timestamp of the last 'git fetch'.
	HasRemoteUpdates bool      `json:"has_remote_updates"` // Flag for UI to indicate remote changes.

	// --- Kritische Zustände ---
	InMergeConflict bool `json:"in_merge_conflict"` // True if a merge conflict is active.
	HasNoRemote     bool `json:"has_no_remote"`     // True if the repository exists only locally.

	// --- Metadaten für UI ---
	MainLanguage string `json:"main_language"` // Primary programming language (optional).
	Error        error  `json:"-"`             // Any error that occurred during scanning (e.g., not a git repo).
	ErrorMsg     string `json:"error,omitempty"` // String representation of the error for JSON output.
}
