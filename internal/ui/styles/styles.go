// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	Clean  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))  // Clean (Green)
	Dirty  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Dirty/Modified (Orange)
	Ahead  = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // Local commits ahead (Blue)
	Behind = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Remote commits behind (Red)
	Muted  = lipgloss.NewStyle().Foreground(lipgloss.Color("242")) // Muted/Metadata text (Gray)
	Branch = lipgloss.NewStyle().Foreground(lipgloss.Color("212")) // Branch name (Pink/Purple)
	Path   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // File path (Dark Gray)
	Header = lipgloss.NewStyle().Bold(true).Underline(true)        // Table header labels
	NoRemote = lipgloss.NewStyle().Foreground(lipgloss.Color("86")) // No Remote tag (Cyan/Teal)

	// ColName is the style for the Project Name column. Its width is adjusted dynamically.
	ColName = lipgloss.NewStyle().Width(25).PaddingRight(2)
	// ColBranch is the style for the current branch column. Its width is adjusted dynamically.
	ColBranch = lipgloss.NewStyle().PaddingRight(2)
	// ColStatus is the style for the status column (CLEAN/DIRTY).
	ColStatus = lipgloss.NewStyle().PaddingRight(2)
)
