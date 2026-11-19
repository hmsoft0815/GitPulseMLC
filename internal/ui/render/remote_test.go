// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"testing"

	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
	"github.com/hmsoft0815/GitPulseMLC/pkg/visibility"
)

func TestRemoteText(t *testing.T) {
	github := gitmonitor.RemoteInfo{Name: "origin", URL: "git@github-work:o/r.git", Host: "github.com", Visibility: visibility.Public}
	nas := gitmonitor.RemoteInfo{Name: "nas", URL: "nas.local:/r.git", Host: "nas.local", Visibility: visibility.Local}
	tests := []struct {
		name   string
		status gitmonitor.RepoStatus
		want   string
	}{
		{"no remote", gitmonitor.RepoStatus{}, "—"},
		{"scan error", gitmonitor.RepoStatus{ErrorMsg: "boom"}, ""},
		{"public", gitmonitor.RepoStatus{Remotes: []gitmonitor.RemoteInfo{github}, Visibility: visibility.Public}, "github pub"},
		{"most open decides", gitmonitor.RepoStatus{Remotes: []gitmonitor.RemoteInfo{nas, github}, Visibility: visibility.Public}, "github pub +1"},
		{"lan", gitmonitor.RepoStatus{Remotes: []gitmonitor.RemoteInfo{nas}, Visibility: visibility.Local}, "nas.local lan"},
		{"third party shows kind, not visibility", gitmonitor.RepoStatus{ThirdParty: true, Remotes: []gitmonitor.RemoteInfo{{Name: "origin", URL: "https://github.com/x/y"}}}, "github 3rd"},
		{"not checked", gitmonitor.RepoStatus{Remotes: []gitmonitor.RemoteInfo{{Name: "origin", URL: "https://gitlab.com/o/r.git"}}}, "gitlab"},
		{"file remote", gitmonitor.RepoStatus{Remotes: []gitmonitor.RemoteInfo{{Name: "origin", URL: "/srv/r.git", Visibility: visibility.Local}}, Visibility: visibility.Local}, "file lan"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := remoteText(&tt.status); got != tt.want {
				t.Errorf("remoteText() = %q, want %q", got, tt.want)
			}
		})
	}
}
