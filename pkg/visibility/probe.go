// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package visibility

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Visibility says who can read a remote.
type Visibility string

const (
	Public  Visibility = "public"  // Anyone can read it without credentials.
	Private Visibility = "private" // External host that refused an anonymous read.
	Local   Visibility = "local"   // Path or host on the local network; private by definition.
	Unknown Visibility = "unknown" // Not checked (offline) or the host could not be reached.
	None    Visibility = "none"    // The repository has no remote.
)

// rank orders visibilities from most to least open.
var rank = map[Visibility]int{Public: 4, Unknown: 3, Private: 2, Local: 1, None: 0}

// MoreOpen reports whether a is more open than b; a repository is as open as
// its most open remote.
func MoreOpen(a, b Visibility) bool { return rank[a] > rank[b] }

// probeURL builds the smart HTTP URL that lists a repository's refs.
func probeURL(e Endpoint) string {
	scheme := "https"
	if e.Scheme == "http" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/%s/info/refs?service=git-upload-pack", scheme, e.Host, strings.TrimSuffix(e.Path, "/"))
}

// probe asks the host anonymously for the repository's refs. Only a Git
// ref advertisement counts as public; an answer without one (401, 403, 404,
// a login page) is private, and a failed connection or a server error is
// unknown. No credentials are sent: the request does not use Git, its
// configuration, credential helpers or .netrc.
func probe(ctx context.Context, client *http.Client, rawURL string) Visibility {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Unknown
	}
	// Forges answer Git clients with 401 instead of a login page.
	req.Header.Set("User-Agent", "git/2.0 (gitpulse)")
	req.Header.Set("Git-Protocol", "version=2")
	resp, err := client.Do(req)
	if err != nil {
		return Unknown
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))

	switch {
	case resp.StatusCode == http.StatusOK &&
		strings.HasPrefix(resp.Header.Get("Content-Type"), "application/x-git-upload-pack-advertisement"):
		return Public
	case resp.StatusCode >= 500:
		return Unknown
	default:
		return Private
	}
}
