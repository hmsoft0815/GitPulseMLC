// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/styles"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

// statusCell builds the STATUS column for one repository: every reason it needs
// attention, each in its own colour, or CLEAN if there is none. needsAction is
// true for anything except CLEAN and informational markers (no remote, stale).
func statusCell(s *gitmonitor.RepoStatus) (cell string, needsAction bool) {
	if s.ErrorMsg != "" {
		return styles.Behind.Render("ERROR"), true
	}

	var parts []string
	if !s.IsClean {
		parts = append(parts, styles.Dirty.Render("DIRTY"+changedFilesSummary(s.ChangedFiles)))
	}
	ahead, behind, stale := 0, 0, 0
	for _, b := range s.LocalBranches {
		ahead += b.Ahead
		behind += b.Behind
		if b.IsStale {
			stale++
		}
	}
	if ahead > 0 {
		parts = append(parts, styles.Ahead.Render(fmt.Sprintf("↑%d", ahead)))
	}
	if behind > 0 {
		parts = append(parts, styles.Behind.Render(fmt.Sprintf("↓%d", behind)))
	}
	if s.StashCount > 0 {
		parts = append(parts, styles.Dirty.Render(fmt.Sprintf("%d stash", s.StashCount)))
	}
	needsAction = len(parts) > 0
	if !needsAction {
		parts = append(parts, styles.Clean.Render("CLEAN"))
	}

	if stale > 0 {
		parts = append(parts, styles.Muted.Render(fmt.Sprintf("%d stale", stale)))
	}
	if s.HasNoRemote {
		parts = append(parts, styles.NoRemote.Render("(No Remote)"))
	}
	return strings.Join(parts, " "), needsAction
}

// changedFilesSummary renders " (2.go, 1.css)" for at most four extensions.
func changedFilesSummary(files map[string]int) string {
	if len(files) == 0 {
		return ""
	}
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	const limit = 4
	var parts []string
	for i, k := range keys {
		if i >= limit {
			parts = append(parts, fmt.Sprintf("+%d", len(keys)-limit))
			break
		}
		parts = append(parts, fmt.Sprintf("%d%s", files[k], k))
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
