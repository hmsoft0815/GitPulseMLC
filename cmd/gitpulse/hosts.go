// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/render"
	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

// defaultHostTimeout bounds a remote scan when host_timeout is not set.
const defaultHostTimeout = 180 * time.Second

// hostScans are the remote scans running in the background.
type hostScans struct {
	hosts   []config.Host
	results [][]*gitmonitor.RepoStatus
	done    []atomic.Bool
	start   time.Time
	wg      sync.WaitGroup
}

// startHosts runs gitpulse on every configured host in the background.
func startHosts(hosts []config.Host, offline bool, timeout time.Duration) *hostScans {
	if timeout <= 0 {
		timeout = defaultHostTimeout
	}
	s := &hostScans{
		hosts:   hosts,
		results: make([][]*gitmonitor.RepoStatus, len(hosts)),
		done:    make([]atomic.Bool, len(hosts)),
		start:   time.Now(),
	}
	for i, h := range hosts {
		s.wg.Go(func() {
			s.results[i] = scanHost(h, offline, timeout)
			s.done[i].Store(true)
		})
	}
	return s
}

// wait waits for all hosts and returns their results in host order, each
// repository tagged with its host; a host that fails yields one error entry
// instead. With progress, a line on stderr names the hosts still running,
// so a slow host does not look like a hang.
func (s *hostScans) wait(progress bool) []*gitmonitor.RepoStatus {
	finished := make(chan struct{})
	go func() { s.wg.Wait(); close(finished) }()
	if progress && len(s.hosts) > 0 {
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		width := 0
	loop:
		for {
			line := fmt.Sprintf("Waiting for %s … %ds", strings.Join(s.pending(), ", "), int(time.Since(s.start).Seconds()))
			width = max(width, len(line))
			fmt.Fprintf(os.Stderr, "\r%-*s", width, line)
			select {
			case <-finished:
				break loop
			case <-tick.C:
			}
		}
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", width))
	}
	<-finished
	var all []*gitmonitor.RepoStatus
	for _, r := range s.results {
		all = append(all, r...)
	}
	return all
}

// pending names the hosts that have not answered yet.
func (s *hostScans) pending() []string {
	var names []string
	for i, h := range s.hosts {
		if !s.done[i].Load() {
			names = append(names, h.Name)
		}
	}
	return names
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
