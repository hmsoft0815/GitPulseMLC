// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package util

import (
	"fmt"
	"time"
)

// RelativeTime returns a human-readable string representing the time elapsed since t.
// It returns "unknown" for zero times, hours for times < 24h, days for times < 30d,
// and a formatted date (YYYY-MM-DD) for older timestamps.
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	if d.Hours() < 24 {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d.Hours() < 24*30 {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("2006-01-02") // Ab einem Monat festes Datum
}
