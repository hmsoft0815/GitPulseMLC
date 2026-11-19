// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/styles"
	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

// row holds the rendered cells of one repository.
type row struct {
	status                     *gitmonitor.RepoStatus
	name, branch, remote, path string // plain text
	statusCell                 string // styled
	needsAction                bool
}

// buildRows renders the cells of every repository to display.
func buildRows(results []*gitmonitor.RepoStatus, settings config.Settings) ([]row, columnWidths) {
	c := columnWidths{name: len("NAME"), branch: len("BRANCH") + 1, remote: len("REMOTE"), status: len("STATUS"), path: len("PATH")}
	var rows []row
	for _, s := range results {
		r := row{status: s, name: s.Name, branch: "???", remote: remoteText(s),
			path: ShortenPath(s.Path, settings.ReplacePathPrefix, settings.ReplaceWith)}
		if s.ErrorMsg == "" {
			r.branch = s.CurrentBranch
		}
		r.statusCell, r.needsAction = statusCell(s)
		if settings.CompactMode && !r.needsAction {
			continue
		}
		rows = append(rows, r)
		c.name = max(c.name, lipgloss.Width(r.name))
		c.branch = max(c.branch, lipgloss.Width(r.branch))
		c.remote = max(c.remote, lipgloss.Width(r.remote))
		c.status = max(c.status, lipgloss.Width(r.statusCell))
		c.path = max(c.path, lipgloss.Width(r.path))
	}
	return rows, c
}

// pad fills a styled string with spaces to width (plus the column gap).
func pad(s string, width int) string {
	return s + strings.Repeat(" ", max(0, width-lipgloss.Width(s))+gap)
}

// shorten cuts plain text to width with an ellipsis.
func shorten(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "…")
}

// fit cuts a styled line to the terminal width; 0 means no limit.
func fit(s string, width int) string {
	if width <= 0 {
		return s
	}
	return ansi.Truncate(s, width, "…")
}

// writeHeader prints the column titles and the rule for a layout.
func writeHeader(w io.Writer, layout Layout, c columnWidths, width int) {
	h := styles.Header.Render
	var line string
	ruleLen := c.wide()
	switch layout {
	case LayoutWide:
		line = pad(h("NAME"), c.name) + pad(h("BRANCH"), c.branch) + pad(h("REMOTE"), c.remote) + pad(h("STATUS"), c.status) + h("PATH")
	case LayoutNarrow:
		line = pad(h("NAME"), c.name) + pad(h("BRANCH"), min(c.branch, narrowBranchWidth)) + pad(h("REMOTE"), c.remote) + h("STATUS")
		ruleLen = c.narrow()
	default:
		line = pad(h("NAME"), c.name) + h("STATUS")
		ruleLen = c.name + gap + c.status
	}
	if width > 0 {
		ruleLen = min(ruleLen, width)
	}
	fmt.Fprintln(w, fit(line, width))
	fmt.Fprintln(w, styles.Muted.Render(strings.Repeat("─", ruleLen)))
}

// writeRow prints one repository in the given layout.
func writeRow(w io.Writer, layout Layout, r row, c columnWidths, width int) {
	name := pad(styles.Header.Render(r.name), c.name)
	remote := remoteStyle(r.status.Visibility).Render(r.remote)
	switch layout {
	case LayoutWide:
		fmt.Fprintln(w, fit(name+pad(styles.Branch.Render(r.branch), c.branch)+pad(remote, c.remote)+
			pad(r.statusCell, c.status)+styles.Path.Render(r.path), width))
	case LayoutNarrow:
		bw := min(c.branch, narrowBranchWidth)
		writeWrapped(w, name+pad(styles.Branch.Render(shorten(r.branch, bw)), bw)+pad(remote, c.remote), c.narrowLead(), r.statusCell, width)
	default:
		dot := styles.Muted.Render(" · ")
		writeWrapped(w, name, c.name+gap, r.statusCell, width)
		fmt.Fprintln(w, fit("  "+styles.Branch.Render(shorten(r.branch, narrowBranchWidth))+dot+remote+dot+styles.Path.Render(r.path), width))
	}
}

// writeWrapped prints lead followed by the status; a status too long for the
// rest of the line wraps onto further lines, indented to the status column.
func writeWrapped(w io.Writer, lead string, leadWidth int, status string, width int) {
	room := width - leadWidth
	if width <= 0 || lipgloss.Width(status) <= room || room < minStatusWidth/2 {
		fmt.Fprintln(w, fit(lead+status, width))
		return
	}
	indent := strings.Repeat(" ", leadWidth)
	for i, part := range strings.Split(ansi.Wrap(status, room, " "), "\n") {
		if i == 0 {
			fmt.Fprintln(w, lead+part)
			continue
		}
		fmt.Fprintln(w, indent+part)
	}
}
