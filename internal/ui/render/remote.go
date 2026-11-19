// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"fmt"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/styles"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
	"github.com/hmsoft0815/GitPulseMLC/pkg/visibility"

	"github.com/charmbracelet/lipgloss"
)

// visibilityTags are the short labels of the REMOTE column.
var visibilityTags = map[visibility.Visibility]string{
	visibility.Public:  "pub",
	visibility.Private: "priv",
	visibility.Local:   "lan",
	visibility.Unknown: "?",
}

// remoteText renders the REMOTE column: the host of the remote that decides
// the repository's visibility, its tag, and how many other remotes exist,
// e.g. "github pub +1". Repositories without remotes show "—".
func remoteText(s *gitmonitor.RepoStatus) string {
	if len(s.Remotes) == 0 {
		if s.ErrorMsg != "" {
			return ""
		}
		return "—"
	}
	r := s.Remotes[0]
	for _, candidate := range s.Remotes {
		if candidate.Visibility == s.Visibility {
			r = candidate
			break
		}
	}
	text := remoteHost(r)
	if tag := visibilityTags[r.Visibility]; tag != "" {
		text += " " + tag
	}
	if len(s.Remotes) > 1 {
		text += fmt.Sprintf(" +%d", len(s.Remotes)-1)
	}
	return text
}

// remoteHost names the remote's host for display; file remotes show "file".
func remoteHost(r gitmonitor.RemoteInfo) string {
	host := r.Host
	if host == "" {
		host = visibility.Parse(r.URL).Host
	}
	if host == "" {
		return "file"
	}
	return visibility.ShortHost(host)
}

// remoteStyle highlights public repositories and mutes everything else.
func remoteStyle(v visibility.Visibility) lipgloss.Style {
	if v == visibility.Public {
		return styles.Public
	}
	return styles.Muted
}

// printRemoteDetails prints one line per remote for --all.
func printRemoteDetails(remotes []gitmonitor.RemoteInfo) {
	for _, r := range remotes {
		v := string(r.Visibility)
		if v == "" {
			v = "not checked"
		}
		fmt.Printf("  ├─ Remote: %-10s %s %s\n", styles.Branch.Render(r.Name),
			styles.Path.Render(r.URL), remoteStyle(r.Visibility).Render(v))
	}
}
