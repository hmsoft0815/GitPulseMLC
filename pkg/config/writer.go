// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

// Project is a single entry of the [projects] section.
type Project struct {
	Name    string
	Path    string
	Section string // Target section; empty means SectionOwn.
}

// Known lists the project names and cleaned paths already in the config file,
// enabled or disabled. A missing file yields empty sets and no error.
func (m *Manager) Known() (names, paths map[string]bool, err error) {
	names, paths = make(map[string]bool), make(map[string]bool)
	cfg, err := loadINI(m.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return names, paths, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("load %s: %w", m.Path, err)
	}
	for _, section := range projectSections {
		for _, key := range cfg.Section(section).Keys() {
			names[key.Name()] = true
			for _, value := range key.ValueWithShadows() {
				paths[filepath.Clean(value)] = true
			}
		}
	}
	return names, paths, nil
}

// AddProjects appends projects to the [projects] section. If the config file
// does not exist yet, it is created (with its directory) and seeded with the
// default settings; replacePrefix, if set, is shown as "~" in the output.
func (m *Manager) AddProjects(projects []Project, replacePrefix string) error {
	cfg, err := loadINI(m.Path)
	if errors.Is(err, fs.ErrNotExist) {
		cfg = newConfigFile(replacePrefix)
	} else if err != nil {
		return fmt.Errorf("load %s: %w", m.Path, err)
	}

	for _, p := range projects {
		section := p.Section
		if section == "" {
			section = SectionOwn
		}
		cfg.Section(section).Key(p.Name).SetValue(filepath.Clean(p.Path))
	}

	if err := os.MkdirAll(filepath.Dir(m.Path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := cfg.SaveTo(m.Path); err != nil {
		return fmt.Errorf("save %s: %w", m.Path, err)
	}
	return nil
}

// newConfigFile returns a config skeleton with commented default settings.
func newConfigFile(replacePrefix string) *ini.File {
	cfg := ini.Empty(iniOptions)

	projects := cfg.Section("projects")
	projects.Comment = "# Your own repositories to watch: name = absolute path.\n" +
		"# Manage them with `gitpulse add`, `ignore`, `watch`, `mark` and `list`."

	general := cfg.Section("general")
	general.Comment = "# Shorten paths in the output: replace_path_prefix is shown as replace_with."
	if replacePrefix != "" {
		general.Key("replace_path_prefix").SetValue(replacePrefix)
		general.Key("replace_with").SetValue("~")
	}

	settings := cfg.Section("settings")
	settings.Key("column_name_width").SetValue("25")
	settings.Key("show_summary").SetValue("true")
	settings.Key("compact_mode").SetValue("false")
	settings.Key("compact_mode").Comment = "# true = only list repositories that need action"
	settings.Key("own_remotes").SetValue("")
	settings.Key("own_remotes").Comment = "# Your remotes, e.g. github.com/me, git.example.com — others are added as third party"

	thirdParty := cfg.Section(SectionThirdParty)
	thirdParty.Comment = "# Repositories of others, checked out to look at or test.\n" +
		"# Local changes here are hints, not action items."

	disabled := cfg.Section("disabled_projects")
	disabled.Comment = "# Ignored: not scanned, and `gitpulse add` will not add them again."
	return cfg
}
