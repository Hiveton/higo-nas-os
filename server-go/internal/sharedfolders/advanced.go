package sharedfolders

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// runSharedCommand is the default host command runner. On dev hosts (no real NAS
// root) the advanced ops are never reached, so this only runs on a Linux NAS.
func runSharedCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// btrfsSubvolCreate creates a Btrfs subvolume at abs (parent dirs first). Returns
// an error if btrfs is unavailable or the create fails, so Create can fall back
// to a plain directory.
func (s *Service) btrfsSubvolCreate(ctx context.Context, abs string) error {
	if s.runner == nil {
		return fmt.Errorf("no command runner")
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o2770); err != nil {
		return err
	}
	if _, err := os.Stat(abs); err == nil {
		// Path already exists — treat as already-provisioned, not a subvolume.
		return fmt.Errorf("path exists")
	}
	if out, err := s.runner(ctx, "btrfs", "subvolume", "create", abs); err != nil {
		return fmt.Errorf("btrfs subvolume create: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	_ = os.Chmod(abs, 0o2770)
	return nil
}

// snapshotDir is where a folder's read-only snapshots live: a hidden sibling
// directory keyed by folder id, so it never appears inside the share itself.
func snapshotDir(abs, folderID string) string {
	return filepath.Join(filepath.Dir(abs), ".higoos-snapshots", folderID)
}

// SetAdvanced updates a folder's advanced settings (recycle bin / quota /
// encryption intent). Recycle re-derives the Samba export; quota is applied as a
// Btrfs qgroup limit best-effort; encryption is recorded as intent. Host ops that
// require Btrfs degrade gracefully (intent persisted, no host change) on ext4.
func (s *Service) SetAdvanced(ctx context.Context, folderID string, req SetAdvancedRequest) (SharedFolderView, error) {
	s.mu.Lock()
	idx := s.indexOf(folderID)
	if idx < 0 {
		s.mu.Unlock()
		return SharedFolderView{}, fmt.Errorf("shared folder not found: %s", folderID)
	}
	if req.Recycle != nil {
		s.folders[idx].Recycle = *req.Recycle
	}
	if req.QuotaBytes != nil {
		s.folders[idx].QuotaBytes = *req.QuotaBytes
	}
	if req.Encrypted != nil {
		s.folders[idx].Encrypted = *req.Encrypted
	}
	folder := s.folders[idx]
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		return SharedFolderView{}, err
	}
	s.mu.Unlock()

	// Recycle flag changes the SMB share definition → re-derive exports.
	if req.Recycle != nil {
		if err := s.deriveAll(ctx, folder); err != nil {
			return SharedFolderView{}, err
		}
	}
	// Quota: best-effort Btrfs qgroup limit on a real subvolume. ext4/non-subvol
	// folders keep the intent without a host change.
	if req.QuotaBytes != nil && folder.Subvolume && s.runner != nil && strings.TrimSpace(s.nasRoot) != "" {
		abs := s.absPath(ctx, folder)
		if folder.QuotaBytes <= 0 {
			_, _ = s.runner(ctx, "btrfs", "qgroup", "limit", "none", abs)
		} else {
			_, _ = s.runner(ctx, "btrfs", "qgroup", "limit", fmt.Sprintf("%d", folder.QuotaBytes), abs)
		}
	}
	return s.refresh(ctx, folder)
}

// CreateSnapshot takes a read-only Btrfs snapshot of the folder. Requires the
// folder to be a Btrfs subvolume; otherwise returns a clear error so the UI can
// explain the filesystem doesn't support it.
func (s *Service) CreateSnapshot(ctx context.Context, folderID string, req SnapshotRequest) (Snapshot, error) {
	folder, ok := s.folderByID(folderID)
	if !ok {
		return Snapshot{}, fmt.Errorf("shared folder not found: %s", folderID)
	}
	if !folder.Subvolume {
		return Snapshot{}, fmt.Errorf("该共享文件夹所在文件系统不支持快照(需要 Btrfs 子卷)")
	}
	if s.runner == nil || strings.TrimSpace(s.nasRoot) == "" {
		return Snapshot{}, fmt.Errorf("快照在当前主机不可用")
	}
	abs := s.absPath(ctx, folder)
	name := sanitizeSnapName(req.Name)
	if name == "" {
		name = "snap-" + s.now().Format("20060102-150405")
	}
	dir := snapshotDir(abs, folder.ID)
	if err := os.MkdirAll(dir, 0o2750); err != nil {
		return Snapshot{}, err
	}
	target := filepath.Join(dir, name)
	if out, err := s.runner(ctx, "btrfs", "subvolume", "snapshot", "-r", abs, target); err != nil {
		return Snapshot{}, fmt.Errorf("创建快照失败: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return Snapshot{Name: name, CreatedAt: s.now().Format("2006-01-02 15:04:05")}, nil
}

// ListSnapshots returns the folder's snapshots newest-first.
func (s *Service) ListSnapshots(ctx context.Context, folderID string) ([]Snapshot, error) {
	folder, ok := s.folderByID(folderID)
	if !ok {
		return nil, fmt.Errorf("shared folder not found: %s", folderID)
	}
	abs := s.absPath(ctx, folder)
	dir := snapshotDir(abs, folder.ID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Snapshot{}, nil
		}
		return nil, err
	}
	out := make([]Snapshot, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		created := ""
		if info, err := e.Info(); err == nil {
			created = info.ModTime().Format("2006-01-02 15:04:05")
		}
		out = append(out, Snapshot{Name: e.Name(), CreatedAt: created})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}

// DeleteSnapshot removes a read-only Btrfs snapshot by name.
func (s *Service) DeleteSnapshot(ctx context.Context, folderID, name string) error {
	folder, ok := s.folderByID(folderID)
	if !ok {
		return fmt.Errorf("shared folder not found: %s", folderID)
	}
	name = sanitizeSnapName(name)
	if name == "" {
		return fmt.Errorf("快照名称无效")
	}
	if s.runner == nil || strings.TrimSpace(s.nasRoot) == "" {
		return fmt.Errorf("快照在当前主机不可用")
	}
	abs := s.absPath(ctx, folder)
	target := filepath.Join(snapshotDir(abs, folder.ID), name)
	if out, err := s.runner(ctx, "btrfs", "subvolume", "delete", target); err != nil {
		return fmt.Errorf("删除快照失败: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// sanitizeSnapName keeps snapshot names to a safe single path segment.
func sanitizeSnapName(in string) string {
	in = strings.TrimSpace(in)
	in = filepath.Base(in)
	if in == "." || in == ".." || in == "/" {
		return ""
	}
	in = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		default:
			return '-'
		}
	}, in)
	return strings.TrimPrefix(in, ".")
}
