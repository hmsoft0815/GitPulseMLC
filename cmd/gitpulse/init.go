// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/discover"
	"github.com/hmsoft0815/GitPulseMLC/pkg/ownership"
)

// runAdd implements `gitpulse add [flags] [PATH ...]` (alias: init). A PATH
// that is a repository is added itself; any other folder is searched for
// repositories. Ignored projects are never added again.
func runAdd(args []string, globalConfig string, out io.Writer) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	configFlag := fs.String("config", globalConfig, "Config file to create or extend")
	depth := fs.Int("depth", discover.DefaultMaxDepth, "How many directory levels to search")
	dryRun := fs.Bool("dry-run", false, "Only list what would be added; write nothing")
	own := fs.Bool("own", false, "Add as your own projects, whatever their remote")
	third := fs.Bool("3rd", false, "Add as third party, whatever their remote")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: gitpulse add [flags] [PATH ...]\n\n")
		fmt.Fprintf(fs.Output(), "Adds the repository at PATH (default: current directory), or every\n")
		fmt.Fprintf(fs.Output(), "repository below PATH if it is a plain folder. Repositories whose remote\n")
		fmt.Fprintf(fs.Output(), "is not in own_remotes are added as third party.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	roots, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if *own && *third {
		return errors.New("--own and --3rd exclude each other")
	}
	if len(roots) == 0 {
		roots = []string{"."}
	}

	manager := config.NewManager(initTarget(*configFlag))
	settings, err := settingsOf(manager.Path)
	if err != nil {
		return err
	}
	names, paths, err := manager.Known()
	if err != nil {
		return err
	}

	newPaths, total, err := findNew(out, roots, *depth, paths)
	if err != nil {
		return err
	}

	classifier := &ownership.Classifier{OwnRemotes: settings.OwnRemotes, PrivateHosts: settings.PrivateHosts}
	force := forcedSection(*own, *third)
	projects := make([]config.Project, 0, len(newPaths))
	for _, r := range discover.Name(newPaths, names) {
		section := force
		if section == "" {
			section = sectionOf(classifier, r.Path)
		}
		projects = append(projects, config.Project{Name: r.Name, Path: r.Path, Section: section})
	}
	if err := saveFound(out, manager, projects, total, *dryRun); err != nil {
		return err
	}
	if len(settings.OwnRemotes) == 0 && force == "" {
		paths := make([]string, len(projects))
		for i, p := range projects {
			paths[i] = p.Path
		}
		suggestOwnRemotes(out, classifier, paths)
	}
	return nil
}

// findNew searches every root and returns the repositories not in known yet,
// plus how many were found in total.
func findNew(out io.Writer, roots []string, depth int, known map[string]bool) ([]string, int, error) {
	var newPaths []string
	total := 0
	for _, root := range roots {
		fmt.Fprintf(out, "Searching %s (depth %d) ...\n", root, depth)
		found, err := discover.Find(root, depth)
		if err != nil {
			return nil, 0, err
		}
		total += len(found)
		for _, p := range found {
			if !known[p] {
				known[p] = true
				newPaths = append(newPaths, p)
			}
		}
	}
	return newPaths, total, nil
}

// forcedSection returns the section --own or --3rd asks for, or "" to
// classify by remote.
func forcedSection(own, third bool) string {
	switch {
	case own:
		return config.SectionOwn
	case third:
		return config.SectionThirdParty
	}
	return ""
}

// saveFound prints the new projects and writes them to the config.
func saveFound(out io.Writer, m *config.Manager, projects []config.Project, total int, dryRun bool) error {
	home, _ := os.UserHomeDir()
	if total == 0 {
		fmt.Fprintf(out, "\nNo Git repositories found. Try a parent folder or a larger --depth.\n")
		return nil
	}
	fmt.Fprintf(out, "\nFound %d repositories, %d new.\n", total, len(projects))
	if len(projects) == 0 {
		fmt.Fprintf(out, "Everything is already in %s (watched or ignored).\n", tildePath(m.Path, home))
		return nil
	}

	width := 0
	for _, p := range projects {
		width = max(width, len(p.Name))
	}
	for _, p := range projects {
		fmt.Fprintf(out, "  + %-*s  %-3s  %s\n", width, p.Name, kindLabel(p.Section), tildePath(p.Path, home))
	}

	if dryRun {
		fmt.Fprintf(out, "\nDry run: nothing written to %s.\n", tildePath(m.Path, home))
		return nil
	}
	if err := m.AddProjects(projects, home); err != nil {
		return err
	}
	fmt.Fprintf(out, "\nSaved to %s\n", tildePath(m.Path, home))
	fmt.Fprintf(out, "Next: run `gitpulse`. `gitpulse ignore NAME` stops watching a project,\n")
	fmt.Fprintf(out, "      `gitpulse mark NAME 3rd` marks it as third party.\n")
	return nil
}

// sectionOf picks [projects] or [third_party] for a repository by its remotes.
func sectionOf(c *ownership.Classifier, repoPath string) string {
	urls, _ := ownership.RemoteURLs(repoPath)
	if c.Classify(urls) == ownership.ThirdParty {
		return config.SectionThirdParty
	}
	return config.SectionOwn
}

// kindLabel names a section in listings.
func kindLabel(section string) string {
	switch section {
	case config.SectionThirdParty:
		return "3rd"
	case config.SectionIgnored:
		return "—"
	default:
		return "own"
	}
}

// settingsOf reads the settings of an existing config; a missing file has
// default settings.
func settingsOf(path string) (config.Settings, error) {
	cfg, err := config.LoadConfig(path)
	if errors.Is(err, fs.ErrNotExist) {
		return config.Settings{}, nil
	}
	if err != nil {
		return config.Settings{}, err
	}
	return cfg.Settings, nil
}

// parseArgs parses flags that may stand before, between or after the
// positional arguments, as in `gitpulse add ~/code --3rd`.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return positional, nil
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
}

// initTarget returns the config to extend: the one in use, or the default
// location for a first run.
func initTarget(explicit string) string {
	if path := findConfig(explicit); path != "" {
		return path
	}
	return defaultConfigPath()
}

// defaultConfigPath is where a first run creates the config.
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("config", "repos.ini")
	}
	return filepath.Join(home, ".gitpulsemlc", "repos.ini")
}

// tildePath shortens a path below home to "~/...".
func tildePath(path, home string) string {
	if home == "" {
		return path
	}
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.Join("~", rel)
	}
	return path
}
