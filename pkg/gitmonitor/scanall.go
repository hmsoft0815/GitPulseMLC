// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"runtime"
	"sync"
)

// Target is one repository to scan.
type Target struct {
	Name string
	Path string
}

// ScanAll scans targets concurrently and returns their statuses in the same
// order as targets. workers <= 0 uses one worker per CPU. onDone, if not nil,
// is called after each scan with the number of finished scans; calls never
// overlap, so it may write to a terminal without locking.
func ScanAll(s GitScanner, targets []Target, workers int, onDone func(done int, t Target)) []*RepoStatus {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	workers = min(workers, len(targets))

	results := make([]*RepoStatus, len(targets))
	jobs := make(chan int)
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		done int
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				results[i] = scanOne(s, targets[i])
				if onDone != nil {
					mu.Lock()
					done++
					onDone(done, targets[i])
					mu.Unlock()
				}
			}
		}()
	}
	for i := range targets {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return results
}

// scanOne always returns a status: a scan error ends up in ErrorMsg, so one
// broken repository never leaves a hole in the results.
func scanOne(s GitScanner, t Target) *RepoStatus {
	status, err := s.Scan(t.Name, t.Path)
	if status == nil {
		status = &RepoStatus{Name: t.Name, Path: t.Path}
	}
	if err != nil && status.ErrorMsg == "" {
		status.Error = err
		status.ErrorMsg = err.Error()
	}
	return status
}
