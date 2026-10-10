// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// Package discover finds Git repositories below a directory tree.
package discover

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultMaxDepth is how many directory levels below the root are searched.
const DefaultMaxDepth = 4

// skipDirs are directory names that never contain repositories worth watching.
var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
}

// keySafe replaces characters that would break an INI key.
var keySafe = strings.NewReplacer("=", "-", ":", "-", "[", "-", "]", "-", "#", "-", ";", "-")

// Repo is a Git repository found on disk.
type Repo struct {
	Name string // Suggested project name, unique within one result set.
	Path string // Absolute path to the working tree.
}

// Find walks root up to maxDepth levels deep and returns every Git working tree
// it finds. It does not descend into a repository once found, and skips hidden
// directories, node_modules and vendor. Unreadable directories are skipped silently.
func Find(root string, maxDepth int) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", root, err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	var found []string
	walkErr := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == absRoot {
				return err
			}
			return fs.SkipDir
		}
		if !d.IsDir() {
			return nil
		}
		if path != absRoot && (strings.HasPrefix(d.Name(), ".") || skipDirs[d.Name()]) {
			return fs.SkipDir
		}
		if isRepo(path) {
			found = append(found, path)
			return fs.SkipDir
		}
		if depth(absRoot, path) >= maxDepth {
			return fs.SkipDir
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("scan %s: %w", root, walkErr)
	}
	sort.Strings(found)
	return found, nil
}

// isRepo reports whether dir is a Git working tree. A .git file (worktrees,
// submodules) counts as well as a .git directory.
func isRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func depth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}

// Name assigns project names to paths: the directory's base name, prefixed with
// its parent when two repositories share a base name. taken holds names already
// in use (e.g. from an existing config); it is updated with the new names.
func Name(paths []string, taken map[string]bool) []Repo {
	baseCount := make(map[string]int)
	for _, p := range paths {
		baseCount[filepath.Base(p)]++
	}

	repos := make([]Repo, 0, len(paths))
	for _, p := range paths {
		name := filepath.Base(p)
		if baseCount[name] > 1 {
			name = filepath.Base(filepath.Dir(p)) + "-" + name
		}
		name = unique(keySafe.Replace(name), taken)
		taken[name] = true
		repos = append(repos, Repo{Name: name, Path: p})
	}
	return repos
}

func unique(name string, taken map[string]bool) string {
	if !taken[name] {
		return name
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", name, i)
		if !taken[candidate] {
			return candidate
		}
	}
}
