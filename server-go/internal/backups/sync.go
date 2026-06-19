package backups

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// syncResult summarises one incremental sync run.
type syncResult struct {
	Copied  int
	Skipped int
	Bytes   int64
}

// syncTree performs an incremental one-way copy of source into target: files
// missing in target or differing by size/mtime are copied; identical files are
// skipped. Directory structure is mirrored. This is the real work behind a
// backup "run" — it replaces the previous state-string simulation.
func syncTree(source, target string) (syncResult, error) {
	var res syncResult
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
		if existing, err := os.Stat(dest); err == nil {
			if existing.Size() == fi.Size() && existing.ModTime().Equal(fi.ModTime()) {
				res.Skipped++
				return nil
			}
		}
		if err := copyFile(path, dest, fi); err != nil {
			return err
		}
		res.Copied++
		res.Bytes += fi.Size()
		return nil
	})
	return res, err
}

// verifyResult summarises one verify run.
type verifyResult struct {
	Checked  int
	Mismatch int
	Missing  int
}

// verifyTree checks that every regular file under source exists in target with
// identical size and content hash. It is the real work behind a backup
// "verify", replacing the previous state-string simulation.
func verifyTree(source, target string) (verifyResult, error) {
	var res verifyResult
	info, err := os.Stat(source)
	if err != nil {
		return res, fmt.Errorf("source unavailable: %w", err)
	}
	if !info.IsDir() {
		return res, fmt.Errorf("source is not a directory: %s", source)
	}
	err = filepath.Walk(source, func(path string, fi os.FileInfo, walkErr error) error {
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
		same, err := sameContent(path, dest)
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

// sameContent compares two files by SHA-256, streaming to avoid loading whole
// files into memory.
func sameContent(a, b string) (bool, error) {
	ha, err := hashFile(a)
	if err != nil {
		return false, err
	}
	hb, err := hashFile(b)
	if err != nil {
		return false, err
	}
	return ha == hb, nil
}

func hashFile(path string) (string, error) {
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

// copyFile copies src to dest atomically (temp + rename) and preserves the
// modification time so the next incremental run can skip unchanged files.
func copyFile(src, dest string, fi os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// Use a unique temp file so concurrent syncs of the same job (e.g. a
	// double-clicked Run) can't write the same scratch path and corrupt it.
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
	// Preserve the source file's permissions and mtime so the backup is faithful
	// and the incremental skip-check stays accurate.
	_ = os.Chmod(tmp, fi.Mode().Perm())
	_ = os.Chtimes(tmp, fi.ModTime(), fi.ModTime())
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
