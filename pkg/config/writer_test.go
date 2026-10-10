// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddProjectsCreatesAndExtends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "repos.ini")
	m := NewManager(path)

	names, paths, err := m.Known()
	if err != nil || len(names) != 0 || len(paths) != 0 {
		t.Fatalf("Known() on missing file = %v, %v, %v; want empty, nil", names, paths, err)
	}

	if err := m.AddProjects([]Project{{"app", "/code/app"}}, "/home/me"); err != nil {
		t.Fatalf("AddProjects (create): %v", err)
	}
	if err := m.AddProjects([]Project{{"web", "/code/web#1"}}, "/ignored"); err != nil {
		t.Fatalf("AddProjects (extend): %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	want := map[string]string{
		"app": filepath.Clean("/code/app"),
		"web": filepath.Clean("/code/web#1"),
	}
	for name, p := range want {
		if cfg.Projects[name] != p {
			t.Errorf("project %s = %q, want %q", name, cfg.Projects[name], p)
		}
	}
	if cfg.Settings.ReplacePathPrefix != "/home/me" || cfg.Settings.ReplaceWith != "~" {
		t.Errorf("path replacement = %q→%q, want /home/me→~", cfg.Settings.ReplacePathPrefix, cfg.Settings.ReplaceWith)
	}
	if !cfg.Settings.ShowSummary || cfg.Settings.CompactMode {
		t.Errorf("default settings not written: %+v", cfg.Settings)
	}
}

func TestKnownIncludesDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repos.ini")
	content := "[projects]\na = /x/a\n\n[disabled_projects]\nb = /x/b/\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	names, paths, err := NewManager(path).Known()
	if err != nil {
		t.Fatal(err)
	}
	if !names["a"] || !names["b"] || !paths[filepath.Clean("/x/b")] {
		t.Errorf("Known() = %v, %v; want a, b and cleaned /x/b", names, paths)
	}
}
