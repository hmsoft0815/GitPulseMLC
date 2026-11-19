// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package visibility

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// DefaultTTL is how long a probe result is trusted.
const DefaultTTL = 7 * 24 * time.Hour

// Options configure a Checker. Zero values pick the defaults.
type Options struct {
	PrivateHosts []string                 // Glob patterns of hosts that are always local.
	Offline      bool                     // Never probe; use cached results regardless of age.
	CachePath    string                   // Cache file; "" keeps results in memory only.
	TTL          time.Duration            // Age after which a result is probed again.
	Client       *http.Client             // HTTP client for probes.
	ResolveSSH   func(host string) string // Maps SSH aliases to host names (default: ssh -G).
}

// Result is the verdict for one remote URL.
type Result struct {
	Host       string     `json:"host"`       // Real host name, SSH aliases resolved; empty for file remotes.
	Visibility Visibility `json:"visibility"` // Who can read the remote.
}

// Checker decides the visibility of remotes. It is safe for concurrent use.
type Checker struct {
	opts  Options
	cache *cache

	mu       sync.Mutex
	sshHosts map[string]string
}

// NewChecker loads the cache and returns a Checker.
func NewChecker(opts Options) *Checker {
	if opts.TTL == 0 {
		opts.TTL = DefaultTTL
	}
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 8 * time.Second}
	}
	if opts.ResolveSSH == nil {
		opts.ResolveSSH = SSHHostname
	}
	return &Checker{opts: opts, cache: loadCache(opts.CachePath, opts.TTL), sshHosts: map[string]string{}}
}

// Check classifies one remote URL.
func (c *Checker) Check(ctx context.Context, rawURL string) Result {
	e := Parse(rawURL)
	if e.IsFile() {
		return Result{Visibility: Local}
	}
	if e.Scheme == "ssh" {
		e.Host = c.sshHost(e.Host)
	}
	res := Result{Host: e.Host}
	if IsLocalHost(e.Host, c.opts.PrivateHosts) {
		res.Visibility = Local
		return res
	}

	key := probeURL(e)
	if v, ok := c.cache.get(key, c.opts.Offline); ok {
		res.Visibility = v
		return res
	}
	if c.opts.Offline {
		res.Visibility = Unknown
		return res
	}
	res.Visibility = probe(ctx, c.opts.Client, key)
	c.cache.put(key, res.Visibility)
	return res
}

// Save writes new probe results to the cache file.
func (c *Checker) Save() error { return c.cache.save() }

// sshHost resolves an SSH alias once per run.
func (c *Checker) sshHost(alias string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if host, ok := c.sshHosts[alias]; ok {
		return host
	}
	host := c.opts.ResolveSSH(alias)
	if host == "" {
		host = alias
	}
	c.sshHosts[alias] = host
	return host
}
