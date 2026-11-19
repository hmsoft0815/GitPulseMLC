// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

// Settings holds the global application configuration options.
type Settings struct {
	ColumnNameWidth   int    // The width of the project name column in the TUI.
	ShowSummary       bool   // Whether to display the summary block at the end.
	CompactMode       bool   // If true, only shows repositories that need attention.
	ReplacePathPrefix string // Prefix to replace in display paths.
	ReplaceWith       string // Replacement string for the prefix.
	// PrivateHosts are glob patterns of Git hosts that count as local network
	// and are never probed for public access (e.g. "git.example.com, *.corp").
	PrivateHosts []string
	// OwnRemotes are remotes that belong to you, as "host/owner" or "host"
	// patterns (e.g. "github.com/me", "git.example.com"). New repositories
	// with a remote elsewhere are added as third party.
	OwnRemotes []string
}

// Sections of the config file that list projects.
const (
	SectionOwn        = "projects"          // Your own projects.
	SectionThirdParty = "third_party"       // Checked out to look at or test.
	SectionIgnored    = "disabled_projects" // Remembered, but not scanned.
)

// Config holds the mapping of project names to their absolute paths and global settings.
type Config struct {
	Projects   map[string]string // Watched projects, own and third party: name → path.
	ThirdParty map[string]bool   // Names of watched projects that are third party.
	Settings   Settings          // Global application settings.
}

// Manager handles reading and writing the configuration file.
type Manager struct {
	Path string
}

// NewManager creates a new ConfigManager.
func NewManager(path string) *Manager {
	return &Manager{Path: path}
}

// Load reads the configuration from the file system.
func (m *Manager) Load() (*Config, error) {
	cfg, err := loadINI(m.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load ini file: %w", err)
	}

	// 1. Parse Projects: own first, third party after
	projects := make(map[string]string)
	thirdParty := make(map[string]bool)
	for _, section := range []string{SectionOwn, SectionThirdParty} {
		for _, key := range cfg.Section(section).Keys() {
			if values := key.ValueWithShadows(); len(values) > 1 {
				fmt.Fprintf(os.Stderr, "Warning: project name '%s' is used %d times; only %s is watched — rename the others\n",
					key.Name(), len(values), values[len(values)-1])
			}
			if _, dup := projects[key.Name()]; dup {
				fmt.Fprintf(os.Stderr, "Warning: project '%s' is listed twice; using [%s]\n", key.Name(), section)
			}
			values := key.ValueWithShadows()
			projects[key.Name()] = projectPath(key.Name(), values[len(values)-1])
			thirdParty[key.Name()] = section == SectionThirdParty
		}
	}

	// 2. Parse Settings (with defaults)
	settings := Settings{
		ColumnNameWidth: 25,
		ShowSummary:     true,
		CompactMode:     false,
	}

	setSection := cfg.Section("settings")
	if setSection != nil {
		if val, err := setSection.Key("column_name_width").Int(); err == nil {
			settings.ColumnNameWidth = val
		}
		if val, err := setSection.Key("show_summary").Bool(); err == nil {
			settings.ShowSummary = val
		}
		if val, err := setSection.Key("compact_mode").Bool(); err == nil {
			settings.CompactMode = val
		}
		settings.PrivateHosts = splitList(setSection.Key("private_hosts").String())
		settings.OwnRemotes = splitList(setSection.Key("own_remotes").String())
	}

	// 3. Parse General Settings
	genSection := cfg.Section("general")
	if genSection != nil {
		settings.ReplacePathPrefix = genSection.Key("replace_path_prefix").String()
		settings.ReplaceWith = genSection.Key("replace_with").String()
	}

	return &Config{
		Projects:   projects,
		ThirdParty: thirdParty,
		Settings:   settings,
	}, nil
}

// projectPath cleans a configured path; a path ending in .git (a common
// mistake) points to its parent directory.
func projectPath(name, repoPath string) string {
	cleaned := filepath.Clean(repoPath)
	if strings.HasSuffix(cleaned, ".git") {
		fmt.Fprintf(os.Stderr, "Warning: Project '%s' path ends in '.git'. Using parent directory: %s\n", name, filepath.Dir(cleaned))
		cleaned = filepath.Dir(cleaned)
	}
	return cleaned
}

// splitList splits a comma-separated value and drops empty items.
func splitList(value string) []string {
	var items []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// AddProject adds or updates a project in the configuration.
func (m *Manager) AddProject(name, path string) error {
	cfg, err := loadINI(m.Path)
	if err != nil {
		// If file doesn't exist, create a new one
		cfg = ini.Empty(iniOptions)
	}

	cleanedPath := filepath.Clean(path)
	if strings.HasSuffix(cleanedPath, ".git") {
		cleanedPath = filepath.Dir(cleanedPath)
	}

	cfg.Section("projects").Key(name).SetValue(cleanedPath)
	return cfg.SaveTo(m.Path)
}

// RemoveProject removes a project from the configuration.
func (m *Manager) RemoveProject(name string) error {
	cfg, err := loadINI(m.Path)
	if err != nil {
		return err
	}

	cfg.Section("projects").DeleteKey(name)
	return cfg.SaveTo(m.Path)
}

// ToggleProject moves a project between [projects] and [disabled_projects].
//
// Deprecated: use Move, which also knows [third_party].
func (m *Manager) ToggleProject(name string, enable bool) error {
	if enable {
		return m.Move(name, SectionOwn)
	}
	return m.Move(name, SectionIgnored)
}

// LoadConfig is a convenience wrapper for NewManager(path).Load().
func LoadConfig(path string) (*Config, error) {
	return NewManager(path).Load()
}
