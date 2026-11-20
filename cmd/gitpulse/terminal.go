// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/render"
	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

// terminalWidth returns the width of the terminal stdout writes to, or 0
// when stdout is not a terminal (pipe, file, cron), so that output for
// scripts never changes with the window size. $COLUMNS overrides the
// detected width of a terminal.
func terminalWidth() int {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return 0
	}
	if c, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && c > 0 {
		return c
	}
	w, _, err := term.GetSize(fd)
	if err != nil {
		return 0
	}
	return w
}

// newScanner applies the stat_check setting (full or minimal) and the
// backend: --backend, else the backend setting, else auto.
func newScanner(settings config.Settings, backendFlag string) (gitmonitor.GitScanner, error) {
	var opts gitmonitor.Options
	switch strings.ToLower(strings.TrimSpace(settings.StatCheck)) {
	case "", "full":
	case "minimal":
		opts.MinimalStat = true
	default:
		return nil, fmt.Errorf("unknown stat_check %q — use full or minimal", settings.StatCheck)
	}
	name := settings.Backend
	if backendFlag != "" {
		name = backendFlag
	}
	backend, err := gitmonitor.ParseBackend(name)
	if err != nil {
		return nil, err
	}
	if backend == gitmonitor.BackendGit && !gitmonitor.GitInstalled() {
		return nil, fmt.Errorf("backend git: no git executable found on the PATH — install git or use backend auto or go-git")
	}
	opts.Backend = backend
	return gitmonitor.NewScannerWithOptions(opts), nil
}

// chooseLayout takes --layout, else the layout setting, else auto.
func chooseLayout(flagValue, setting string) (render.Layout, error) {
	if flagValue != "" {
		return render.ParseLayout(flagValue)
	}
	return render.ParseLayout(setting)
}
