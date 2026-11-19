// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"fmt"
	"strings"
)

// Layout says how the terminal table uses the available width.
type Layout string

const (
	LayoutAuto    Layout = "auto"    // Pick the widest layout that fits the terminal.
	LayoutWide    Layout = "wide"    // One line per repository with every column.
	LayoutNarrow  Layout = "narrow"  // One line without PATH; long branch names shortened.
	LayoutStacked Layout = "stacked" // Two lines: name and status, then branch · remote · path.
)

// narrowBranchWidth is the most a branch name may take in the narrow layout.
const narrowBranchWidth = 20

// minStatusWidth is the room the status needs at least before a layout
// counts as fitting; longer statuses wrap within their column.
const minStatusWidth = 32

// ParseLayout reads a layout name; "" means auto.
func ParseLayout(s string) (Layout, error) {
	switch l := Layout(strings.ToLower(strings.TrimSpace(s))); l {
	case "":
		return LayoutAuto, nil
	case LayoutAuto, LayoutWide, LayoutNarrow, LayoutStacked:
		return l, nil
	}
	return "", fmt.Errorf("unknown layout %q — use auto, wide, narrow or stacked", s)
}

// columnWidths are the widest cell of each column, without padding.
type columnWidths struct {
	name, branch, remote, status, path int
}

// gap separates columns.
const gap = 2

// wide is the line width of the wide layout.
func (c columnWidths) wide() int {
	return c.name + c.branch + c.remote + c.status + 4*gap + c.path
}

// narrowLead is the width before the status column in the narrow layout.
func (c columnWidths) narrowLead() int {
	return c.name + min(c.branch, narrowBranchWidth) + c.remote + 3*gap
}

// narrow is the line width of the narrow layout without wrapping.
func (c columnWidths) narrow() int {
	return c.narrowLead() + c.status
}

// resolve turns auto into a concrete layout for a terminal of the given
// width; width 0 means unlimited (output is not a terminal).
func (l Layout) resolve(c columnWidths, width int) Layout {
	if l != LayoutAuto {
		return l
	}
	switch {
	case width <= 0 || c.wide() <= width:
		return LayoutWide
	case c.narrowLead()+min(c.status, minStatusWidth) <= width:
		return LayoutNarrow // a longer status wraps within its column
	default:
		return LayoutStacked
	}
}
