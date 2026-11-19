// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"sort"

	"github.com/go-git/go-git/v5"

	"github.com/hmsoft0815/GitPulseMLC/pkg/visibility"
)

// collectRemotes lists the repository's remotes with their fetch URL,
// credentials removed, "origin" first and the rest by name.
func collectRemotes(repo *git.Repository) []RemoteInfo {
	remotes, err := repo.Remotes()
	if err != nil {
		return nil
	}
	var infos []RemoteInfo
	for _, r := range remotes {
		cfg := r.Config()
		if len(cfg.URLs) == 0 {
			continue
		}
		infos = append(infos, RemoteInfo{Name: cfg.Name, URL: visibility.Sanitize(cfg.URLs[0])})
	}
	sort.Slice(infos, func(i, j int) bool {
		if (infos[i].Name == "origin") != (infos[j].Name == "origin") {
			return infos[i].Name == "origin"
		}
		return infos[i].Name < infos[j].Name
	})
	return infos
}
