// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

func htmlFixture() []*gitmonitor.RepoStatus {
	return []*gitmonitor.RepoStatus{
		{Name: "clean-repo", IsClean: true, CurrentBranch: "main"},
		{Name: "ahead-repo", IsClean: true, CurrentBranch: "main",
			LocalBranches: []gitmonitor.BranchInfo{{Ahead: 2}}},
		{Name: "<script>x</script>", CurrentBranch: "feat/<b>", Path: "/home/me/evil",
			ChangedFiles: map[string]int{".go": 1}},
		{Name: "broken", ErrorMsg: "path does not exist: /gone"},
	}
}

func renderHTML(t *testing.T, compact, colored bool) string {
	t.Helper()
	var buf bytes.Buffer
	settings := config.Settings{CompactMode: compact, ReplacePathPrefix: "/home/me", ReplaceWith: "~"}
	if err := HTML(&buf, "1.2.1", htmlFixture(), gitmonitor.Summary{Total: 4}, colored, settings); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestHTMLShowsStatusParts(t *testing.T) {
	out := renderHTML(t, false, true)
	for _, want := range []string{"clean-repo", "CLEAN", "↑2", "DIRTY (1.go)", "path does not exist: /gone", "~/evil", "color:#2563eb"} {
		if !strings.Contains(out, want) {
			t.Errorf("report misses %q", want)
		}
	}
}

func TestHTMLEscapesRepositoryData(t *testing.T) {
	out := renderHTML(t, false, true)
	if strings.Contains(out, "<script>") || strings.Contains(out, "<b>") {
		t.Error("repository names and branches must be HTML-escaped")
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Error("escaped name missing")
	}
}

func TestHTMLCompactHidesClean(t *testing.T) {
	out := renderHTML(t, true, true)
	if strings.Contains(out, "clean-repo") {
		t.Error("compact report lists a clean repository")
	}
	if !strings.Contains(out, "ahead-repo") || !strings.Contains(out, "1 clean hidden") {
		t.Error("compact report must list repositories that need action and count hidden ones")
	}
}

func TestHTMLNoColor(t *testing.T) {
	if out := renderHTML(t, false, false); strings.Contains(out, "color:#2563eb;font-weight") {
		t.Error("colored=false must not use status colours")
	}
}
