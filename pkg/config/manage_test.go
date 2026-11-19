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

func TestRemove(t *testing.T) {
	m := writeConfig(t, `[projects]
app = /code/app
# keep me
gone = /code/gone
rag = /code/rag
rag = /code/other/rag
web = /code/web
`)
	for _, e := range []Entry{
		{Name: "gone", Path: "/code/gone", Section: SectionOwn},
		{Name: "rag", Path: "/code/rag", Section: SectionOwn},
	} {
		if err := m.Remove(e); err != nil {
			t.Fatalf("Remove(%s): %v", e.Name, err)
		}
	}
	entries, err := m.Entries()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name+"="+filepath.ToSlash(e.Path))
	}
	if strings.Join(got, " ") != "app=/code/app web=/code/web rag=/code/other/rag" &&
		strings.Join(got, " ") != "app=/code/app rag=/code/other/rag web=/code/web" {
		t.Errorf("after Remove: %v", got)
	}
	data, _ := os.ReadFile(m.Path)
	if !strings.Contains(string(data), "# keep me") {
		t.Errorf("comment above removed key lost:\n%s", data)
	}
	if err := m.Remove(Entry{Name: "app", Path: "/elsewhere", Section: SectionOwn}); !errors.Is(err, ErrNotFound) {
		t.Errorf("Remove with wrong path = %v, want ErrNotFound", err)
	}
}
