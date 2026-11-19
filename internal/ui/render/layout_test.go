// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
	"github.com/hmsoft0815/GitPulseMLC/pkg/visibility"
)

func TestParseLayout(t *testing.T) {
	for in, want := range map[string]Layout{"": LayoutAuto, "AUTO": LayoutAuto, "wide": LayoutWide, " narrow ": LayoutNarrow, "stacked": LayoutStacked} {
		if got, err := ParseLayout(in); err != nil || got != want {
			t.Errorf("ParseLayout(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ParseLayout("tiny"); err == nil {
		t.Error("ParseLayout(tiny) succeeded")
	}
}

func TestResolveLayout(t *testing.T) {
	c := columnWidths{name: 15, branch: 33, remote: 15, status: 60, path: 30} // wide 161, narrow lead 56
	tests := []struct {
		layout Layout
		width  int
		want   Layout
	}{
		{LayoutAuto, 0, LayoutWide}, // not a terminal: never change the output
		{LayoutAuto, 200, LayoutWide},
		{LayoutAuto, 160, LayoutNarrow},
		{LayoutAuto, 88, LayoutNarrow}, // 56 + 32 fits, the status wraps
		{LayoutAuto, 87, LayoutStacked},
		{LayoutStacked, 200, LayoutStacked}, // an explicit layout is kept
		{LayoutWide, 40, LayoutWide},
	}
	for _, tt := range tests {
		if got := tt.layout.resolve(c, tt.width); got != tt.want {
			t.Errorf("%s at %d columns = %s, want %s", tt.layout, tt.width, got, tt.want)
		}
	}
}

// layoutFixture has a long branch name and a long status, like real configs.
func layoutFixture() []*gitmonitor.RepoStatus {
	return []*gitmonitor.RepoStatus{
		{Name: "api-server", Path: "/home/me/code/api-server", CurrentBranch: "main", IsClean: true,
			Remotes:    []gitmonitor.RemoteInfo{{Name: "origin", URL: "https://github.com/me/api.git", Host: "github.com", Visibility: visibility.Public}},
			Visibility: visibility.Public, LocalBranches: []gitmonitor.BranchInfo{{Name: "main", IsHead: true, Ahead: 2}}},
		{Name: "upscale", Path: "/home/me/code/upscale", CurrentBranch: "feature/build-engines-from-source",
			ChangedFiles: map[string]int{"(no ext)": 1, ".nsi": 1, ".sh": 2, ".yml": 1, ".go": 4}, StashCount: 1,
			LocalBranches: []gitmonitor.BranchInfo{{Name: "feature/build-engines-from-source", IsHead: true, Ahead: 1}, {Name: "old", IsStale: true}}},
	}
}

func renderTUI(t *testing.T, layout Layout, width int) []string {
	t.Helper()
	lipgloss.SetColorProfile(termenv.Ascii)
	cfg := &config.Config{Settings: config.Settings{ShowSummary: true, ReplacePathPrefix: "/home/me", ReplaceWith: "~"}}
	var buf bytes.Buffer
	TUI(&buf, layoutFixture(), gitmonitor.Summary{Total: 2, SubmodulePointers: 1}, cfg, TUIOptions{Layout: layout, Width: width, Details: true})
	return strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
}

func TestTUILayoutsFitTheWidth(t *testing.T) {
	for _, width := range []int{120, 90, 70, 50} {
		lines := renderTUI(t, LayoutAuto, width)
		for _, l := range lines {
			if w := lipgloss.Width(l); w > width {
				t.Errorf("width %d: line is %d wide: %q", width, w, l)
			}
		}
		text := strings.Join(lines, "\n")
		// Nothing important may be cut: the sync state ends the status.
		if !strings.Contains(text, "↑1") || !strings.Contains(text, "1 stash") {
			t.Errorf("width %d: status parts lost:\n%s", width, text)
		}
	}
}

func TestTUILayoutShapes(t *testing.T) {
	wide := strings.Join(renderTUI(t, LayoutAuto, 0), "\n")
	if !strings.Contains(wide, "PATH") || !strings.Contains(wide, "feature/build-engines-from-source") {
		t.Errorf("wide layout lacks PATH or the full branch:\n%s", wide)
	}
	narrow := renderTUI(t, LayoutNarrow, 0)
	if strings.Contains(narrow[0], "PATH") || !strings.Contains(strings.Join(narrow, "\n"), "feature/build-engin…") {
		t.Errorf("narrow layout should drop PATH and shorten the branch:\n%s", strings.Join(narrow, "\n"))
	}
	stacked := renderTUI(t, LayoutStacked, 0)
	if stacked[0] != "NAME        STATUS" {
		t.Errorf("stacked header = %q", stacked[0])
	}
	var second string
	for i, l := range stacked {
		if strings.HasPrefix(l, "api-server") {
			second = stacked[i+1]
		}
	}
	if second != "  main · github pub · ~/code/api-server" {
		t.Errorf("stacked second line = %q", second)
	}
}

func TestDetailsStashIsNotTheLastBranch(t *testing.T) {
	text := strings.Join(renderTUI(t, LayoutWide, 0), "\n")
	if strings.Contains(text, "└─ Stashes") || !strings.Contains(text, "├─ Stashes: 1") {
		t.Errorf("stash detail line should use ├─:\n%s", text)
	}
}

func TestThirdPartyHeadingWraps(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	results := []*gitmonitor.RepoStatus{{Name: "whisper.cpp", Path: "/src/whisper.cpp", CurrentBranch: "main", IsClean: true, ThirdParty: true}}
	for _, width := range []int{0, 100, 60, 40} {
		var buf bytes.Buffer
		TUI(&buf, results, gitmonitor.Summary{}, &config.Config{}, TUIOptions{Width: width})
		text := buf.String()
		if strings.Contains(text, "…") || !strings.Contains(text, "only hints") {
			t.Errorf("width %d: heading cut or incomplete:\n%s", width, text)
		}
		for _, l := range strings.Split(text, "\n") {
			if width > 0 && lipgloss.Width(l) > width {
				t.Errorf("width %d: line %q is too wide", width, l)
			}
		}
	}
}
