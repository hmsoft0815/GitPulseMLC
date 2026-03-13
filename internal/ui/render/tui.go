// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"fmt"
	"strings"
	"sort"

	"gitpulsemcl/internal/ui/styles"
	"gitpulsemcl/internal/util"
	"gitpulsemcl/pkg/config"
	"gitpulsemcl/pkg/gitmonitor"

	"github.com/charmbracelet/lipgloss"
)

type rowInfo struct {
	status      *gitmonitor.RepoStatus
	statusStr   string
	statusStyle lipgloss.Style
	needsAction bool
}

func TUI(allResults []*gitmonitor.RepoStatus, summary gitmonitor.Summary, cfg *config.Config, details bool) {
	maxNameLen := len("NAME")
	maxBranchLen := len("BRANCH") + 1
	maxStatusLen := len("STATUS")

	var toDisplay []rowInfo

	for _, status := range allResults {
		info := rowInfo{status: status}

		if status.ErrorMsg != "" {
			info.statusStr = "ERROR"
			info.statusStyle = styles.Behind
			info.needsAction = true
		} else {
			hasAhead, hasBehind := false, false
			for _, b := range status.LocalBranches {
				if b.Ahead > 0 {
					hasAhead = true
				}
				if b.Behind > 0 {
					hasBehind = true
				}
			}

			if !status.IsClean {
				info.statusStr = "DIRTY"
				info.statusStyle = styles.Dirty
				if len(status.ChangedFiles) > 0 {
					var parts []string
					keys := make([]string, 0, len(status.ChangedFiles))
					for k := range status.ChangedFiles {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					const limit = 4
					for i, k := range keys {
						if i >= limit {
							parts = append(parts, fmt.Sprintf("+%d", len(keys)-limit))
							break
						}
						parts = append(parts, fmt.Sprintf("%d%s", status.ChangedFiles[k], k))
					}
					info.statusStr += fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
				}
			} else {
				info.statusStr = "CLEAN"
				info.statusStyle = styles.Clean
			}

			if status.HasNoRemote {
				info.statusStr += " " + styles.NoRemote.Render("(No Remote)")
			}
			info.needsAction = status.ErrorMsg != "" || !status.IsClean || status.StashCount > 0 || hasAhead || hasBehind
		}

		if !cfg.Settings.CompactMode || info.needsAction {
			toDisplay = append(toDisplay, info)
			if len(status.Name) > maxNameLen {
				maxNameLen = len(status.Name)
			}
			if status.ErrorMsg == "" && len(status.CurrentBranch) > maxBranchLen {
				maxBranchLen = len(status.CurrentBranch)
			}
			sLen := lipgloss.Width(info.statusStr)
			if sLen > maxStatusLen {
				maxStatusLen = sLen
			}
		}
	}

	styles.ColName = styles.ColName.Width(maxNameLen + 2)
	styles.ColBranch = styles.ColBranch.Width(maxBranchLen + 2)
	styles.ColStatus = styles.ColStatus.Width(maxStatusLen + 2)

	fmt.Printf("%s%s%s%s\n",
		styles.ColName.Render(styles.Header.Render("NAME")),
		styles.ColBranch.Render(styles.Header.Render("BRANCH")),
		styles.ColStatus.Render(styles.Header.Render("STATUS")),
		styles.Header.Render("PATH"))
	fmt.Println(styles.Muted.Render(strings.Repeat("─", maxNameLen+maxBranchLen+maxStatusLen+60)))

	for _, info := range toDisplay {
		s := info.status
		branch := "???"
		if s.ErrorMsg == "" {
			branch = s.CurrentBranch
		}

		displayPath := ShortenPath(s.Path, cfg.Settings.ReplacePathPrefix, cfg.Settings.ReplaceWith)

		fmt.Printf("%s%s%s%s\n",
			styles.ColName.Render(styles.Header.Render(s.Name)),
			styles.ColBranch.Render(styles.Branch.Render(branch)),
			styles.ColStatus.Render(info.statusStyle.Render(info.statusStr)),
			styles.Path.Render(displayPath))

		if details && s.ErrorMsg == "" {
			if s.StashCount > 0 {
				fmt.Printf("  └─ %s: %d\n", styles.Dirty.Render("Stashes"), s.StashCount)
			}
			for _, b := range s.LocalBranches {
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
				ageStyle := styles.Muted
				stale := ""
				if b.IsStale {
					ageStyle = styles.Behind
					stale = " [STALE]"
				}
				fmt.Printf("%s Branch: %-25s%s%s\n", prefix, styles.Branch.Render(b.Name), sync,
					ageStyle.Render(fmt.Sprintf(" (%s)%s", util.RelativeTime(b.LastCommitAt), stale)))
			}
			fmt.Println(styles.Muted.Render(strings.Repeat("─", 80)))
		}
	}

	if cfg.Settings.ShowSummary {
		fmt.Println()
		fmt.Println(styles.Muted.Render(strings.Repeat("─", 40)))
		fmt.Println(styles.Header.Render("SUMMARY:"))
		fmt.Printf("  Total Projects: %d\n", summary.Total)
		if cfg.Settings.CompactMode {
			fmt.Printf(styles.Muted.Render("  (Compact Mode: showing %d repos needing action)\n"), len(toDisplay))
		}
		renderStat := func(count int, active lipgloss.Style) string {
			if count == 0 {
				return styles.Muted.Render("0")
			}
			return active.Render(fmt.Sprintf("%d", count))
		}
		fmt.Printf("  Dirty:         [ %s ] %s\n", renderStat(summary.Dirty, styles.Dirty), func() string {
			if summary.Dirty > 0 {
				return styles.Dirty.Render("(Action required!)")
			}
			return ""
		}())
		fmt.Printf("  Ahead:         [ %s ] %s\n", renderStat(summary.Ahead, styles.Ahead), func() string {
			if summary.Ahead > 0 {
				return styles.Ahead.Render("(Need to push)")
			}
			return ""
		}())
		fmt.Printf("  Behind:        [ %s ] %s\n", renderStat(summary.Behind, styles.Behind), func() string {
			if summary.Behind > 0 {
				return styles.Behind.Render("(Need to pull)")
			}
			return ""
		}())
		fmt.Printf("  Errors:        [ %s ]\n", renderStat(summary.Errors, styles.Behind))
		fmt.Println(styles.Muted.Render(strings.Repeat("─", 40)))
	}
}
