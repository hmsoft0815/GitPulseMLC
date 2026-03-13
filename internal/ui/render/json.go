// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package render

import (
	"encoding/json"
	"fmt"

	"gitpulsemcl/pkg/gitmonitor"
)

type Output struct {
	Version string                   `json:"version"`
	Results []*gitmonitor.RepoStatus `json:"results"`
	Summary gitmonitor.Summary       `json:"summary"`
}

func JSON(version string, results []*gitmonitor.RepoStatus, summary gitmonitor.Summary) {
	out := Output{
		Version: version,
		Results: results,
		Summary: summary,
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}
