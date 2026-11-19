// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package visibility

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// cacheEntry is one remembered probe result.
type cacheEntry struct {
	Visibility Visibility `json:"visibility"`
	CheckedAt  time.Time  `json:"checked_at"`
}

// cache remembers probe results by probe URL in a JSON file.
type cache struct {
	mu      sync.Mutex
	path    string
	ttl     time.Duration
	entries map[string]cacheEntry
	dirty   bool
}

// DefaultCachePath returns the cache file below the user's cache directory,
// or "" if there is none.
func DefaultCachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gitpulse", "visibility.json")
}

// loadCache reads the cache file; a missing or broken file starts empty.
func loadCache(path string, ttl time.Duration) *cache {
	c := &cache{path: path, ttl: ttl, entries: map[string]cacheEntry{}}
	if path == "" {
		return c
	}
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &c.entries)
	}
	return c
}

// get returns a result younger than the TTL. With ignoreAge every cached
// result counts, which offline mode uses.
func (c *cache) get(key string, ignoreAge bool) (Visibility, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || (!ignoreAge && time.Since(e.CheckedAt) > c.ttl) {
		return "", false
	}
	return e.Visibility, true
}

// put remembers a definite result; Unknown is never cached.
func (c *cache) put(key string, v Visibility) {
	if v != Public && v != Private {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{Visibility: v, CheckedAt: time.Now()}
	c.dirty = true
}

// save writes the cache atomically if anything changed.
func (c *cache) save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dirty || c.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode visibility cache: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write visibility cache: %w", err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return errors.Join(fmt.Errorf("replace visibility cache: %w", err), os.Remove(tmp))
	}
	c.dirty = false
	return nil
}
