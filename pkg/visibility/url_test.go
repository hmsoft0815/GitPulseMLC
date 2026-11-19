// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package visibility

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		raw  string
		want Endpoint
	}{
		{"https://github.com/owner/repo.git", Endpoint{"https", "github.com", "owner/repo.git"}},
		{"https://TOKEN@GitHub.com/owner/repo", Endpoint{"https", "github.com", "owner/repo"}},
		{"ssh://git@gitlab.example.com:2222/group/repo.git", Endpoint{"ssh", "gitlab.example.com", "group/repo.git"}},
		{"git@github-work:owner/repo.git", Endpoint{"ssh", "github-work", "owner/repo.git"}},
		{"mlc@nas.local:/volume1/repos/x.git", Endpoint{"ssh", "nas.local", "volume1/repos/x.git"}},
		{"server:repos/x.git", Endpoint{"ssh", "server", "repos/x.git"}},
		{"/srv/git/x.git", Endpoint{"file", "", "/srv/git/x.git"}},
		{"../x.git", Endpoint{"file", "", "../x.git"}},
		{`C:\repos\x.git`, Endpoint{"file", "", `C:\repos\x.git`}},
		{"file:///srv/git/x.git", Endpoint{"file", "", "srv/git/x.git"}},
	}
	for _, tt := range tests {
		if got := Parse(tt.raw); got != tt.want {
			t.Errorf("Parse(%q) = %+v, want %+v", tt.raw, got, tt.want)
		}
	}
}

func TestSanitize(t *testing.T) {
	tests := map[string]string{
		"https://user:secret@github.com/o/r.git": "https://github.com/o/r.git",
		"https://ghp_token@github.com/o/r.git":   "https://github.com/o/r.git",
		"ssh://git:pw@host/r.git":                "ssh://git@host/r.git",
		"git@github.com:o/r.git":                 "git@github.com:o/r.git",
		"/srv/git/r.git":                         "/srv/git/r.git",
	}
	for raw, want := range tests {
		if got := Sanitize(raw); got != want {
			t.Errorf("Sanitize(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestShortHost(t *testing.T) {
	tests := map[string]string{"github.com": "github", "gitlab.com": "gitlab", "git.sr.ht": "sr", "mlcgo.eu": "mlcgo.eu"}
	for host, want := range tests {
		if got := ShortHost(host); got != want {
			t.Errorf("ShortHost(%q) = %q, want %q", host, got, want)
		}
	}
}
