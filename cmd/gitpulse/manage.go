// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/discover"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
	"github.com/hmsoft0815/GitPulseMLC/pkg/ownership"
)

// newFlagSet returns a flag set for a project command with a usage text.
func newFlagSet(name, usage, about string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: gitpulse %s\n\n%s\n\nFlags:\n", usage, about)
		fs.PrintDefaults()
	}
	return fs
}

// openConfig returns a manager for the config in use, or explains how to
// create one.
func openConfig(explicit string) (*config.Manager, error) {
	path := findConfig(explicit)
	if path == "" {
		return nil, errors.New("no config yet — start with: gitpulse add <DIR>")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("config %s: %w — start with: gitpulse add <DIR>", path, err)
	}
	return config.NewManager(path), nil
}

// runIgnore implements `gitpulse ignore NAME|PATH ...`.
func runIgnore(args []string, globalConfig string, out io.Writer) error {
	fs := newFlagSet("ignore", "ignore [flags] NAME|PATH ...",
		"Stops watching the projects but remembers them, so `gitpulse add` will not\n"+
			"add them again. A repository that is not in the config yet is recorded as ignored.")
	configFlag := fs.String("config", globalConfig, "Config file to change")
	targets, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fs.Usage()
		return errors.New("name or path missing")
	}
	m, err := openConfig(*configFlag)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if err := ignoreOne(out, m, target); err != nil {
			return err
		}
	}
	return nil
}

// ignoreOne ignores one project, or records an unknown repository as ignored.
func ignoreOne(out io.Writer, m *config.Manager, target string) error {
	e, err := m.Lookup(target)
	if errors.Is(err, config.ErrNotFound) {
		return ignoreNew(out, m, target)
	}
	if err != nil {
		return err
	}
	if e.Section == config.SectionIgnored {
		fmt.Fprintf(out, "%s is already ignored.\n", e.Name)
		return nil
	}
	if err := m.Move(e.Name, config.SectionIgnored); err != nil {
		return err
	}
	fmt.Fprintf(out, "Ignored %s — `gitpulse watch %s` brings it back.\n", e.Name, e.Name)
	return nil
}

// ignoreNew records a repository that is not in the config yet as ignored.
func ignoreNew(out io.Writer, m *config.Manager, target string) error {
	found, err := discover.Find(target, 0)
	if err != nil || len(found) == 0 {
		return fmt.Errorf("%q is neither a project in %s nor a Git repository", target, m.Path)
	}
	names, _, err := m.Known()
	if err != nil {
		return err
	}
	repo := discover.Name(found, names)[0]
	home, _ := os.UserHomeDir()
	if err := m.AddProjects([]config.Project{{Name: repo.Name, Path: repo.Path, Section: config.SectionIgnored}}, home); err != nil {
		return err
	}
	fmt.Fprintf(out, "Ignored %s (%s) — `gitpulse add` will skip it.\n", repo.Name, tildePath(repo.Path, home))
	return nil
}

// runWatch implements `gitpulse watch NAME|PATH ...`.
func runWatch(args []string, globalConfig string, out io.Writer) error {
	fs := newFlagSet("watch", "watch [flags] NAME|PATH ...",
		"Watches ignored projects again, as own or third party depending on own_remotes.")
	configFlag := fs.String("config", globalConfig, "Config file to change")
	targets, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fs.Usage()
		return errors.New("name or path missing")
	}
	m, err := openConfig(*configFlag)
	if err != nil {
		return err
	}
	settings, err := settingsOf(m.Path)
	if err != nil {
		return err
	}
	classifier := &ownership.Classifier{OwnRemotes: settings.OwnRemotes, PrivateHosts: settings.PrivateHosts}
	for _, target := range targets {
		e, err := m.Lookup(target)
		if errors.Is(err, config.ErrNotFound) {
			return fmt.Errorf("%q is not in %s — add it with: gitpulse add %s", target, m.Path, target)
		}
		if err != nil {
			return err
		}
		if e.Section != config.SectionIgnored {
			fmt.Fprintf(out, "%s is already watched (%s).\n", e.Name, kindLabel(e.Section))
			continue
		}
		section := sectionOf(classifier, e.Path)
		if err := m.Move(e.Name, section); err != nil {
			return err
		}
		fmt.Fprintf(out, "Watching %s again (%s).\n", e.Name, kindLabel(section))
	}
	return nil
}

// runList implements `gitpulse list`.
func runList(args []string, globalConfig string, out io.Writer) error {
	fs := newFlagSet("list", "list [flags]", "Lists every project of the config: kind (own or 3rd), state and path.")
	configFlag := fs.String("config", globalConfig, "Config file to read")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	m, err := openConfig(*configFlag)
	if err != nil {
		return err
	}
	entries, err := m.Entries()
	if err != nil {
		return err
	}
	settings, err := settingsOf(m.Path)
	if err != nil {
		return err
	}
	printEntries(out, entries, settings)
	fmt.Fprintf(out, "\nConfig: %s\n", m.Path)
	if len(settings.OwnRemotes) == 0 {
		fmt.Fprintf(out, "Tip: set own_remotes in [settings] to sort other people's repositories into third party.\n")
	}
	return nil
}

// printEntries prints the projects as a table, with counts per state.
func printEntries(out io.Writer, entries []config.Entry, settings config.Settings) {
	width := len("NAME")
	for _, e := range entries {
		width = max(width, len(e.Name))
	}
	fmt.Fprintf(out, "%-*s  KIND  STATE    PATH\n", width, "NAME")
	counts := map[string]int{}
	remaining := map[string]int{} // occurrences of each name still to come
	anyDup, anyMissing := false, false
	for _, e := range entries {
		remaining[e.Name]++
	}
	for _, e := range entries {
		remaining[e.Name]--
		state := "watched"
		switch {
		case remaining[e.Name] > 0:
			state = "dup" // only the last entry of a name is used
			anyDup = true
		case gitmonitor.Gone(e.Path) != "":
			state = "missing"
			anyMissing = true
		case e.Section == config.SectionIgnored:
			state = "ignored"
		}
		counts[e.Section]++
		path := e.Path
		if settings.ReplacePathPrefix != "" {
			if rel, err := filepath.Rel(settings.ReplacePathPrefix, path); err == nil && filepath.IsLocal(rel) {
				path = filepath.Join(settings.ReplaceWith, rel)
			}
		}
		fmt.Fprintf(out, "%-*s  %-4s  %-7s  %s\n", width, e.Name, kindLabel(e.Section), state, path)
	}
	fmt.Fprintf(out, "\n%d own, %d third party, %d ignored.\n",
		counts[config.SectionOwn], counts[config.SectionThirdParty], counts[config.SectionIgnored])
	if anyMissing {
		fmt.Fprintf(out, "missing = the folder is gone or holds no Git repository — `gitpulse remove --missing` forgets them.\n")
	}
	if anyDup {
		fmt.Fprintf(out, "dup = the name occurs again further down; only the last entry is used — rename one.\n")
	}
}
