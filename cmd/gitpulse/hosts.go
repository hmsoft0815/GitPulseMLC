// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/render"
	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

// defaultHostTimeout bounds a remote scan when host_timeout is not set.
const defaultHostTimeout = 180 * time.Second

// startHosts runs gitpulse on every configured host in the background and
// returns a function that waits for all of them. Results come back in host
// order, each repository tagged with its host; a host that fails yields one
// error entry instead.
func startHosts(hosts []config.Host, offline bool, timeout time.Duration) func() []*gitmonitor.RepoStatus {
	if timeout <= 0 {
		timeout = defaultHostTimeout
	}
	results := make([][]*gitmonitor.RepoStatus, len(hosts))
	var wg sync.WaitGroup
	for i, h := range hosts {
		wg.Go(func() { results[i] = scanHost(h, offline, timeout) })
	}
	return func() []*gitmonitor.RepoStatus {
		wg.Wait()
		var all []*gitmonitor.RepoStatus
		for _, r := range results {
			all = append(all, r...)
		}
		return all
	}
}

// scanHost runs the host's command with --json and reads its results.
func scanHost(h config.Host, offline bool, timeout time.Duration) []*gitmonitor.RepoStatus {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	args := append(append([]string{}, h.Command[1:]...), "--json")
	if offline {
		args = append(args, "--offline")
	}
	cmd := exec.CommandContext(ctx, h.Command[0], args...)
	// After a kill on timeout, child processes (ssh, wsl.exe) may still hold
	// the output open; stop waiting for them after a short grace period.
	cmd.WaitDelay = 2 * time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return hostError(h, fmt.Errorf("no answer within %s (host_timeout)", timeout))
	}
	if err != nil {
		return hostError(h, fmt.Errorf("%w%s", err, lastLine(stderr.String())))
	}
	var doc render.Output
	if err := json.Unmarshal(out, &doc); err != nil {
		return hostError(h, fmt.Errorf("no gitpulse JSON (%w)%s", err, lastLine(stderr.String())))
	}
	for _, r := range doc.Results {
		r.Host = h.Name
	}
	return doc.Results
}

// hostError is the single entry shown for a host that could not be scanned.
func hostError(h config.Host, err error) []*gitmonitor.RepoStatus {
	msg := "host " + h.Name + ": " + err.Error()
	return []*gitmonitor.RepoStatus{{Name: h.Name, Host: h.Name, Error: errors.New(msg), ErrorMsg: msg}}
}

// lastLine returns ": <last non-empty line>" of a command's error output.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if last := strings.TrimSpace(lines[len(lines)-1]); last != "" {
		return ": " + last
	}
	return ""
}

// orderResults puts local repositories first, then each host in config
// order; within each group own repositories come before third party.
func orderResults(results []*gitmonitor.RepoStatus, hosts []config.Host) {
	rank := map[string]int{"": 0}
	for i, h := range hosts {
		rank[h.Name] = i + 1
	}
	sort.SliceStable(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if rank[a.Host] != rank[b.Host] {
			return rank[a.Host] < rank[b.Host]
		}
		return !a.ThirdParty && b.ThirdParty
	})
}
