// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package ownership

import "testing"

func TestMatches(t *testing.T) {
	tests := []struct {
		pattern, key string
		want         bool
	}{
		{"github.com/me", "github.com/me/tool", true},
		{"GitHub.com/Me", "github.com/me/tool", true},
		{"github.com/me", "github.com/meyer/tool", false},
		{"github.com/me/", "github.com/me/tool", true},
		{"mlcgo.eu", "mlcgo.eu/home/git/x", true},
		{"*.example.com", "git.example.com/group/x", true},
		{"github.com/*", "github.com/anyone/x", true},
		{"github.com/me/tool/extra", "github.com/me/tool", false},
	}
	for _, tt := range tests {
		if got := Matches(tt.pattern, tt.key); got != tt.want {
			t.Errorf("Matches(%q, %q) = %v, want %v", tt.pattern, tt.key, got, tt.want)
		}
	}
}

func TestClassify(t *testing.T) {
	c := &Classifier{
		OwnRemotes: []string{"github.com/me", "mlcgo.eu"},
		ResolveSSH: func(h string) string { return map[string]string{"github-work": "github.com"}[h] },
	}
	tests := []struct {
		name string
		urls []string
		want Kind
	}{
		{"no remote", nil, Own},
		{"own forge", []string{"https://github.com/me/tool.git"}, Own},
		{"own via ssh alias", []string{"git@github-work:me/tool.git"}, Own},
		{"own server", []string{"git@mlcgo.eu:/home/git/x.git"}, Own},
		{"lan", []string{"mlc@nas.local:/volume1/x.git"}, Own},
		{"file", []string{"/srv/git/x.git"}, Own},
		{"foreign", []string{"https://github.com/ggerganov/whisper.cpp.git"}, ThirdParty},
		{"fork with own remote", []string{"https://github.com/upstream/x.git", "git@github.com:me/x.git"}, Own},
	}
	for _, tt := range tests {
		if got := c.Classify(tt.urls); got != tt.want {
			t.Errorf("%s: Classify(%v) = %v, want %v", tt.name, tt.urls, got, tt.want)
		}
	}

	none := &Classifier{}
	if got := none.Classify([]string{"https://github.com/ggerganov/whisper.cpp"}); got != Own {
		t.Errorf("without own_remotes: Classify = %v, want own", got)
	}
}

func TestOwnerKey(t *testing.T) {
	tests := map[string]string{
		"github.com/me/tool":    "github.com/me",
		"gitlab.com/grp/sub/x":  "gitlab.com/grp",
		"mlcgo.eu/home/git/x":   "mlcgo.eu",
		"git.example.com/grp/x": "git.example.com",
	}
	for key, want := range tests {
		if got := OwnerKey(key); got != want {
			t.Errorf("OwnerKey(%q) = %q, want %q", key, got, want)
		}
	}
}
