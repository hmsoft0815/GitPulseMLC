// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"strings"
	"testing"

	"github.com/hmsoft0815/GitPulseMLC/pkg/config"
)

func TestNewScannerSettings(t *testing.T) {
	tests := []struct {
		statCheck, backend, flag string
		wantErr                  string
	}{
		{"", "", "", ""},
		{"minimal", "go-git", "", ""},
		{"full", "GIT", "", ""},
		{"", "go-git", "auto", ""},
		{"", "auto", "libgit", "unknown backend"},
		{"", "svn", "", "unknown backend"},
		{"fast", "", "", "unknown stat_check"},
	}
	for _, tt := range tests {
		_, err := newScanner(config.Settings{StatCheck: tt.statCheck, Backend: tt.backend}, tt.flag)
		if tt.wantErr == "" && err != nil {
			t.Errorf("%+v: %v", tt, err)
		}
		if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
			t.Errorf("%+v: error %v, want %q", tt, err, tt.wantErr)
		}
	}
}
