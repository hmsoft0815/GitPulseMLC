// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package gitmonitor

import (
	"bytes"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// maxEOLCheckSize skips the line-ending comparison for big files.
const maxEOLCheckSize = 8 << 20

// eolOnlyChanges returns the modified, unstaged files whose content differs
// from the index only in line endings (CRLF ↔ LF). Such a file looks
// changed in every line, but a commit would only swap line endings — usually
// written by a Windows tool into a checkout that Linux git reads.
func eolOnlyChanges(repo *git.Repository, wt *git.Worktree, st git.Status) map[string]bool {
	var candidates []string
	for path, fs := range st {
		if fs.Worktree == git.Modified && fs.Staging == git.Unmodified {
			candidates = append(candidates, path)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	idx, err := repo.Storer.Index()
	if err != nil {
		return nil
	}
	eol := map[string]bool{}
	for _, path := range candidates {
		entry, err := idx.Entry(path)
		if err != nil || entry.Size > maxEOLCheckSize {
			continue
		}
		staged, err := blobBytes(repo, entry.Hash)
		if err != nil {
			continue
		}
		current, err := fileBytes(wt, path)
		if err != nil || bytes.Equal(staged, current) {
			continue
		}
		if bytes.Equal(normalizeEOL(staged), normalizeEOL(current)) {
			eol[path] = true
		}
	}
	return eol
}

// normalizeEOL turns CRLF into LF.
func normalizeEOL(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}

// blobBytes reads a blob from the object store.
func blobBytes(repo *git.Repository, hash plumbing.Hash) ([]byte, error) {
	blob, err := repo.BlobObject(hash)
	if err != nil {
		return nil, err
	}
	r, err := blob.Reader()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, maxEOLCheckSize+1))
}

// fileBytes reads a worktree file.
func fileBytes(wt *git.Worktree, path string) ([]byte, error) {
	f, err := wt.Filesystem.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, maxEOLCheckSize+1))
}
