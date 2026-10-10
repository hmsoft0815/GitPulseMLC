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
	Name string
	Path string
}

// Known lists the project names and cleaned paths already in the config file,
// enabled or disabled. A missing file yields empty sets and no error.
func (m *Manager) Known() (names, paths map[string]bool, err error) {
	names, paths = make(map[string]bool), make(map[string]bool)
	cfg, err := ini.Load(m.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return names, paths, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("load %s: %w", m.Path, err)
	}
	for _, section := range []string{"projects", "disabled_projects"} {
		for _, key := range cfg.Section(section).Keys() {
			names[key.Name()] = true
			paths[filepath.Clean(key.String())] = true
		}
	}
	return names, paths, nil
}

// AddProjects appends projects to the [projects] section. If the config file
// does not exist yet, it is created (with its directory) and seeded with the
// default settings; replacePrefix, if set, is shown as "~" in the output.
func (m *Manager) AddProjects(projects []Project, replacePrefix string) error {
	cfg, err := ini.Load(m.Path)
	if errors.Is(err, fs.ErrNotExist) {
		cfg = newConfigFile(replacePrefix)
	} else if err != nil {
		return fmt.Errorf("load %s: %w", m.Path, err)
	}

	section := cfg.Section("projects")
	for _, p := range projects {
		section.Key(p.Name).SetValue(filepath.Clean(p.Path))
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
	cfg := ini.Empty()

	projects := cfg.Section("projects")
	projects.Comment = "# Repositories to watch: name = absolute path.\n" +
		"# Add more with `gitpulse init <dir>` or by hand."

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

	disabled := cfg.Section("disabled_projects")
	disabled.Comment = "# Move a project here to stop watching it without forgetting it."
	return cfg
}
