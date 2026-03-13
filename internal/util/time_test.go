// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package util

import (
	"testing"
	"time"
)

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{"Zero Time", time.Time{}, "unknown"},
		{"Just now", now.Add(-1 * time.Hour), "1h ago"},
		{"Few hours ago", now.Add(-5 * time.Hour), "5h ago"},
		{"Yesterday", now.Add(-25 * time.Hour), "1d ago"},
		{"A week ago", now.Add(-7 * 24 * time.Hour), "7d ago"},
		{"A month ago", now.Add(-31 * 24 * time.Hour), now.Add(-31 * 24 * time.Hour).Format("2006-01-02")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RelativeTime(tt.input)
			if got != tt.expected {
				t.Errorf("RelativeTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}
