// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

func TestScanner_ScanRepo_InMemory(t *testing.T) {
	// 1. Create In-Memory Repo
	fs := memfs.New()
	st := memory.NewStorage()
	repo, err := git.Init(st, fs)
	if err != nil {
		t.Fatalf("failed to init in-memory repo: %v", err)
	}

	wt, _ := repo.Worktree()
	
	// Create initial commit
	file, _ := fs.Create("main.go")
	file.Write([]byte("package main"))
	file.Close()
	wt.Add("main.go")
	
	sig := &object.Signature{
		Name:  "Test User",
		Email: "test@example.com",
		When:  time.Now().Add(-40 * 24 * time.Hour), // Old commit to test staleness if needed
	}
	wt.Commit("initial commit", &git.CommitOptions{Author: sig})

	// 2. Test Scanner Logic
	s := &scanner{}
	status := &RepoStatus{Name: "test-memory", Path: ""}
	
	res, err := s.scanRepo(status, repo)
	if err != nil {
		t.Fatalf("scanRepo failed: %v", err)
	}

	if res.Name != "test-memory" {
		t.Errorf("expected name 'test-memory', got %v", res.Name)
	}

	if !res.IsClean {
		t.Errorf("expected repo to be clean")
	}

	if !res.HasNoRemote {
		t.Errorf("expected HasNoRemote to be true for in-memory repo without remotes")
	}

	// 3. Make it DIRTY
	file, _ = fs.Create("dirty.txt")
	file.Write([]byte("dirty content"))
	file.Close()
	// No wt.Add() -> Untracked
	
	res, _ = s.scanRepo(status, repo)
	if res.IsClean {
		t.Errorf("expected repo to be dirty (untracked files)")
	}
	if res.UntrackedCount != 1 {
		t.Errorf("expected 1 untracked file, got %d", res.UntrackedCount)
	}
	if res.ChangedFiles[".txt"] != 1 {
		t.Errorf("expected 1 .txt changed file, got %d", res.ChangedFiles[".txt"])
	}
}

func TestScanner_Staleness(t *testing.T) {
	fs := memfs.New()
	st := memory.NewStorage()
	repo, err := git.Init(st, fs)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	wt, _ := repo.Worktree()
	
	// Commit 40 days ago
	sig := &object.Signature{
		Name:  "Test",
		Email: "test@test.com",
		When:  time.Now().Add(-40 * 24 * time.Hour),
	}
	
	file, _ := fs.Create("old.txt")
	file.Close()
	wt.Add("old.txt")
	wt.Commit("old commit", &git.CommitOptions{Author: sig})

	// Create a feature branch
	h, _ := repo.Head()
	err = repo.CreateBranch(&config.Branch{Name: "feature/stale"})
	if err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	// We need to actually have a reference for it
	_ = repo.Storer.SetReference(plumbing.NewHashReference("refs/heads/feature/stale", h.Hash()))

	s := &scanner{}
	status := &RepoStatus{Name: "stale-test"}
	res, _ := s.scanRepo(status, repo)

	foundStale := false
	for _, b := range res.LocalBranches {
		if b.Name == "feature/stale" {
			if b.IsStale {
				foundStale = true
			}
		}
	}

	if !foundStale {
		t.Errorf("expected feature/stale to be marked as stale")
	}
}

func TestScanner_CountCommitsBetween(t *testing.T) {
	fs := memfs.New()
	st := memory.NewStorage()
	repo, _ := git.Init(st, fs)
	wt, _ := repo.Worktree()

	sig := &object.Signature{Name: "T", Email: "t", When: time.Now()}
	
	// Commit 1
	file1, _ := fs.Create("f1")
	file1.Close()
	wt.Add("f1")
	h1, _ := wt.Commit("c1", &git.CommitOptions{Author: sig})
	
	// Commit 2
	file2, _ := fs.Create("f2")
	file2.Close()
	wt.Add("f2")
	_, _ = wt.Commit("c2", &git.CommitOptions{Author: sig})
	
	// Commit 3
	file3, _ := fs.Create("f3")
	file3.Close()
	wt.Add("f3")
	h3, _ := wt.Commit("c3", &git.CommitOptions{Author: sig})

	s := &scanner{}
	
	// Between 1 and 3 should be 2 commits (c2, c3)
	count, err := s.countCommitsBetween(repo, h1, h3)
	if err != nil {
		t.Errorf("countCommitsBetween errored: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 commits between h1 and h3, got %d", count)
	}

	// Between 1 and 1 should be 0
	count, _ = s.countCommitsBetween(repo, h1, h1)
	if count != 0 {
		t.Errorf("expected 0 commits between h1 and h1, got %d", count)
	}
}

func TestScanner_AheadBehind(t *testing.T) {
	fs := memfs.New()
	st := memory.NewStorage()
	repo, _ := git.Init(st, fs)
	wt, _ := repo.Worktree()
	sig := &object.Signature{Name: "T", Email: "t", When: time.Now()}

	// 1. Initial shared commit
	f0, _ := fs.Create("base")
	f0.Close()
	wt.Add("base")
	base, _ := wt.Commit("base", &git.CommitOptions{Author: sig})

	// 2. Local commit (ahead) on master
	f1, _ := fs.Create("local")
	f1.Close()
	wt.Add("local")
	_, _ = wt.Commit("local", &git.CommitOptions{Author: sig})

	// 3. Mock a remote branch (behind)
	err := repo.CreateBranch(&config.Branch{Name: "temp-remote"})
	if err != nil { t.Fatalf("create branch failed: %v", err) }
	
	err = repo.Storer.SetReference(plumbing.NewHashReference("refs/heads/temp-remote", base))
	if err != nil { t.Fatalf("set reference failed: %v", err) }
	
	err = wt.Checkout(&git.CheckoutOptions{Branch: "refs/heads/temp-remote"})
	if err != nil { t.Fatalf("checkout failed: %v", err) }
	
	f2, _ := fs.Create("behind1")
	f2.Close()
	wt.Add("behind1")
	_, _ = wt.Commit("behind1", &git.CommitOptions{Author: sig})
	
	f3, _ := fs.Create("behind2")
	f3.Close()
	wt.Add("behind2")
	behind2, _ := wt.Commit("behind2", &git.CommitOptions{Author: sig})
	
	// Set origin/master to behind2
	err = repo.Storer.SetReference(plumbing.NewHashReference("refs/remotes/origin/master", behind2))
	if err != nil { t.Fatalf("set remote ref failed: %v", err) }
	
	// Back to master
	err = wt.Checkout(&git.CheckoutOptions{Branch: "refs/heads/master"})
	if err != nil { t.Fatalf("checkout master failed: %v", err) }
	
	// Configure tracking
	c, _ := repo.Config()
	c.Branches["master"] = &config.Branch{
		Name:   "master",
		Remote: "origin",
		Merge:  "refs/heads/master",
	}
	repo.SetConfig(c)

	s := &scanner{}
	ref, err := repo.Reference(plumbing.ReferenceName("refs/heads/master"), true)
	if err != nil {
		t.Fatalf("could not get master ref: %v", err)
	}
	
	ahead, behind, err := s.getAheadBehindForBranch(repo, ref)
	if err != nil {
		t.Fatalf("getAheadBehind failed: %v", err)
	}

	if ahead != 1 {
		t.Errorf("expected 1 commit ahead, got %d", ahead)
	}
	if behind != 2 {
		t.Errorf("expected 2 commits behind, got %d", behind)
	}
}
