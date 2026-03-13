// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary INI file
	tmpFile, err := os.CreateTemp("", "repos-*.ini")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `
[projects]
test1 = /path/to/test1
test2 = /path/to/test2
test3 = /path/to/test3/.git

[settings]
column_name_width = 30
show_summary = false
show_icons = false
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test LoadConfig
	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(cfg.Projects) != 3 {
		t.Errorf("expected 3 projects, got %d", len(cfg.Projects))
	}

	if cfg.Projects["test1"] != "/path/to/test1" {
		t.Errorf("expected path '/path/to/test1', got %s", cfg.Projects["test1"])
	}

	if cfg.Projects["test3"] != "/path/to/test3" {
		t.Errorf("expected path '/path/to/test3' for test3 (corrected from .git), got %s", cfg.Projects["test3"])
	}

	// Test Settings
	if cfg.Settings.ColumnNameWidth != 30 {
		t.Errorf("expected ColumnNameWidth 30, got %d", cfg.Settings.ColumnNameWidth)
	}
}

func TestManager_CRUD(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "crud-*.ini")
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	m := NewManager(tmpFile.Name())

	// 1. Add
	err := m.AddProject("new-proj", "/path/to/new")
	if err != nil {
		t.Fatalf("AddProject failed: %v", err)
	}

	cfg, _ := m.Load()
	if cfg.Projects["new-proj"] != "/path/to/new" {
		t.Errorf("expected /path/to/new, got %s", cfg.Projects["new-proj"])
	}

	// 2. Add with .git
	m.AddProject("git-proj", "/path/to/git/.git")
	cfg, _ = m.Load()
	if cfg.Projects["git-proj"] != "/path/to/git" {
		t.Errorf("expected /path/to/git, got %s", cfg.Projects["git-proj"])
	}

	// 3. Toggle Disable
	err = m.ToggleProject("new-proj", false)
	if err != nil {
		t.Fatalf("ToggleProject disable failed: %v", err)
	}
	cfg, _ = m.Load()
	if _, ok := cfg.Projects["new-proj"]; ok {
		t.Errorf("new-proj should be removed from active projects")
	}

	// 4. Toggle Enable
	err = m.ToggleProject("new-proj", true)
	if err != nil {
		t.Fatalf("ToggleProject enable failed: %v", err)
	}
	cfg, _ = m.Load()
	if cfg.Projects["new-proj"] != "/path/to/new" {
		t.Errorf("new-proj should be back in active projects")
	}

	// 5. Remove
	err = m.RemoveProject("new-proj")
	if err != nil {
		t.Fatalf("RemoveProject failed: %v", err)
	}
	cfg, _ = m.Load()
	if _, ok := cfg.Projects["new-proj"]; ok {
		t.Errorf("new-proj should be completely removed")
	}
}
