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
}

// Config holds the mapping of project names to their absolute paths and global settings.
type Config struct {
	Projects map[string]string // Mapping of project name to its file system path.
	Settings Settings          // Global application settings.
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
	cfg, err := ini.Load(m.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load ini file: %w", err)
	}

	// 1. Parse Projects
	projects := make(map[string]string)
	projSection := cfg.Section("projects")
	if projSection != nil {
		for _, key := range projSection.Keys() {
			repoPath := key.String()
			cleanedPath := filepath.Clean(repoPath)

			// Check if the path ends with .git (common mistake)
			if strings.HasSuffix(cleanedPath, ".git") {
				fmt.Fprintf(os.Stderr, "Warning: Project '%s' path ends in '.git'. Using parent directory: %s\n", key.Name(), filepath.Dir(cleanedPath))
				cleanedPath = filepath.Dir(cleanedPath)
			}

			projects[key.Name()] = cleanedPath
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
	}

	// 3. Parse General Settings
	genSection := cfg.Section("general")
	if genSection != nil {
		settings.ReplacePathPrefix = genSection.Key("replace_path_prefix").String()
		settings.ReplaceWith = genSection.Key("replace_with").String()
	}

	return &Config{
		Projects: projects,
		Settings: settings,
	}, nil
}

// AddProject adds or updates a project in the configuration.
func (m *Manager) AddProject(name, path string) error {
	cfg, err := ini.Load(m.Path)
	if err != nil {
		// If file doesn't exist, create a new one
		cfg = ini.Empty()
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
	cfg, err := ini.Load(m.Path)
	if err != nil {
		return err
	}

	cfg.Section("projects").DeleteKey(name)
	return cfg.SaveTo(m.Path)
}

// ToggleProject moves a project between [projects] and [disabled_projects].
func (m *Manager) ToggleProject(name string, enable bool) error {
	cfg, err := ini.Load(m.Path)
	if err != nil {
		return err
	}

	sourceSection := "disabled_projects"
	targetSection := "projects"
	if !enable {
		sourceSection = "projects"
		targetSection = "disabled_projects"
	}

	key := cfg.Section(sourceSection).Key(name)
	if key == nil {
		return fmt.Errorf("project '%s' not found in section [%s]", name, sourceSection)
	}

	cfg.Section(targetSection).Key(name).SetValue(key.String())
	cfg.Section(sourceSection).DeleteKey(name)

	return cfg.SaveTo(m.Path)
}

// LoadConfig is a convenience wrapper for NewManager(path).Load().
func LoadConfig(path string) (*Config, error) {
	return NewManager(path).Load()
}
