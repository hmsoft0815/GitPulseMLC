// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/discover"
)

// runInit implements `gitpulse init [flags] [DIR ...]`: it finds Git
// repositories below each DIR and adds the new ones to the config file.
func runInit(args []string, globalConfig string, out io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	configFlag := fs.String("config", globalConfig, "Config file to create or extend")
	depth := fs.Int("depth", discover.DefaultMaxDepth, "How many directory levels to search")
	dryRun := fs.Bool("dry-run", false, "Only list what would be added; write nothing")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: gitpulse init [flags] [DIR ...]\n\n")
		fmt.Fprintf(fs.Output(), "Finds Git repositories below each DIR (default: current directory)\n")
		fmt.Fprintf(fs.Output(), "and adds them to your config. Run it again to add more folders.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	roots := fs.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}
	configPath := initTarget(*configFlag)
	manager := config.NewManager(configPath)

	names, paths, err := manager.Known()
	if err != nil {
		return err
	}

	var newPaths []string
	total := 0
	for _, root := range roots {
		fmt.Fprintf(out, "Searching %s (depth %d) ...\n", root, *depth)
		found, err := discover.Find(root, *depth)
		if err != nil {
			return err
		}
		total += len(found)
		for _, p := range found {
			if !paths[p] {
				paths[p] = true
				newPaths = append(newPaths, p)
			}
		}
	}
	return saveFound(out, manager, discover.Name(newPaths, names), total, *dryRun)
}

// saveFound reports the found repositories and writes the new ones to the config.
func saveFound(out io.Writer, m *config.Manager, repos []discover.Repo, total int, dryRun bool) error {
	home, _ := os.UserHomeDir()
	if total == 0 {
		fmt.Fprintf(out, "\nNo Git repositories found. Try a parent folder or a larger --depth.\n")
		return nil
	}
	fmt.Fprintf(out, "\nFound %d repositories, %d new.\n", total, len(repos))
	if len(repos) == 0 {
		fmt.Fprintf(out, "Everything is already in %s.\n", tildePath(m.Path, home))
		return nil
	}

	width := 0
	for _, r := range repos {
		width = max(width, len(r.Name))
	}
	projects := make([]config.Project, 0, len(repos))
	for _, r := range repos {
		fmt.Fprintf(out, "  + %-*s  %s\n", width, r.Name, tildePath(r.Path, home))
		projects = append(projects, config.Project{Name: r.Name, Path: r.Path})
	}

	if dryRun {
		fmt.Fprintf(out, "\nDry run: nothing written to %s.\n", tildePath(m.Path, home))
		return nil
	}
	if err := m.AddProjects(projects, home); err != nil {
		return err
	}
	fmt.Fprintf(out, "\nSaved to %s\n", tildePath(m.Path, home))
	fmt.Fprintf(out, "Next: run `gitpulse` — or `gitpulse --compact` to see only what needs attention.\n")
	return nil
}

// initTarget picks the config file init writes to: an explicit path, else the
// config gitpulse would read today, else ~/.gitpulsemlc/repos.ini.
func initTarget(explicit string) string {
	if path := findConfig(explicit); path != "" {
		return path
	}
	return defaultConfigPath()
}

// defaultConfigPath is the per-user config file, falling back to the
// current directory if the home directory is unknown.
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("config", "repos.ini")
	}
	return filepath.Join(home, ".gitpulsemlc", "repos.ini")
}

// tildePath shortens a path below home to ~/... for display.
func tildePath(path, home string) string {
	if home == "" {
		return path
	}
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.Join("~", rel)
	}
	return path
}
