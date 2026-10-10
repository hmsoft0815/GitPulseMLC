// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
)

// loadConfigOrExplain loads the config gitpulse would use. If there is none,
// or it lists no projects, it tells the user how to get started and exits.
func loadConfigOrExplain(explicit string) *config.Config {
	path := findConfig(explicit)
	if path == "" {
		explainNoConfig(os.Stderr)
		os.Exit(1)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Config file not found: %s\n\n", path)
		fmt.Fprintf(os.Stderr, "Create it with:\n  gitpulse init --config %s <DIR>\n", path)
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read config %s: %v\n", path, err)
		os.Exit(1)
	}
	if len(cfg.Projects) == 0 {
		fmt.Fprintf(os.Stderr, "No repositories configured in %s.\n\n", path)
		fmt.Fprintf(os.Stderr, "Add some with:\n  gitpulse init <DIR>     e.g. gitpulse init ~/code\n")
		os.Exit(1)
	}
	return cfg
}

// explainNoConfig is the first-run message: what gitpulse needs and the
// quickest way to provide it.
func explainNoConfig(w io.Writer) {
	home, _ := os.UserHomeDir()
	fmt.Fprintf(w, `Welcome to GitPulseMLC!

GitPulse shows which of your local Git repositories need attention:
uncommitted changes, unpushed commits, stashes and stale branches.
It needs to know which repositories to watch — there is no config yet.

Quickest start: let gitpulse find your repositories.

  gitpulse init ~/code        search ~/code and save what it finds
  gitpulse                    show the status

Or write the config by hand: %s

  [projects]
  myapp = /path/to/myapp

Config lookup order: --config, $GITPULSE_CONFIG, ./config/repos.ini, ~/.gitpulsemlc/repos.ini
`, tildePath(defaultConfigPath(), home))
}
