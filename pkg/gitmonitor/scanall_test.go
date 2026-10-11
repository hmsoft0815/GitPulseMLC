// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// fakeScanner records how many scans run at once.
type fakeScanner struct {
	running, peak atomic.Int32
}

func (f *fakeScanner) Scan(name, path string) (*RepoStatus, error) {
	n := f.running.Add(1)
	defer f.running.Add(-1)
	for {
		p := f.peak.Load()
		if n <= p || f.peak.CompareAndSwap(p, n) {
			break
		}
	}
	time.Sleep(5 * time.Millisecond)
	switch name {
	case "fails":
		return nil, errors.New("boom")
	default:
		return &RepoStatus{Name: name, Path: path, IsClean: true}, nil
	}
}

func targets(n int) []Target {
	ts := make([]Target, n)
	for i := range ts {
		ts[i] = Target{Name: fmt.Sprintf("repo-%02d", i), Path: fmt.Sprintf("/r/%d", i)}
	}
	return ts
}

func TestScanAllKeepsOrderAndReportsProgress(t *testing.T) {
	ts := targets(20)
	fake := &fakeScanner{}
	var calls []int

	results := ScanAll(fake, ts, 4, func(done int, _ Target) { calls = append(calls, done) })

	for i, r := range results {
		if r.Name != ts[i].Name {
			t.Fatalf("results[%d] = %s, want %s", i, r.Name, ts[i].Name)
		}
	}
	if len(calls) != len(ts) || calls[len(calls)-1] != len(ts) {
		t.Errorf("onDone calls = %v, want 1..%d", calls, len(ts))
	}
	if peak := fake.peak.Load(); peak < 2 || peak > 4 {
		t.Errorf("peak concurrency = %d, want 2..4", peak)
	}
}

func TestScanAllTurnsErrorsIntoStatus(t *testing.T) {
	results := ScanAll(&fakeScanner{}, []Target{{Name: "fails", Path: "/x"}, {Name: "ok", Path: "/y"}}, 0, nil)
	if results[0] == nil || results[0].ErrorMsg != "boom" || results[0].Path != "/x" {
		t.Errorf("failed scan = %+v, want ErrorMsg boom and path /x", results[0])
	}
	if results[1].ErrorMsg != "" {
		t.Errorf("ok scan has error %q", results[1].ErrorMsg)
	}
}

func TestScanAllEmpty(t *testing.T) {
	if got := ScanAll(&fakeScanner{}, nil, 0, nil); len(got) != 0 {
		t.Errorf("ScanAll(nil) = %v", got)
	}
}
