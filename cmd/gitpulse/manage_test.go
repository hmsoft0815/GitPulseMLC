// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
)

// newRepo creates a repository below root with an optional origin URL.
func newRepo(t *testing.T, root, name, origin string) string {
	t.Helper()
	path := filepath.Join(root, name)
	repo, err := git.PlainInit(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if origin != "" {
		if _, err := repo.CreateRemote(&gitconfig.RemoteConfig{Name: "origin", URLs: []string{origin}}); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func sectionsOf(t *testing.T, cfgPath string) map[string]string {
	t.Helper()
	entries, err := config.NewManager(cfgPath).Entries()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range entries {
		got[e.Name] = e.Section
	}
	return got
}

func TestProjectCommands(t *testing.T) {
	root := t.TempDir()
	newRepo(t, root, "mine", "https://github.com/me/mine.git")
	newRepo(t, root, "lan", "nas.local:/repos/lan.git")
	newRepo(t, root, "solo", "")
	upstream := newRepo(t, root, "upstream", "https://github.com/someone/upstream.git")
	other := newRepo(t, t.TempDir(), "other", "")

	cfgPath := filepath.Join(t.TempDir(), "repos.ini")
	if err := os.WriteFile(cfgPath, []byte("[settings]\nown_remotes = github.com/me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer

	if err := runAdd([]string{root}, cfgPath, &out); err != nil {
		t.Fatalf("add: %v\n%s", err, out.String())
	}
	want := map[string]string{"mine": "projects", "lan": "projects", "solo": "projects", "upstream": "third_party"}
	for name, section := range want {
		if got := sectionsOf(t, cfgPath)[name]; got != section {
			t.Errorf("after add: %s in %q, want %q", name, got, section)
		}
	}

	steps := []struct {
		name string
		run  func() error
		repo string
		want string
	}{
		{"ignore by path", func() error { return runIgnore([]string{upstream}, cfgPath, &out) }, "upstream", config.SectionIgnored},
		{"mark ignored fails", func() error { return expectErr(runMark([]string{"upstream", "own"}, cfgPath, &out)) }, "upstream", config.SectionIgnored},
		{"watch classifies again", func() error { return runWatch([]string{"upstream"}, cfgPath, &out) }, "upstream", config.SectionThirdParty},
		{"mark own, flags after args", func() error { return runMark([]string{"upstream", "own"}, cfgPath, &out) }, "upstream", config.SectionOwn},
		{"mark --auto", func() error { return runMark([]string{"--auto"}, cfgPath, &out) }, "upstream", config.SectionThirdParty},
		{"ignore unknown repo", func() error { return runIgnore([]string{other}, cfgPath, &out) }, "other", config.SectionIgnored},
		{"add skips ignored", func() error { return runAdd([]string{other}, cfgPath, &out) }, "other", config.SectionIgnored},
		{"add --3rd after path", func() error { return runAdd([]string{filepath.Join(root, "solo"), "--3rd"}, cfgPath, &out) }, "solo", config.SectionOwn},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			t.Fatalf("%s: %v\n%s", step.name, err, out.String())
		}
		if got := sectionsOf(t, cfgPath)[step.repo]; got != step.want {
			t.Errorf("%s: %s in %q, want %q", step.name, step.repo, got, step.want)
		}
	}

	out.Reset()
	if err := runList(nil, cfgPath, &out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("3 own, 1 third party, 1 ignored.")) {
		t.Errorf("list output:\n%s", out.String())
	}
}

// expectErr turns an expected error into success and a nil error into one.
func expectErr(err error) error {
	if err == nil {
		return os.ErrInvalid
	}
	return nil
}
