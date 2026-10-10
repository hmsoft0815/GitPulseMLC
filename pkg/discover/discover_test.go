// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package discover

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// mkRepo creates dir and marks it as a Git working tree.
func mkRepo(t *testing.T, dir string, gitFile bool) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, ".git")
	var err error
	if gitFile {
		err = os.WriteFile(marker, []byte("gitdir: ../elsewhere\n"), 0o644)
	} else {
		err = os.Mkdir(marker, 0o755)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestFind(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "app"), false)
	mkRepo(t, filepath.Join(root, "app", "nested"), false)         // inside a repo: skipped
	mkRepo(t, filepath.Join(root, "work", "client", "api"), false) // depth 3
	mkRepo(t, filepath.Join(root, "worktree"), true)               // .git file
	mkRepo(t, filepath.Join(root, ".cache", "hidden"), false)      // hidden dir: skipped
	mkRepo(t, filepath.Join(root, "web", "node_modules", "dep"), false)
	mkRepo(t, filepath.Join(root, "a", "b", "c", "d", "deep"), false) // depth 5: skipped

	tests := []struct {
		name  string
		depth int
		want  []string
	}{
		{"default depth", DefaultMaxDepth, []string{"app", "work/client/api", "worktree"}},
		{"shallow", 1, []string{"app", "worktree"}},
		{"deep", 5, []string{"a/b/c/d/deep", "app", "work/client/api", "worktree"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Find(root, tt.depth)
			if err != nil {
				t.Fatal(err)
			}
			want := make([]string, len(tt.want))
			for i, rel := range tt.want {
				want[i] = filepath.Join(root, filepath.FromSlash(rel))
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Find() = %v, want %v", got, want)
			}
		})
	}
}

func TestFindRootIsRepo(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, root, false)
	got, err := Find(root, DefaultMaxDepth)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != root {
		t.Errorf("Find() = %v, want [%s]", got, root)
	}
}

func TestFindMissingRoot(t *testing.T) {
	if _, err := Find(filepath.Join(t.TempDir(), "nope"), 2); err == nil {
		t.Error("expected error for missing root")
	}
}

func TestName(t *testing.T) {
	sep := string(filepath.Separator)
	paths := []string{
		sep + filepath.Join("code", "api"),
		sep + filepath.Join("work", "api"),
		sep + filepath.Join("code", "web"),
		sep + filepath.Join("code", "a=b"),
	}
	taken := map[string]bool{"web": true}

	got := Name(paths, taken)
	want := []string{"code-api", "work-api", "web-2", "a-b"}
	for i, r := range got {
		if r.Name != want[i] {
			t.Errorf("Name(%s) = %q, want %q", r.Path, r.Name, want[i])
		}
	}
	if !taken["web-2"] {
		t.Error("taken was not updated with new names")
	}
}
