package foldersync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"higoos/server-go/internal/filesync"
)

// conflictRec is a detected two-way collision (relative path + reason) enriched
// with both sides' size/mtime so the UI can show an informed comparison.
type conflictRec struct {
	rel         string
	detail      string
	sourceSize  int64
	targetSize  int64
	sourceMtime time.Time
	targetMtime time.Time
}

// twoWaySync performs an additive two-way merge of dirs a and b: every file
// present on either side ends up on both, taking the newer copy. A same-path,
// different-content pair with equal mtimes is a conflict resolved per policy
// (recorded and skipped under ConflictManual). It does not propagate deletions
// (no base-state tracking) — an intentional, documented v1 limitation.
func twoWaySync(ctx context.Context, a, b string, policy ConflictPolicy, filter filesync.Filter) (filesync.SyncResult, []conflictRec, error) {
	var res filesync.SyncResult
	for _, dir := range []string{a, b} {
		if info, err := os.Stat(dir); err != nil {
			return res, nil, fmt.Errorf("sync path unavailable: %w", err)
		} else if !info.IsDir() {
			return res, nil, fmt.Errorf("sync path is not a directory: %s", dir)
		}
	}

	files := map[string]struct{}{} // union of relative file paths
	collect := func(root string) error {
		return filepath.Walk(root, func(path string, fi os.FileInfo, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if walkErr != nil {
				return walkErr
			}
			if fi.IsDir() || !fi.Mode().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if filter != nil && !filter(rel) {
				return nil
			}
			files[rel] = struct{}{}
			return nil
		})
	}
	if err := collect(a); err != nil {
		return res, nil, err
	}
	if err := collect(b); err != nil {
		return res, nil, err
	}

	var conflicts []conflictRec
	for rel := range files {
		if err := ctx.Err(); err != nil {
			return res, conflicts, err
		}
		ap := filepath.Join(a, filepath.FromSlash(rel))
		bp := filepath.Join(b, filepath.FromSlash(rel))
		ai, aerr := os.Stat(ap)
		bi, berr := os.Stat(bp)
		switch {
		case aerr == nil && berr != nil: // only on a → copy to b
			if err := filesync.CopyFile(ap, bp, ai); err != nil {
				return res, conflicts, err
			}
			res.Copied++
			res.Bytes += ai.Size()
		case berr == nil && aerr != nil: // only on b → copy to a
			if err := filesync.CopyFile(bp, ap, bi); err != nil {
				return res, conflicts, err
			}
			res.Copied++
			res.Bytes += bi.Size()
		case aerr == nil && berr == nil: // both sides have it
			if ai.Size() == bi.Size() {
				if same, err := filesync.SameContent(ap, bp); err != nil {
					return res, conflicts, err
				} else if same {
					res.Skipped++
					continue
				}
			}
			from, to, fromInfo, isConflict := resolveTwoWay(ap, bp, ai, bi, policy)
			if isConflict {
				conflicts = append(conflicts, conflictRec{
					rel:         rel,
					detail:      "两端内容不同且修改时间一致，需手动处理",
					sourceSize:  ai.Size(),
					targetSize:  bi.Size(),
					sourceMtime: ai.ModTime().UTC(),
					targetMtime: bi.ModTime().UTC(),
				})
				continue
			}
			if err := filesync.CopyFile(from, to, fromInfo); err != nil {
				return res, conflicts, err
			}
			res.Copied++
			res.Bytes += fromInfo.Size()
		}
	}
	return res, conflicts, nil
}

// resolveTwoWay picks the winning side for a both-exist divergence. Returns the
// (from, to) paths to copy, the winner's FileInfo, and whether it's an
// unresolvable conflict (manual policy, or equal mtimes under newer policy).
func resolveTwoWay(ap, bp string, ai, bi os.FileInfo, policy ConflictPolicy) (from, to string, info os.FileInfo, conflict bool) {
	switch policy {
	case ConflictSource:
		return ap, bp, ai, false
	case ConflictTarget:
		return bp, ap, bi, false
	case ConflictManual:
		return "", "", nil, true
	default: // ConflictNewer
		if ai.ModTime().After(bi.ModTime()) {
			return ap, bp, ai, false
		}
		if bi.ModTime().After(ai.ModTime()) {
			return bp, ap, bi, false
		}
		return "", "", nil, true // equal mtime, different content → conflict
	}
}
