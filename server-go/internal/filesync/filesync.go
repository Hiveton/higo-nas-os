// Package filesync is the shared, dependency-free file synchronization core used
// by the backups and sync domains: an incremental one-way copy (atomic
// temp+rename, mtime-preserving) and a SHA-256 content verify. Extracted so both
// domains share one correct implementation instead of copy-pasting it.
package filesync

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SyncResult summarises one incremental sync run.
type SyncResult struct {
	Copied  int
	Skipped int
	Bytes   int64
}

// VerifyResult summarises one verify run.
type VerifyResult struct {
	Checked  int
	Mismatch int
	Missing  int
}

// Filter decides whether a relative path (slash-separated, relative to the sync
// source) participates in the sync. A nil filter includes everything. It lets
// the sync domain implement selective / on-demand sync without forking the walk.
type Filter func(rel string) bool

// SyncTree performs an incremental one-way copy of source into target: files
// missing in target or differing by size/mtime are copied; identical files are
// skipped. Directory structure is mirrored. When filter is non-nil, only files
// (and the directories needed to hold them) for which it returns true are
// synced. Honors ctx cancellation so an in-flight run can be aborted mid-walk.
func SyncTree(ctx context.Context, source, target string, filter Filter) (SyncResult, error) {
	var res SyncResult
	info, err := os.Stat(source)
	if err != nil {
		return res, fmt.Errorf("source unavailable: %w", err)
	}
	if !info.IsDir() {
		return res, fmt.Errorf("source is not a directory: %s", source)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return res, err
	}
	err = filepath.Walk(source, func(path string, fi os.FileInfo, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dest := filepath.Join(target, rel)
		if fi.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		if !fi.Mode().IsRegular() {
			return nil // skip symlinks/devices
		}
		if filter != nil && !filter(filepath.ToSlash(rel)) {
			return nil
		}
		if existing, err := os.Stat(dest); err == nil {
			if existing.Size() == fi.Size() && existing.ModTime().Equal(fi.ModTime()) {
				res.Skipped++
				return nil
			}
		}
		if err := CopyFile(path, dest, fi); err != nil {
			return err
		}
		res.Copied++
		res.Bytes += fi.Size()
		return nil
	})
	return res, err
}

// VerifyTree checks that every regular file under source exists in target with
// identical size and content hash.
func VerifyTree(ctx context.Context, source, target string) (VerifyResult, error) {
	var res VerifyResult
	info, err := os.Stat(source)
	if err != nil {
		return res, fmt.Errorf("source unavailable: %w", err)
	}
	if !info.IsDir() {
		return res, fmt.Errorf("source is not a directory: %s", source)
	}
	err = filepath.Walk(source, func(path string, fi os.FileInfo, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if fi.IsDir() || !fi.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		res.Checked++
		destInfo, err := os.Stat(dest)
		if err != nil {
			res.Missing++
			return nil
		}
		if destInfo.Size() != fi.Size() {
			res.Mismatch++
			return nil
		}
		same, err := SameContent(path, dest)
		if err != nil {
			return err
		}
		if !same {
			res.Mismatch++
		}
		return nil
	})
	return res, err
}

// SameContent compares two files by streaming SHA-256.
func SameContent(a, b string) (bool, error) {
	ha, err := HashFile(a)
	if err != nil {
		return false, err
	}
	hb, err := HashFile(b)
	if err != nil {
		return false, err
	}
	return ha == hb, nil
}

// HashFile returns the hex SHA-256 of a file's contents.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// CopyFile copies src to dest atomically (temp + rename) and preserves the
// source's permissions + modification time so incremental runs stay accurate.
func CopyFile(src, dest string, fi os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := out.Name()
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Chmod(tmp, fi.Mode().Perm())
	_ = os.Chtimes(tmp, fi.ModTime(), fi.ModTime())
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
