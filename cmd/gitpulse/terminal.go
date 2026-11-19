// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"os"
	"strconv"

	"golang.org/x/term"

	"github.com/hmsoft0815/GitPulseMLC/internal/ui/render"
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

// chooseLayout takes --layout, else the layout setting, else auto.
func chooseLayout(flagValue, setting string) (render.Layout, error) {
	if flagValue != "" {
		return render.ParseLayout(flagValue)
	}
	return render.ParseLayout(setting)
}
