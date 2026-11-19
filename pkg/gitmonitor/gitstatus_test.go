// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
)

func TestParsePorcelain(t *testing.T) {
	out := []byte(" M a.go\x00?? new dir/x.txt\x00R  new.go\x00old.go\x00D  gone.md\x00 T link\x00")
	st, err := parsePorcelain(out)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]git.StatusCode{
		"a.go":          {git.Unmodified, git.Modified},
		"new dir/x.txt": {git.Untracked, git.Untracked},
		"new.go":        {git.Renamed, git.Unmodified},
		"gone.md":       {git.Deleted, git.Unmodified},
		"link":          {git.Unmodified, git.Modified},
	}
	if len(st) != len(want) {
		t.Fatalf("parsed %d entries, want %d: %v", len(st), len(want), st)
	}
	for path, codes := range want {
		fs := st[path]
		if fs == nil || fs.Staging != codes[0] || fs.Worktree != codes[1] {
			t.Errorf("%s = %+v, want staging %q worktree %q", path, fs, codes[0], codes[1])
		}
	}
	if _, err := parsePorcelain([]byte("garbage")); err == nil {
		t.Error("parsePorcelain(garbage) succeeded")
	}
}

// statusRepo creates a repository with a committed LF file, then rewrites it
// with CRLF, changes another file for real and adds an untracked one.
func statusRepo(t *testing.T, autocrlf string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	if autocrlf != "" {
		gitRun(t, dir, "config", "core.autocrlf", autocrlf)
	}
	writeFile(t, filepath.Join(dir, "crlf.txt"), "one\ntwo\nthree\n")
	writeFile(t, filepath.Join(dir, "real.go"), "package a\n")
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-q", "-m", "init")
	writeFile(t, filepath.Join(dir, "crlf.txt"), "one\r\ntwo\r\nthree\r\n")
	writeFile(t, filepath.Join(dir, "real.go"), "package b\n")
	writeFile(t, filepath.Join(dir, "new.md"), "new\n")
	return dir
}

func TestScanSeesLineEndingOnlyChanges(t *testing.T) {
	dir := statusRepo(t, "")
	st, err := NewScanner().Scan("r", dir)
	if err != nil || st.ErrorMsg != "" {
		t.Fatalf("Scan: %v %s", err, st.ErrorMsg)
	}
	if st.IsClean || st.EOLOnlyCount != 1 || st.ModifiedCount != 2 {
		t.Errorf("IsClean=%v EOLOnlyCount=%d ModifiedCount=%d; want false, 1 (crlf.txt), 2 (real.go, new.md)",
			st.IsClean, st.EOLOnlyCount, st.ModifiedCount)
	}
	if !reflect.DeepEqual(st.ChangedFiles, map[string]int{".go": 1, ".md": 1}) {
		t.Errorf("ChangedFiles = %v; the CRLF-only file must not be listed", st.ChangedFiles)
	}
}

func TestGitAndGoGitAgree(t *testing.T) {
	dir := statusRepo(t, "")
	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatal(err)
	}
	wt, _ := repo.Worktree()
	viaGit, err := gitCLIStatus(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	viaGoGit, err := wt.Status()
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range viaGoGit {
		got := viaGit[path]
		if got == nil || got.Staging != want.Staging || got.Worktree != want.Worktree {
			t.Errorf("%s: git %+v, go-git %+v", path, got, want)
		}
	}
	if len(viaGit) != len(viaGoGit) {
		t.Errorf("git reports %d files, go-git %d", len(viaGit), len(viaGoGit))
	}
}

// With core.autocrlf=true git still lists a rewritten CRLF file as modified
// (its stat data changed), although `git diff` is empty. gitpulse reports it
// as a line-ending change, not as DIRTY: true, and harmless there.
func TestAutocrlfRepositoryShowsLineEndingsOnly(t *testing.T) {
	dir := statusRepo(t, "true")
	st, err := NewScanner().Scan("r", dir)
	if err != nil || st.ErrorMsg != "" {
		t.Fatalf("Scan: %v %s", err, st.ErrorMsg)
	}
	if st.EOLOnlyCount != 1 || st.ModifiedCount != 2 {
		t.Errorf("EOLOnlyCount=%d ModifiedCount=%d; want 1, 2", st.EOLOnlyCount, st.ModifiedCount)
	}
}

func TestGitStatusDoesNotWriteTheIndex(t *testing.T) {
	dir := statusRepo(t, "")
	index := filepath.Join(dir, ".git", "index")
	before, _ := os.Stat(index)
	if _, err := gitCLIStatus(dir, true); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(index)
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("git status rewrote .git/index; gitpulse must only read")
	}
}

func TestGitBranchesMatchGoGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	origin := filepath.Join(root, "origin")
	gitRun(t, root, "init", "-q", "--bare", origin)
	gitRun(t, root, "clone", "-q", origin, "a")
	a := filepath.Join(root, "a")
	writeFile(t, filepath.Join(a, "x"), "1\n")
	gitRun(t, a, "add", ".")
	gitRun(t, a, "commit", "-q", "-m", "one")
	gitRun(t, a, "push", "-q", "origin", "HEAD:main")
	gitRun(t, a, "branch", "-q", "--set-upstream-to=origin/main")
	gitRun(t, root, "clone", "-q", origin, "b")
	b := filepath.Join(root, "b")
	writeFile(t, filepath.Join(b, "y"), "2\n")
	gitRun(t, b, "add", ".")
	gitRun(t, b, "commit", "-q", "-m", "upstream news")
	gitRun(t, b, "push", "-q", "origin", "HEAD:main")
	writeFile(t, filepath.Join(a, "z"), "3\n")
	gitRun(t, a, "add", ".")
	gitRun(t, a, "commit", "-q", "-m", "local")
	gitRun(t, a, "checkout", "-q", "-b", "feature")
	gitRun(t, a, "checkout", "-q", "main")
	gitRun(t, a, "fetch", "-q")

	repo, err := git.PlainOpen(a)
	if err != nil {
		t.Fatal(err)
	}
	head, _ := repo.Head()
	s := &scanner{}
	viaGoGit := &RepoStatus{}
	viaGoGit.LocalBranches, _ = s.collectBranchDetails(repo)
	s.trackHead(repo, head, viaGoGit)
	viaGit := &RepoStatus{Path: a}
	s.branchState(repo, head, viaGit)

	if !reflect.DeepEqual(branchMap(viaGit.LocalBranches), branchMap(viaGoGit.LocalBranches)) {
		t.Errorf("branches: git %+v, go-git %+v", viaGit.LocalBranches, viaGoGit.LocalBranches)
	}
	for _, f := range []struct {
		name     string
		git, gog any
	}{
		{"ahead", viaGit.Ahead, viaGoGit.Ahead}, {"behind", viaGit.Behind, viaGoGit.Behind},
		{"tracking", viaGit.RemoteBranchName, viaGoGit.RemoteBranchName},
		{"incoming", viaGit.IncomingCommits, viaGoGit.IncomingCommits},
	} {
		if !reflect.DeepEqual(f.git, f.gog) {
			t.Errorf("%s: git %v, go-git %v", f.name, f.git, f.gog)
		}
	}
	if viaGit.Ahead != 1 || viaGit.Behind != 1 {
		t.Errorf("fixture: ahead=%d behind=%d, want 1 and 1", viaGit.Ahead, viaGit.Behind)
	}
}

// branchMap keys branches by name, with the commit time cut to seconds.
func branchMap(bs []BranchInfo) map[string]BranchInfo {
	m := map[string]BranchInfo{}
	for _, b := range bs {
		b.LastCommitAt = b.LastCommitAt.Truncate(time.Second).Local()
		m[b.Name] = b
	}
	return m
}

// After a merge upstream, the go-git fallback must count both sides of the
// merge (it used to stop at the first visit of the merge base).
func TestBehindCountsBothSidesOfAMerge(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	commit := func(name string) {
		writeFile(t, filepath.Join(dir, name), name+"\n")
		gitRun(t, dir, "add", ".")
		gitRun(t, dir, "commit", "-q", "-m", name)
	}
	commit("base")
	gitRun(t, dir, "branch", "-q", "local")
	gitRun(t, dir, "checkout", "-q", "-b", "x")
	commit("x1")
	commit("x2")
	gitRun(t, dir, "checkout", "-q", "-b", "y", "local")
	commit("y1")
	commit("y2")
	commit("y3")
	gitRun(t, dir, "checkout", "-q", "x")
	gitRun(t, dir, "merge", "-q", "--no-edit", "y")

	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatal(err)
	}
	local, _ := repo.Reference("refs/heads/local", true)
	tip, _ := repo.Reference("refs/heads/x", true)
	s := &scanner{}
	ahead, behind, err := s.aheadBehind(repo, local.Hash(), tip.Hash())
	if err != nil || ahead != 0 || behind != 6 {
		t.Errorf("aheadBehind = %d, %d, %v; want 0, 6 (x1, x2, y1, y2, y3, merge)", ahead, behind, err)
	}
	if got := s.incoming(repo, local.Hash(), tip.Hash()); len(got) != 6 {
		t.Errorf("incoming = %v, want 6 subjects", got)
	}
}
