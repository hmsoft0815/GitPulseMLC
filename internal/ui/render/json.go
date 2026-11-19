// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hmsoft0815/GitPulseMLC/pkg/gitmonitor"
)

// Output is the document `gitpulse --json` writes (see docs/schema.json).
type Output struct {
	Version string                   `json:"version"`
	Results []*gitmonitor.RepoStatus `json:"results"`
	Summary gitmonitor.Summary       `json:"summary"`
}

// JSON writes the scan results as indented JSON.
func JSON(w io.Writer, version string, results []*gitmonitor.RepoStatus, summary gitmonitor.Summary) error {
	if results == nil {
		results = []*gitmonitor.RepoStatus{}
	}
	b, err := json.MarshalIndent(Output{Version: version, Results: results, Summary: summary}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
