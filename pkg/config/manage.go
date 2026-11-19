// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package config

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

// iniOptions keep a project name that occurs twice instead of silently
// dropping all but the last path when the file is written back.
var iniOptions = ini.LoadOptions{AllowShadows: true}

// loadINI reads the config file with iniOptions.
func loadINI(path string) (*ini.File, error) { return ini.LoadSources(iniOptions, path) }

// projectSections are the sections that list projects, in display order.
var projectSections = []string{SectionOwn, SectionThirdParty, SectionIgnored}

// Entry is one project as written in the config file.
type Entry struct {
	Name    string
	Path    string
	Section string // SectionOwn, SectionThirdParty or SectionIgnored.
}

// ErrNotFound is returned when no project matches a name or path.
var ErrNotFound = errors.New("project not found")

// Entries lists every project of the config file, own first, then third
// party, then ignored. A missing file has no entries.
func (m *Manager) Entries() ([]Entry, error) {
	cfg, err := loadINI(m.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", m.Path, err)
	}
	var entries []Entry
	for _, section := range projectSections {
		for _, key := range cfg.Section(section).Keys() {
			for _, value := range key.ValueWithShadows() {
				entries = append(entries, Entry{Name: key.Name(), Path: filepath.Clean(value), Section: section})
			}
		}
	}
	return entries, nil
}

// Lookup finds a project by its name or, failing that, by its path. A
// relative path such as "." is taken relative to the working directory.
func (m *Manager) Lookup(nameOrPath string) (Entry, error) {
	entries, err := m.Entries()
	if err != nil {
		return Entry{}, err
	}
	for _, e := range entries {
		if e.Name == nameOrPath {
			return e, nil
		}
	}
	target := absPath(nameOrPath)
	for _, e := range entries {
		if absPath(e.Path) == target {
			return e, nil
		}
	}
	return Entry{}, fmt.Errorf("%q: %w in %s", nameOrPath, ErrNotFound, m.Path)
}

// keepComment leaves the comment above a moved key where it was, on the
// next key of the old section; without one it travels with the key.
func keepComment(from *ini.Section, key, moved *ini.Key) {
	if key.Comment == "" {
		return
	}
	keys := from.Keys()
	after := false
	for _, k := range keys {
		if k.Name() == key.Name() {
			after = true
			continue
		}
		// A name that occurs twice loses its comment on the next load
		// (ini keeps the comment of the last occurrence), so skip it.
		if after && len(k.ValueWithShadows()) == 1 {
			k.Comment = strings.TrimSpace(key.Comment + "\n" + k.Comment)
			return
		}
	}
	moved.Comment = key.Comment
}

// absPath returns the absolute, symlink-free form of p, or p cleaned if it
// cannot be resolved.
func absPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// Move puts the project named name into section, keeping its path.
func (m *Manager) Move(name, section string) error {
	cfg, err := loadINI(m.Path)
	if err != nil {
		return fmt.Errorf("load %s: %w", m.Path, err)
	}
	for _, from := range projectSections {
		key, err := cfg.Section(from).GetKey(name)
		if err != nil {
			continue
		}
		if from == section {
			return nil
		}
		if len(key.ValueWithShadows()) > 1 {
			return fmt.Errorf("%q names more than one project in %s — rename one first", name, m.Path)
		}
		moved := cfg.Section(section).Key(name)
		moved.SetValue(key.String())
		keepComment(cfg.Section(from), key, moved)
		cfg.Section(from).DeleteKey(name)
		if err := cfg.SaveTo(m.Path); err != nil {
			return fmt.Errorf("save %s: %w", m.Path, err)
		}
		return nil
	}
	return fmt.Errorf("%q: %w in %s", name, ErrNotFound, m.Path)
}

// Remove deletes one entry — name, path and section as returned by Entries
// or Lookup. Of a name that occurs twice only the entry with that path goes.
func (m *Manager) Remove(e Entry) error {
	cfg, err := loadINI(m.Path)
	if err != nil {
		return fmt.Errorf("load %s: %w", m.Path, err)
	}
	section := cfg.Section(e.Section)
	key, err := section.GetKey(e.Name)
	if err != nil {
		return fmt.Errorf("%q: %w in [%s] of %s", e.Name, ErrNotFound, e.Section, m.Path)
	}
	var remaining []string
	for _, v := range key.ValueWithShadows() {
		if filepath.Clean(v) != filepath.Clean(e.Path) {
			remaining = append(remaining, v)
		}
	}
	if len(remaining) == len(key.ValueWithShadows()) {
		return fmt.Errorf("%q with path %s: %w in %s", e.Name, e.Path, ErrNotFound, m.Path)
	}
	comment := key.Comment
	if len(remaining) == 0 {
		keepComment(section, key, key)
	}
	section.DeleteKey(e.Name)
	if len(remaining) > 0 {
		kept := section.Key(e.Name)
		kept.SetValue(remaining[0])
		kept.Comment = comment
		for _, v := range remaining[1:] {
			if err := kept.AddShadow(v); err != nil {
				return fmt.Errorf("keep %q: %w", e.Name, err)
			}
		}
	}
	if err := cfg.SaveTo(m.Path); err != nil {
		return fmt.Errorf("save %s: %w", m.Path, err)
	}
	return nil
}
