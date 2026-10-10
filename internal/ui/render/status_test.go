// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"testing"

	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestStatusCell(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)

	tests := []struct {
		name       string
		status     gitmonitor.RepoStatus
		want       string
		wantAction bool
	}{
		{"clean", gitmonitor.RepoStatus{IsClean: true}, "CLEAN", false},
		{"error", gitmonitor.RepoStatus{ErrorMsg: "boom"}, "ERROR", true},
		{"dirty", gitmonitor.RepoStatus{ChangedFiles: map[string]int{".go": 2, ".css": 1}}, "DIRTY (1.css, 2.go)", true},
		{"ahead and behind", gitmonitor.RepoStatus{IsClean: true, LocalBranches: []gitmonitor.BranchInfo{
			{Ahead: 2}, {Ahead: 1, Behind: 3},
		}}, "↑3 ↓3", true},
		{"stash", gitmonitor.RepoStatus{IsClean: true, StashCount: 1}, "1 stash", true},
		{"stale is info only", gitmonitor.RepoStatus{IsClean: true, LocalBranches: []gitmonitor.BranchInfo{{IsStale: true}}}, "CLEAN 1 stale", false},
		{"no remote is info only", gitmonitor.RepoStatus{IsClean: true, HasNoRemote: true}, "CLEAN (No Remote)", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, action := statusCell(&tt.status)
			if got != tt.want || action != tt.wantAction {
				t.Errorf("statusCell() = %q, %v; want %q, %v", got, action, tt.want, tt.wantAction)
			}
		})
	}
}
