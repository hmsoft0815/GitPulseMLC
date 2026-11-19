// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package visibility

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// forge serves /public.git anonymously and answers everything else with 401,
// like GitHub. Its certificate is valid for example.com, an external host.
func forge(t *testing.T) (*http.Client, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Authorization") != "" {
			t.Error("probe sent credentials")
		}
		switch r.URL.Path {
		case "/o/public.git/info/refs":
			w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
			_, _ = w.Write([]byte("001e# service=git-upload-pack\n0000"))
		case "/o/broken.git/info/refs":
			w.WriteHeader(http.StatusBadGateway)
		case "/o/website.git/info/refs":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html>Sign in</html>"))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	t.Cleanup(srv.Close)

	client := srv.Client()
	transport := client.Transport.(*http.Transport)
	addr := srv.Listener.Addr().String()
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	return client, &hits
}

func newTestChecker(t *testing.T, client *http.Client, cachePath string, offline bool) *Checker {
	t.Helper()
	return NewChecker(Options{
		Client:     client,
		CachePath:  cachePath,
		Offline:    offline,
		ResolveSSH: func(host string) string { return map[string]string{"forge-alias": "example.com"}[host] },
	})
}

func TestCheckerProbes(t *testing.T) {
	client, _ := forge(t)
	c := newTestChecker(t, client, "", false)
	tests := []struct {
		url  string
		want Result
	}{
		{"https://example.com/o/public.git", Result{"example.com", Public}},
		{"git@forge-alias:o/public.git", Result{"example.com", Public}},
		{"https://example.com/o/secret.git", Result{"example.com", Private}},
		{"https://example.com/o/website.git", Result{"example.com", Private}},
		{"https://example.com/o/broken.git", Result{"example.com", Unknown}},
		{"mlc@nas.local:/volume1/x.git", Result{"nas.local", Local}},
		{"/srv/git/x.git", Result{"", Local}},
	}
	for _, tt := range tests {
		if got := c.Check(context.Background(), tt.url); got != tt.want {
			t.Errorf("Check(%q) = %+v, want %+v", tt.url, got, tt.want)
		}
	}
}

func TestCheckerCache(t *testing.T) {
	client, hits := forge(t)
	cachePath := filepath.Join(t.TempDir(), "gitpulse", "visibility.json")
	const url = "https://example.com/o/public.git"

	c := newTestChecker(t, client, cachePath, false)
	c.Check(context.Background(), url)
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}

	again := newTestChecker(t, client, cachePath, false)
	if got := again.Check(context.Background(), url); got.Visibility != Public || hits.Load() != 1 {
		t.Errorf("cached Check = %v after %d probes, want public after 1", got.Visibility, hits.Load())
	}

	// Offline: an expired result still counts, an unknown remote stays unknown.
	again.cache.entries[probeURL(Parse(url))] = cacheEntry{Public, time.Now().Add(-30 * 24 * time.Hour)}
	offline := newTestChecker(t, client, "", true)
	offline.cache = again.cache
	if got := offline.Check(context.Background(), url); got.Visibility != Public {
		t.Errorf("offline with expired cache = %v, want public", got.Visibility)
	}
	if got := offline.Check(context.Background(), "https://example.com/o/other.git"); got.Visibility != Unknown {
		t.Errorf("offline without cache = %v, want unknown", got.Visibility)
	}
	if hits.Load() != 1 {
		t.Errorf("offline mode probed: %d requests, want 1", hits.Load())
	}
}

func TestMoreOpen(t *testing.T) {
	if !MoreOpen(Public, Unknown) || !MoreOpen(Unknown, Private) || !MoreOpen(Private, Local) || MoreOpen(Local, Public) {
		t.Error("MoreOpen order wrong: want public > unknown > private > local")
	}
}
