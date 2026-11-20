// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/render"
	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

// fakeHost writes a script that prints the JSON of results like a remote
// gitpulse would, and records its arguments in args.txt.
func fakeHost(t *testing.T, results []*gitmonitor.RepoStatus) (config.Host, string) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := render.JSON(&buf, "remote", results, gitmonitor.Summary{Total: len(results)}); err != nil {
		t.Fatal(err)
	}
	jsonFile := filepath.Join(dir, "out.json")
	argsFile := filepath.Join(dir, "args.txt")
	if err := os.WriteFile(jsonFile, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "gitpulse")
	body := "#!/bin/sh\necho \"$@\" > " + argsFile + "\ncat " + jsonFile + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return config.Host{Name: "win11", Command: []string{"sh", script}}, argsFile
}

func TestScanHostReadsRemoteJSON(t *testing.T) {
	h, argsFile := fakeHost(t, []*gitmonitor.RepoStatus{
		{Name: "lightcraft", Path: "/mnt/c/development/lightcraft", EOLOnlyCount: 3, ChangedFiles: map[string]int{}},
		{Name: "whisper.cpp", Path: "/mnt/c/development/whisper.cpp", ThirdParty: true, IsClean: true},
	})
	got := startHosts([]config.Host{h}, true, 10*time.Second).wait(false)
	if len(got) != 2 || got[0].Host != "win11" || got[1].Host != "win11" || got[0].EOLOnlyCount != 3 || !got[1].ThirdParty {
		t.Fatalf("results = %+v", got)
	}
	args, _ := os.ReadFile(argsFile)
	if strings.TrimSpace(string(args)) != "--json --offline" {
		t.Errorf("remote got arguments %q, want --json --offline", args)
	}
}

func TestScanHostFailures(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	tests := []struct {
		name    string
		command []string
		timeout time.Duration
		want    string
	}{
		{"exit code", []string{"sh", "-c", "echo 'ssh: connect to host win11: No route' >&2; exit 255"}, 5 * time.Second, "No route"},
		{"missing program", []string{"/nonexistent/gitpulse"}, 5 * time.Second, "host win11:"},
		{"no JSON", []string{"sh", "-c", "echo hello"}, 5 * time.Second, "no gitpulse JSON"},
		{"timeout", []string{"sh", "-c", "sleep 5"}, 300 * time.Millisecond, "host_timeout"},
	}
	for _, tt := range tests {
		start := time.Now()
		got := startHosts([]config.Host{{Name: "win11", Command: tt.command}}, false, tt.timeout).wait(false)
		if len(got) != 1 || got[0].Host != "win11" || !strings.Contains(got[0].ErrorMsg, tt.want) {
			t.Errorf("%s: %+v, want one error containing %q", tt.name, got, tt.want)
		}
		if time.Since(start) > 3*time.Second {
			t.Errorf("%s: took %s — a failing host must not hold the overview", tt.name, time.Since(start))
		}
	}
}

func TestOrderAndRenderHosts(t *testing.T) {
	results := []*gitmonitor.RepoStatus{
		{Name: "remote-3rd", Host: "win11", ThirdParty: true, IsClean: true},
		{Name: "nas-repo", Host: "nas", IsClean: true},
		{Name: "remote-own", Host: "win11", IsClean: true},
		{Name: "local", IsClean: true},
	}
	orderResults(results, []config.Host{{Name: "win11"}, {Name: "nas"}})
	var names []string
	for _, r := range results {
		names = append(names, r.Name)
	}
	if got := strings.Join(names, " "); got != "local remote-own remote-3rd nas-repo" {
		t.Fatalf("order = %s", got)
	}

	lipgloss.SetColorProfile(termenv.Ascii)
	var buf bytes.Buffer
	render.TUI(&buf, results, gitmonitor.Summary{}, &config.Config{}, render.TUIOptions{})
	out := buf.String()
	iWin, iThird, iNas := strings.Index(out, "HOST win11"), strings.Index(out, "THIRD PARTY"), strings.Index(out, "HOST nas")
	if iWin < 0 || iNas < 0 || !(strings.Index(out, "local") < iWin && iWin < iThird && iThird < iNas) {
		t.Errorf("blocks in wrong order or missing:\n%s", out)
	}
}
