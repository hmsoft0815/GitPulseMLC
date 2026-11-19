// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitRun runs git in dir with a fixed identity and fails the test on error.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	base := []string{"-c", "user.name=t", "-c", "user.email=t@example.com",
		"-c", "commit.gpgsign=false", "-c", "protocol.file.allow=always", "-c", "init.defaultBranch=main"}
	cmd := exec.Command("git", append(base, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newParentWithSubmodule builds a parent repository with one submodule at
// ".mlcai" and returns the parent's path.
func newParentWithSubmodule(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	sub := filepath.Join(root, "docs-origin")
	parent := filepath.Join(root, "parent")
	for _, dir := range []string{sub, parent} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		gitRun(t, dir, "init", "-q")
	}
	gitRun(t, sub, "config", "receive.denyCurrentBranch", "updateInstead")
	writeFile(t, filepath.Join(sub, "README.md"), "docs\n")
	gitRun(t, sub, "add", ".")
	gitRun(t, sub, "commit", "-q", "-m", "docs")

	writeFile(t, filepath.Join(parent, "main.go"), "package main\n")
	gitRun(t, parent, "add", ".")
	gitRun(t, parent, "commit", "-q", "-m", "code")
	gitRun(t, parent, "submodule", "add", "-q", sub, ".mlcai")
	gitRun(t, parent, "commit", "-q", "-m", "add submodule")
	return parent
}

func scanPath(t *testing.T, path string) *RepoStatus {
	t.Helper()
	st, err := NewScanner().Scan("parent", path)
	if err != nil || st.ErrorMsg != "" {
		t.Fatalf("Scan: %v %s", err, st.ErrorMsg)
	}
	if len(st.Submodules) != 1 {
		t.Fatalf("Submodules = %d, want 1", len(st.Submodules))
	}
	return st
}

func TestSubmodulePointerIsNotDirty(t *testing.T) {
	parent := newParentWithSubmodule(t)
	subDir := filepath.Join(parent, ".mlcai")

	st := scanPath(t, parent)
	if !st.IsClean || st.Submodules[0].PointerChanged() {
		t.Fatalf("fresh: IsClean=%v PointerChanged=%v, want clean and unchanged", st.IsClean, st.Submodules[0].PointerChanged())
	}

	// Commit inside the submodule: unpushed work needs action.
	writeFile(t, filepath.Join(subDir, "PLAN.md"), "plan\n")
	gitRun(t, subDir, "add", ".")
	gitRun(t, subDir, "commit", "-q", "-m", "plan")
	if st = scanPath(t, parent); !st.Submodules[0].NeedsAction() {
		t.Error("unpushed: NeedsAction() = false, want true")
	}

	// Once pushed, only the parent's pointer lags behind: a hint, not dirty.
	gitRun(t, subDir, "push", "-q")
	st = scanPath(t, parent)
	sub := st.Submodules[0]
	if !st.IsClean || st.ModifiedCount != 0 {
		t.Errorf("pointer only: IsClean=%v ModifiedCount=%d, want clean", st.IsClean, st.ModifiedCount)
	}
	if sub.PointerAhead != 1 || sub.PointerBehind != 0 || sub.NeedsAction() {
		t.Errorf("pointer only: ahead=%d behind=%d needsAction=%v, want 1 0 false", sub.PointerAhead, sub.PointerBehind, sub.NeedsAction())
	}
	if action, pointers := st.SubmoduleAttention(); len(action) != 0 || pointers != 1 {
		t.Errorf("SubmoduleAttention() = %v, %d; want none, 1", action, pointers)
	}

	// Staging the new pointer makes it a change of the parent.
	gitRun(t, parent, "add", ".mlcai")
	st = scanPath(t, parent)
	if st.IsClean || !st.Submodules[0].PointerStaged || st.ChangedFiles["(submodule)"] != 1 {
		t.Errorf("staged: IsClean=%v PointerStaged=%v ChangedFiles=%v", st.IsClean, st.Submodules[0].PointerStaged, st.ChangedFiles)
	}
}

func TestDirtySubmoduleNeedsAction(t *testing.T) {
	parent := newParentWithSubmodule(t)
	writeFile(t, filepath.Join(parent, ".mlcai", "README.md"), "changed\n")

	st := scanPath(t, parent)
	if !st.Submodules[0].NeedsAction() {
		t.Error("NeedsAction() = false for a submodule with uncommitted changes")
	}
	if action, _ := st.SubmoduleAttention(); len(action) != 1 || action[0] != ".mlcai" {
		t.Errorf("SubmoduleAttention() action = %v, want [.mlcai]", action)
	}
}
