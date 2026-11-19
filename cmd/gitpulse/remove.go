// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

// runRemove implements `gitpulse remove NAME|PATH ...` and
// `gitpulse remove --missing`.
func runRemove(args []string, globalConfig string, out io.Writer) error {
	fs := newFlagSet("remove", "remove [flags] NAME|PATH ...\n       gitpulse remove --missing [--dry-run]",
		"Forgets projects: they leave the config completely. Nothing on disk is touched.\n"+
			"Unlike `ignore`, a later `gitpulse add` of a parent folder finds them again.\n"+
			"--missing forgets every project whose folder is gone or holds no Git repository.")
	configFlag := fs.String("config", globalConfig, "Config file to change")
	missing := fs.Bool("missing", false, "Forget all projects that no longer exist")
	dryRun := fs.Bool("dry-run", false, "Only list what would be removed")
	targets, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if !*missing && len(targets) == 0 {
		fs.Usage()
		return errors.New("name or path missing (or use --missing)")
	}
	m, err := openConfig(*configFlag)
	if err != nil {
		return err
	}

	var doomed []config.Entry
	if *missing {
		doomed, err = missingEntries(out, m)
	} else {
		doomed, err = lookupAll(m, targets)
	}
	if err != nil || len(doomed) == 0 {
		return err
	}
	return removeEntries(out, m, doomed, *dryRun)
}

// missingEntries lists the projects, in every section, that no longer exist.
func missingEntries(out io.Writer, m *config.Manager) ([]config.Entry, error) {
	entries, err := m.Entries()
	if err != nil {
		return nil, err
	}
	var gone []config.Entry
	for _, e := range entries {
		if reason := gitmonitor.Gone(e.Path); reason != "" {
			gone = append(gone, e)
			fmt.Fprintf(out, "  %-24s %s (%s)\n", e.Name, e.Path, reason)
		}
	}
	if len(gone) == 0 {
		fmt.Fprintf(out, "Every project in %s still exists.\n", m.Path)
	}
	return gone, nil
}

// lookupAll finds every target by name or path.
func lookupAll(m *config.Manager, targets []string) ([]config.Entry, error) {
	entries := make([]config.Entry, 0, len(targets))
	for _, target := range targets {
		e, err := m.Lookup(target)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// removeEntries deletes the entries from the config unless dryRun is set.
func removeEntries(out io.Writer, m *config.Manager, entries []config.Entry, dryRun bool) error {
	if dryRun {
		fmt.Fprintf(out, "\nDry run: nothing removed from %s.\n", m.Path)
		return nil
	}
	home, _ := os.UserHomeDir()
	for _, e := range entries {
		if err := m.Remove(e); err != nil {
			return err
		}
		fmt.Fprintf(out, "Removed %s (%s) from the config.\n", e.Name, tildePath(e.Path, home))
	}
	return nil
}
