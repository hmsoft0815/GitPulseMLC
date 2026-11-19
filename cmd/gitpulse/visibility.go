// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
	"github.com/hmsoft0815/GitPulseMLC/pkg/visibility"
)

// maxProbes limits concurrent visibility probes; they wait on the network,
// not the CPU.
const maxProbes = 16

// checkVisibility fills Host and Visibility of every remote and sets each
// repository's Visibility to that of its most open remote.
func checkVisibility(results []*gitmonitor.RepoStatus, settings config.Settings, offline bool) {
	checker := visibility.NewChecker(visibility.Options{
		PrivateHosts: settings.PrivateHosts,
		Offline:      offline,
		CachePath:    visibility.DefaultCachePath(),
	})

	sem := make(chan struct{}, maxProbes)
	var wg sync.WaitGroup
	for _, r := range results {
		if r.ThirdParty {
			continue // somebody else's repository: being public is normal there
		}
		for i := range r.Remotes {
			remote := &r.Remotes[i]
			wg.Go(func() {
				sem <- struct{}{}
				defer func() { <-sem }()
				res := checker.Check(context.Background(), remote.URL)
				remote.Host, remote.Visibility = res.Host, res.Visibility
			})
		}
	}
	wg.Wait()

	for _, r := range results {
		if r.ErrorMsg != "" || r.ThirdParty {
			continue
		}
		r.Visibility = visibility.None
		for _, remote := range r.Remotes {
			if r.Visibility == visibility.None || visibility.MoreOpen(remote.Visibility, r.Visibility) {
				r.Visibility = remote.Visibility
			}
		}
	}
	if err := checker.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
}

// onlyPublic keeps the repositories with at least one public remote.
func onlyPublic(results []*gitmonitor.RepoStatus) []*gitmonitor.RepoStatus {
	var public []*gitmonitor.RepoStatus
	for _, r := range results {
		if r.Visibility == visibility.Public {
			public = append(public, r)
		}
	}
	return public
}
