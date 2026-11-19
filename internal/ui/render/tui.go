// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/styles"
	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// TUIOptions control the terminal output.
type TUIOptions struct {
	Details bool   // -v: remotes, submodules, branches and stashes per repository
	Layout  Layout // how the table uses the width; "" means auto
	Width   int    // terminal width in columns; 0 when the output is not a terminal
}

// TUI writes the status table and the summary for a terminal.
func TUI(w io.Writer, results []*gitmonitor.RepoStatus, summary gitmonitor.Summary, cfg *config.Config, opts TUIOptions) {
	rows, cols := buildRows(results, cfg.Settings)
	layout := opts.Layout
	if layout == "" {
		layout = LayoutAuto
	}
	layout = layout.resolve(cols, opts.Width)

	writeHeader(w, layout, cols, opts.Width)
	host, thirdPartyShown := "", false
	for _, r := range rows {
		if r.status.Host != host {
			host, thirdPartyShown = r.status.Host, false
			fmt.Fprintln(w)
			fmt.Fprintln(w, fit(styles.Header.Render("HOST "+host)+" "+
				styles.Muted.Render("— remote repositories, scanned there"), opts.Width))
		}
		if r.status.ThirdParty && !thirdPartyShown {
			thirdPartyShown = true
			writeThirdPartyHeading(w, opts.Width)
		}
		writeRow(w, layout, r, cols, opts.Width)
		if opts.Details && r.status.ErrorMsg == "" {
			writeDetails(w, r.status, opts.Width)
		}
	}

	if cfg.Settings.ShowSummary {
		writeSummary(w, summary, len(rows), cfg.Settings.CompactMode, opts.Width)
	}
}

// writeThirdPartyHeading starts the third-party block. In a narrow terminal
// the explanation moves below the heading and wraps instead of being cut.
func writeThirdPartyHeading(w io.Writer, width int) {
	heading := styles.Header.Render("THIRD PARTY")
	note := "checked out to look at or test; local changes are only hints"
	fmt.Fprintln(w)
	if width <= 0 || lipgloss.Width(heading)+3+len(note) <= width {
		fmt.Fprintln(w, heading+" "+styles.Muted.Render("— "+note))
		return
	}
	fmt.Fprintln(w, heading)
	for _, line := range strings.Split(ansi.Wrap(note, width, " "), "\n") {
		fmt.Fprintln(w, styles.Muted.Render(line))
	}
}

// writeSummary prints the counts below the table.
func writeSummary(w io.Writer, summary gitmonitor.Summary, shown int, compact bool, width int) {
	rule := styles.Muted.Render(strings.Repeat("─", 40))
	line := func(format string, args ...any) { fmt.Fprintln(w, fit(fmt.Sprintf(format, args...), width)) }
	stat := func(count int, active lipgloss.Style) string {
		if count == 0 {
			return styles.Muted.Render("0")
		}
		return active.Render(fmt.Sprintf("%d", count))
	}
	note := func(count int, style lipgloss.Style, text string) string {
		if count == 0 {
			return ""
		}
		return style.Render(text)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, rule)
	fmt.Fprintln(w, styles.Header.Render("SUMMARY:"))
	line("  Total Projects: %d", summary.Total)
	if compact {
		line("%s", styles.Muted.Render(fmt.Sprintf("  (Compact Mode: showing %d repos needing action)", shown)))
	}
	line("  Dirty:         [ %s ] %s", stat(summary.Dirty, styles.Dirty), note(summary.Dirty, styles.Dirty, "(Action required!)"))
	line("  Ahead:         [ %s ] %s", stat(summary.Ahead, styles.Ahead), note(summary.Ahead, styles.Ahead, "(Need to push)"))
	line("  Behind:        [ %s ] %s", stat(summary.Behind, styles.Behind), note(summary.Behind, styles.Behind, "(Need to pull)"))
	line("  Errors:        [ %s ]", stat(summary.Errors, styles.Behind))
	if summary.Missing > 0 {
		line("%s", styles.Muted.Render(fmt.Sprintf("                 %d no longer exist — `gitpulse remove --missing` forgets them", summary.Missing)))
	}
	if summary.Public > 0 {
		line("  Public:        [ %s ] %s", stat(summary.Public, styles.Public), styles.Muted.Render("(readable by anyone)"))
	}
	if summary.ThirdParty > 0 {
		line("  Third party:   [ %s ] %s", stat(summary.ThirdParty, styles.Muted),
			styles.Muted.Render(fmt.Sprintf("(%d with upstream updates)", summary.ThirdPartyUpdates)))
	}
	if summary.SubmodulePointers > 0 {
		line("  Sub-Pointers:  [ %s ] %s", stat(summary.SubmodulePointers, styles.Muted),
			styles.Muted.Render("(commit the new pointer when convenient)"))
	}
	fmt.Fprintln(w, rule)
}
