// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// Package visibility tells whether a Git remote can be read by anyone.
//
// Remotes on the local network (file paths, .local/.lan hosts, private IPs,
// configured hosts) are always private and need no network. For every other
// remote an anonymous request to Git's smart HTTP endpoint decides: if it
// answers without credentials, the repository is public.
package visibility

import (
	"net/url"
	"regexp"
	"strings"
)

// Endpoint is the parsed location of a remote.
type Endpoint struct {
	Scheme string // "https", "http", "ssh", "git" or "file".
	Host   string // Host name without user and port; empty for file remotes.
	Path   string // Repository path without leading slash.
}

// IsFile reports whether the remote is a path on this machine or a share.
func (e Endpoint) IsFile() bool { return e.Scheme == "file" }

// scpLike matches "user@host:path" and "host:path" (not "C:\path").
var scpLike = regexp.MustCompile(`^(?:[^@/:]+@)?([^/:]{2,}):(.*)$`)

// Parse reads the URL forms Git accepts: scheme URLs, the scp-like SSH form
// and local paths.
func Parse(raw string) Endpoint {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err == nil {
			scheme := strings.ToLower(u.Scheme)
			if scheme == "ssh+git" || scheme == "git+ssh" {
				scheme = "ssh"
			}
			return Endpoint{Scheme: scheme, Host: strings.ToLower(u.Hostname()), Path: strings.TrimPrefix(u.Path, "/")}
		}
	}
	if m := scpLike.FindStringSubmatch(raw); m != nil {
		return Endpoint{Scheme: "ssh", Host: strings.ToLower(m[1]), Path: strings.TrimPrefix(m[2], "/")}
	}
	return Endpoint{Scheme: "file", Path: raw}
}

// Sanitize removes credentials from a remote URL so it can be shown or
// written to a report. HTTP(S) URLs lose their user info completely, since
// tokens often sit in the user name; other URLs keep the user but no password.
func Sanitize(raw string) string {
	if !strings.Contains(raw, "://") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		u.User = nil
	default:
		u.User = url.User(u.User.Username())
	}
	return u.String()
}

// ShortHost shortens well-known forge hosts for display ("github.com" →
// "github") and leaves every other host unchanged.
func ShortHost(host string) string {
	switch host {
	case "github.com", "gitlab.com", "codeberg.org", "bitbucket.org", "sr.ht", "git.sr.ht":
		return strings.SplitN(strings.TrimPrefix(host, "git."), ".", 2)[0]
	}
	return host
}
