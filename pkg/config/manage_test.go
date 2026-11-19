// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) *Manager {
	t.Helper()
	path := filepath.Join(t.TempDir(), "repos.ini")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return NewManager(path)
}

func TestMoveKeepsCommentsAndPaths(t *testing.T) {
	m := writeConfig(t, `[projects]
app = /code/app
# old = /code/old
whisper = /code/whisper
web = /code/web
`)
	if err := m.Move("whisper", SectionThirdParty); err != nil {
		t.Fatal(err)
	}
	cfg, err := m.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ThirdParty["whisper"] || cfg.ThirdParty["app"] || cfg.Projects["whisper"] != filepath.FromSlash("/code/whisper") {
		t.Errorf("after Move: ThirdParty=%v Projects=%v", cfg.ThirdParty, cfg.Projects)
	}
	data, _ := os.ReadFile(m.Path)
	text := string(data)
	if !strings.Contains(text, "# old = /code/old") {
		t.Errorf("comment above the moved key was lost:\n%s", text)
	}
	if strings.Index(text, "# old") > strings.Index(text, "[third_party]") {
		t.Errorf("comment moved out of [projects]:\n%s", text)
	}
	if err := m.Move("nothing", SectionIgnored); !errors.Is(err, ErrNotFound) {
		t.Errorf("Move(unknown) = %v, want ErrNotFound", err)
	}
}

func TestDuplicateNamesSurviveWrites(t *testing.T) {
	m := writeConfig(t, `[projects]
rag = /code/rag
rag = /code/other/rag
app = /code/app
`)
	if err := m.Move("app", SectionIgnored); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(m.Path)
	if !strings.Contains(string(data), "/code/rag") || !strings.Contains(string(data), "/code/other/rag") {
		t.Errorf("a duplicated project was dropped on write:\n%s", data)
	}
	if err := m.Move("rag", SectionThirdParty); err == nil {
		t.Error("Move of a duplicated name succeeded, want an error")
	}
	_, paths, err := m.Known()
	if err != nil || !paths[filepath.FromSlash("/code/rag")] || !paths[filepath.FromSlash("/code/other/rag")] {
		t.Errorf("Known() paths = %v, %v; want both rag paths", paths, err)
	}
}

func TestLookupByNameAndPath(t *testing.T) {
	dir := t.TempDir()
	m := writeConfig(t, "[projects]\napp = "+dir+"\n[disabled_projects]\nold = /code/old\n")
	if e, err := m.Lookup("old"); err != nil || e.Section != SectionIgnored {
		t.Errorf("Lookup(name) = %+v, %v", e, err)
	}
	if e, err := m.Lookup(dir + string(filepath.Separator)); err != nil || e.Name != "app" {
		t.Errorf("Lookup(path) = %+v, %v", e, err)
	}
	if _, err := m.Lookup("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Lookup(missing) = %v, want ErrNotFound", err)
	}
}
