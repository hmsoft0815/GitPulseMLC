// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/styles"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"

	"github.com/charmbracelet/lipgloss"
)

// partKind says what a piece of the status column means; each renderer maps
// it to its own colour.
type partKind int

const (
	kindError partKind = iota
	kindDirty
	kindAhead
	kindBehind
	kindStash
	kindClean
	kindStale
	kindSubmodule
	kindSubPointer
	kindHint // muted: work in a third-party repository
)

// statusPart is one piece of the status column, e.g. "↑2".
type statusPart struct {
	kind partKind
	text string
}

// statusParts lists every reason a repository needs attention, or CLEAN if
// there is none, followed by informational markers (stale branches,
// submodule pointers). needsAction is false for CLEAN and informational markers alone.
func statusParts(s *gitmonitor.RepoStatus) (parts []statusPart, needsAction bool) {
	if s.ErrorMsg != "" {
		return []statusPart{{kindError, "ERROR"}}, true
	}

	if !s.IsClean {
		parts = append(parts, statusPart{kindDirty, "DIRTY" + changedFilesSummary(s.ChangedFiles)})
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
		parts = append(parts, statusPart{kindAhead, fmt.Sprintf("↑%d", ahead)})
	}
	if behind > 0 {
		parts = append(parts, statusPart{kindBehind, fmt.Sprintf("↓%d", behind)})
	}
	if s.StashCount > 0 {
		parts = append(parts, statusPart{kindStash, fmt.Sprintf("%d stash", s.StashCount)})
	}
	subAction, subPointers := s.SubmoduleAttention()
	if len(subAction) > 0 {
		parts = append(parts, statusPart{kindSubmodule, submoduleSummary(subAction)})
	}
	needsAction = len(parts) > 0
	if !needsAction {
		parts = append(parts, statusPart{kindClean, "CLEAN"})
	}

	if stale > 0 {
		parts = append(parts, statusPart{kindStale, fmt.Sprintf("%d stale", stale)})
	}
	if subPointers > 0 {
		parts = append(parts, statusPart{kindSubPointer, fmt.Sprintf("%d sub-ptr", subPointers)})
	}
	if s.ThirdParty {
		return muteThirdParty(parts)
	}
	return parts, needsAction
}

// muteThirdParty turns local work in a third-party repository into hints.
// Only new upstream commits (↓) still count, as an update to look at.
func muteThirdParty(parts []statusPart) ([]statusPart, bool) {
	needsAction := false
	for i, p := range parts {
		switch p.kind {
		case kindBehind:
			needsAction = true
		case kindDirty, kindAhead, kindStash, kindSubmodule:
			parts[i].kind = kindHint
		}
	}
	return parts, needsAction
}

// terminalStyles maps each part kind to its terminal colour.
var terminalStyles = map[partKind]lipgloss.Style{
	kindError:      styles.Behind,
	kindDirty:      styles.Dirty,
	kindAhead:      styles.Ahead,
	kindBehind:     styles.Behind,
	kindStash:      styles.Dirty,
	kindClean:      styles.Clean,
	kindStale:      styles.Muted,
	kindSubmodule:  styles.Dirty,
	kindSubPointer: styles.Muted,
	kindHint:       styles.Muted,
}

// statusCell renders the STATUS column for the terminal.
func statusCell(s *gitmonitor.RepoStatus) (cell string, needsAction bool) {
	parts, needsAction := statusParts(s)
	rendered := make([]string, len(parts))
	for i, p := range parts {
		rendered[i] = terminalStyles[p.kind].Render(p.text)
	}
	return strings.Join(rendered, " "), needsAction
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

// submoduleSummary renders "SUB .mlcai" or "SUB .mlcai +2" for the
// submodules that need action.
func submoduleSummary(paths []string) string {
	text := "SUB " + paths[0]
	if len(paths) > 1 {
		text += fmt.Sprintf(" +%d", len(paths)-1)
	}
	return text
}
