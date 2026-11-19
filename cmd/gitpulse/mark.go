// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/ownership"
)

// runMark implements `gitpulse mark NAME|PATH ... own|3rd` and
// `gitpulse mark --auto`.
func runMark(args []string, globalConfig string, out io.Writer) error {
	fs := newFlagSet("mark", "mark [flags] NAME|PATH ... own|3rd\n       gitpulse mark --auto [--dry-run]",
		"Marks projects as your own or third party. Third-party projects are listed\n"+
			"apart; local changes there are hints, not action items.\n"+
			"--auto moves own projects whose remotes are not in own_remotes to third party.")
	configFlag := fs.String("config", globalConfig, "Config file to change")
	auto := fs.Bool("auto", false, "Sort own projects by own_remotes")
	dryRun := fs.Bool("dry-run", false, "With --auto: only list what would move")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	m, err := openConfig(*configFlag)
	if err != nil {
		return err
	}
	if *auto {
		return markAuto(out, m, *dryRun)
	}
	if len(positional) < 2 {
		fs.Usage()
		return errors.New("name or path and own|3rd needed")
	}
	section, err := sectionFor(positional[len(positional)-1])
	if err != nil {
		return err
	}
	for _, target := range positional[:len(positional)-1] {
		if err := markOne(out, m, target, section); err != nil {
			return err
		}
	}
	return nil
}

// sectionFor maps the kind argument of `mark` to a config section.
func sectionFor(kind string) (string, error) {
	switch strings.ToLower(kind) {
	case "own", "mine":
		return config.SectionOwn, nil
	case "3rd", "third", "third-party", "thirdparty":
		return config.SectionThirdParty, nil
	}
	return "", fmt.Errorf("unknown kind %q — use own or 3rd", kind)
}

// markOne moves one watched project to section.
func markOne(out io.Writer, m *config.Manager, target, section string) error {
	e, err := m.Lookup(target)
	if err != nil {
		return err
	}
	if e.Section == config.SectionIgnored {
		return fmt.Errorf("%s is ignored — run `gitpulse watch %s` first", e.Name, e.Name)
	}
	if e.Section == section {
		fmt.Fprintf(out, "%s is already %s.\n", e.Name, kindLabel(section))
		return nil
	}
	if err := m.Move(e.Name, section); err != nil {
		return err
	}
	fmt.Fprintf(out, "Marked %s as %s.\n", e.Name, kindLabel(section))
	return nil
}

// markAuto moves own projects whose remotes are not in own_remotes to third
// party. Projects already in [third_party] stay there.
func markAuto(out io.Writer, m *config.Manager, dryRun bool) error {
	settings, err := settingsOf(m.Path)
	if err != nil {
		return err
	}
	entries, err := m.Entries()
	if err != nil {
		return err
	}
	classifier := &ownership.Classifier{OwnRemotes: settings.OwnRemotes, PrivateHosts: settings.PrivateHosts}
	if len(settings.OwnRemotes) == 0 {
		var paths []string
		for _, e := range entries {
			paths = append(paths, e.Path)
		}
		suggestOwnRemotes(out, classifier, paths)
		return errors.New("own_remotes is not set in [settings]")
	}

	var moves []config.Entry
	for _, e := range entries {
		if e.Section == config.SectionOwn && sectionOf(classifier, e.Path) == config.SectionThirdParty {
			moves = append(moves, e)
		}
	}
	if len(moves) == 0 {
		fmt.Fprintf(out, "Every own project has a remote in own_remotes, or none.\n")
		return nil
	}
	for _, e := range moves {
		fmt.Fprintf(out, "  own → 3rd  %s\n", e.Name)
		if dryRun {
			continue
		}
		if err := m.Move(e.Name, config.SectionThirdParty); err != nil {
			return err
		}
	}
	if dryRun {
		fmt.Fprintf(out, "\nDry run: nothing changed. `gitpulse mark NAME own` keeps a single one as own.\n")
		return nil
	}
	fmt.Fprintf(out, "\nMoved %d projects to third party. `gitpulse mark NAME own` takes one back.\n", len(moves))
	return nil
}

// ownerCount is how many repositories have a remote with this owner.
type ownerCount struct {
	owner string
	count int
}

// suggestOwnRemotes lists the remote owners of the repositories at paths,
// most frequent first, as candidates for own_remotes.
func suggestOwnRemotes(out io.Writer, c *ownership.Classifier, paths []string) {
	counts := map[string]int{}
	for _, p := range paths {
		urls, _ := ownership.RemoteURLs(p)
		seen := map[string]bool{}
		for _, u := range urls {
			key, local := c.Key(u)
			if owner := ownership.OwnerKey(key); !local && !seen[owner] {
				seen[owner] = true
				counts[owner]++
			}
		}
	}
	if len(counts) == 0 {
		return
	}
	owners := make([]ownerCount, 0, len(counts))
	for o, n := range counts {
		owners = append(owners, ownerCount{o, n})
	}
	sort.Slice(owners, func(i, j int) bool {
		if owners[i].count != owners[j].count {
			return owners[i].count > owners[j].count
		}
		return owners[i].owner < owners[j].owner
	})

	fmt.Fprintf(out, "\nWhich of these remotes are yours? Repositories elsewhere count as third party.\n")
	for _, o := range owners {
		fmt.Fprintf(out, "  %-40s %d\n", o.owner, o.count)
	}
	fmt.Fprintf(out, "Put yours into [settings] of the config, e.g.:\n  own_remotes = %s\n", owners[0].owner)
	fmt.Fprintf(out, "then run `gitpulse mark --auto` to sort the projects you already have.\n")
}
