// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/styles"
	"github.com/hmsoft0815/GitPulseMLC/internal/util"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

// writeDetails prints the -v lines below a repository: stashes, remotes,
// submodules and local branches, each cut to the terminal width.
func writeDetails(w io.Writer, s *gitmonitor.RepoStatus, width int) {
	out := func(line string) { fmt.Fprintln(w, fit(line, width)) }
	if s.StashCount > 0 {
		out(fmt.Sprintf("  ├─ %s: %d", styles.Dirty.Render("Stashes"), s.StashCount))
	}
	if s.EOLOnlyCount > 0 {
		out(fmt.Sprintf("  ├─ %s: %d %s", styles.Dirty.Render("Line endings"), s.EOLOnlyCount,
			styles.Muted.Render("file(s) differ only in CRLF/LF — set core.autocrlf or .gitattributes before committing")))
	}
	for _, line := range remoteDetailLines(s.Remotes) {
		out(line)
	}
	for _, line := range submoduleDetailLines(s.Submodules) {
		out(line)
	}
	for _, b := range s.LocalBranches {
		out(branchDetailLine(b))
	}
	rule := 80
	if width > 0 {
		rule = min(rule, width)
	}
	fmt.Fprintln(w, styles.Muted.Render(strings.Repeat("─", rule)))
}

// branchDetailLine renders one local branch with its sync state and age.
func branchDetailLine(b gitmonitor.BranchInfo) string {
	prefix := "  ├─"
	if b.IsHead {
		prefix = styles.Clean.Render("  *─")
	}
	sync := ""
	if b.Ahead > 0 {
		sync += styles.Ahead.Render(fmt.Sprintf(" ↑%d", b.Ahead))
	}
	if b.Behind > 0 {
		sync += styles.Behind.Render(fmt.Sprintf(" ↓%d", b.Behind))
	}
	ageStyle, stale := styles.Muted, ""
	if b.IsStale {
		ageStyle, stale = styles.Behind, " [STALE]"
	}
	return fmt.Sprintf("%s Branch: %-25s%s%s", prefix, styles.Branch.Render(b.Name), sync,
		ageStyle.Render(fmt.Sprintf(" (%s)%s", util.RelativeTime(b.LastCommitAt), stale)))
}
