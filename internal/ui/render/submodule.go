// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"fmt"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/styles"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

// submodulePointerText explains how the submodule's checkout relates to the
// commit its parent records.
func submodulePointerText(sub gitmonitor.SubmoduleInfo) string {
	switch {
	case sub.NotInitialized:
		return "not initialized (git submodule update --init)"
	case sub.PointerStaged:
		return "new pointer staged in parent"
	case sub.PointerAhead > 0:
		return fmt.Sprintf("pointer ↑%d, not committed in parent", sub.PointerAhead)
	case sub.PointerBehind > 0:
		return fmt.Sprintf("checkout ↓%d behind recorded commit (git submodule update)", sub.PointerBehind)
	case sub.PointerChanged():
		return "checkout differs from recorded commit"
	default:
		return "pointer up to date"
	}
}

// submoduleDetailLines renders one line per submodule for -v.
func submoduleDetailLines(subs []gitmonitor.SubmoduleInfo) []string {
	lines := make([]string, 0, len(subs))
	for _, sub := range subs {
		pointerStyle := styles.Muted
		if sub.PointerChanged() || sub.PointerStaged {
			pointerStyle = styles.Dirty
		}
		own := ""
		if sub.Status != nil {
			cell, _ := statusCell(sub.Status)
			own = " · " + cell
		}
		lines = append(lines, fmt.Sprintf("  ├─ Submodule: %-25s%s%s", styles.Branch.Render(sub.Path),
			pointerStyle.Render(submodulePointerText(sub)), own))
	}
	return lines
}
