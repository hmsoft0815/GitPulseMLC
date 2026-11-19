// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package visibility

import "testing"

func TestIsLocalHost(t *testing.T) {
	private := []string{"git.example.com", "*.corp.example"}
	tests := []struct {
		host string
		want bool
	}{
		{"nas", true},
		{"nas.local", true},
		{"router.fritz.box", true},
		{"192.168.1.10", true},
		{"172.16.0.49", true},
		{"10.0.0.1", true},
		{"127.0.0.1", true},
		{"[::1]", true},
		{"git.example.com", true},
		{"build.corp.example", true},
		{"github.com", false},
		{"8.8.8.8", false},
		{"mlcgo.eu", false},
		{"corp.example", false},
	}
	for _, tt := range tests {
		if got := IsLocalHost(tt.host, private); got != tt.want {
			t.Errorf("IsLocalHost(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}

func TestParseSSHConfig(t *testing.T) {
	out := []byte("user git\nhostname github.com\nport 22\n")
	if got := parseSSHConfig(out, "github-work"); got != "github.com" {
		t.Errorf("parseSSHConfig() = %q, want github.com", got)
	}
	if got := parseSSHConfig([]byte("port 22\n"), "alias"); got != "alias" {
		t.Errorf("parseSSHConfig() without hostname = %q, want alias", got)
	}
}
