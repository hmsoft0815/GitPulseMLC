// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// Package ownership decides whether a repository is your own or third party,
// from its remotes and a list of remotes that belong to you.
package ownership

import (
	"path"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"

	"github.com/hmsoft0815/GitPulseMLC/pkg/visibility"
)

// Kind says whose repository it is.
type Kind string

const (
	Own        Kind = "own" // Yours: no remote, a local remote, or one of OwnRemotes.
	ThirdParty Kind = "3rd" // Somebody else's, checked out to look at or test.
)

// forges are hosts where the first path segment names the owner.
var forges = map[string]bool{
	"github.com": true, "gitlab.com": true, "codeberg.org": true, "bitbucket.org": true,
}

// Classifier sorts repositories by their remotes. It is safe for concurrent use.
type Classifier struct {
	OwnRemotes   []string                 // Patterns like "github.com/me" or "git.example.com".
	PrivateHosts []string                 // Local-network hosts, see visibility.IsLocalHost.
	ResolveSSH   func(host string) string // SSH alias resolver (default: visibility.SSHHostname).

	mu       sync.Mutex
	sshHosts map[string]string
}

// Classify returns Own if the repository has no remote, a remote on the
// local network, or a remote matching OwnRemotes. Without OwnRemotes every
// repository is Own, since nothing tells it apart.
func (c *Classifier) Classify(remoteURLs []string) Kind {
	if len(remoteURLs) == 0 || len(c.OwnRemotes) == 0 {
		return Own
	}
	for _, u := range remoteURLs {
		key, local := c.Key(u)
		if local {
			return Own
		}
		for _, pattern := range c.OwnRemotes {
			if Matches(pattern, key) {
				return Own
			}
		}
	}
	return ThirdParty
}

// Key returns "host/path" of a remote without ".git", with SSH aliases
// resolved, and whether the remote is on the local network.
func (c *Classifier) Key(rawURL string) (key string, local bool) {
	e := visibility.Parse(rawURL)
	if e.IsFile() {
		return "", true
	}
	if e.Scheme == "ssh" {
		e.Host = c.sshHost(e.Host)
	}
	key = strings.TrimSuffix(e.Host+"/"+strings.Trim(e.Path, "/"), ".git")
	return key, visibility.IsLocalHost(e.Host, c.PrivateHosts)
}

// OwnerKey shortens a key to what an own_remotes entry would name: the host
// and owner on known forges ("github.com/me"), the host elsewhere.
func OwnerKey(key string) string {
	host, rest, _ := strings.Cut(key, "/")
	if !forges[host] {
		return host
	}
	owner, _, _ := strings.Cut(rest, "/")
	return host + "/" + owner
}

// Matches reports whether pattern names key or a prefix of it, segment by
// segment, with glob patterns per segment: "github.com/me" matches
// "github.com/me/tool", "*.example.com" matches "git.example.com/x".
func Matches(pattern, key string) bool {
	ps := strings.Split(strings.Trim(strings.ToLower(pattern), "/"), "/")
	ks := strings.Split(strings.ToLower(key), "/")
	if len(ps) > len(ks) {
		return false
	}
	for i, p := range ps {
		if ok, _ := path.Match(p, ks[i]); !ok {
			return false
		}
	}
	return true
}

// RemoteURLs returns the fetch URLs of the repository at repoPath.
func RemoteURLs(repoPath string) ([]string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}
	remotes, err := repo.Remotes()
	if err != nil {
		return nil, err
	}
	var urls []string
	for _, r := range remotes {
		if u := r.Config().URLs; len(u) > 0 {
			urls = append(urls, u[0])
		}
	}
	return urls, nil
}

// sshHost resolves an SSH alias once.
func (c *Classifier) sshHost(alias string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if host, ok := c.sshHosts[alias]; ok {
		return host
	}
	resolve := c.ResolveSSH
	if resolve == nil {
		resolve = visibility.SSHHostname
	}
	host := resolve(alias)
	if host == "" {
		host = alias
	}
	if c.sshHosts == nil {
		c.sshHosts = map[string]string{}
	}
	c.sshHosts[alias] = host
	return host
}
