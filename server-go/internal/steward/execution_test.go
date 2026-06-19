package steward

import (
	"context"
	"testing"

	"higoos/server-go/internal/files"
)

// newFilesService builds a real, mutation-capable files service over a temp dir.
func newFilesService(t *testing.T) (*files.Service, string) {
	t.Helper()
	root := t.TempDir()
	repo, err := files.NewRootRepository(root)
	if err != nil {
		t.Fatalf("root repo: %v", err)
	}
	svc, err := files.NewService(repo)
	if err != nil {
		t.Fatalf("files service: %v", err)
	}
	return svc, root
}

func TestConfirmExecutesRealDeleteAndRollbackRestores(t *testing.T) {
	filesSvc, _ := newFilesService(t)
	ctx := context.Background()
	if _, err := filesSvc.CreateFolder(ctx, files.CreateFolderRequest{Space: "家庭空间", Name: "documents"}); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	file, err := filesSvc.CreateFile(ctx, files.CreateFileRequest{Path: "家庭空间/documents", Name: "dup.md", Content: "duplicate"})
	if err != nil {
		t.Fatalf("create file: %v", err)
	}

	steward := NewService()
	steward.AttachFiles(filesSvc)

	if _, err := steward.ReplaceSuggestions(ctx, []Suggestion{{
		ID:         "cleanup-dup",
		Title:      "清理重复文件",
		Risk:       RiskMedium,
		Operations: []SuggestionOp{{Type: "delete", FileID: file.ID}},
	}}); err != nil {
		t.Fatalf("replace suggestions: %v", err)
	}

	preview, err := steward.Preview(ctx, "cleanup-dup", PreviewRequest{ActorID: "admin"})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.ConfirmationID == "" || preview.RollbackID == "" {
		t.Fatalf("medium-risk preview should yield confirmation + rollback ids: %#v", preview)
	}

	result, err := steward.Confirm(ctx, "cleanup-dup", ConfirmRequest{ActorID: "admin", ConfirmationID: preview.ConfirmationID})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if len(result.AuditEntry.ExecutedFileIDs) != 1 || result.AuditEntry.ExecutedFileIDs[0] != file.ID {
		t.Fatalf("audit should record the executed delete: %#v", result.AuditEntry)
	}
	// The file was really deleted (moved to recycle): its original id is gone.
	if _, err := filesSvc.Get(ctx, file.ID); err == nil {
		t.Fatalf("file should be deleted after confirm")
	}

	// Roll back: the file is restored from the recycle bin.
	if _, err := steward.Rollback(ctx, result.AuditEntry.ID, RollbackRequest{ActorID: "admin"}); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if _, err := filesSvc.Get(ctx, file.ID); err != nil {
		t.Fatalf("file should be restored after rollback: %v", err)
	}
}

func TestConfirmExecutesRealMoveAndRollbackRestoresPath(t *testing.T) {
	filesSvc, _ := newFilesService(t)
	ctx := context.Background()
	if _, err := filesSvc.CreateFolder(ctx, files.CreateFolderRequest{Space: "家庭空间", Name: "cold"}); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	file, err := filesSvc.CreateFile(ctx, files.CreateFileRequest{Path: "家庭空间/cold", Name: "big.bin", Content: "x"})
	if err != nil {
		t.Fatalf("create file: %v", err)
	}

	steward := NewService()
	steward.AttachFiles(filesSvc)
	if _, err := steward.ReplaceSuggestions(ctx, []Suggestion{{
		ID:         "archive-candidates",
		Title:      "归档大文件与陈旧文件",
		Risk:       RiskLow, // low risk: no preview/confirmation token required
		Operations: []SuggestionOp{{Type: "move", FileID: file.ID, Dest: archiveSpace}},
	}}); err != nil {
		t.Fatalf("replace suggestions: %v", err)
	}

	result, err := steward.Confirm(ctx, "archive-candidates", ConfirmRequest{ActorID: "admin"})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if len(result.AuditEntry.ExecutedOps) != 1 || result.AuditEntry.ExecutedOps[0].Type != "move" {
		t.Fatalf("audit should record the executed move: %#v", result.AuditEntry.ExecutedOps)
	}
	movedID := result.AuditEntry.ExecutedOps[0].FileID
	// The file now lives in the archive space under a new (path-derived) id.
	if movedID == file.ID {
		t.Fatalf("move should change the file id")
	}
	moved, err := filesSvc.Get(ctx, movedID)
	if err != nil {
		t.Fatalf("moved file should exist under new id: %v", err)
	}
	if moved.Space != archiveSpace {
		t.Fatalf("file should be in %s, got %q", archiveSpace, moved.Space)
	}

	// Roll back: the file is moved back to its original directory.
	if _, err := steward.Rollback(ctx, result.AuditEntry.ID, RollbackRequest{ActorID: "admin"}); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if _, err := filesSvc.Get(ctx, file.ID); err != nil {
		t.Fatalf("file should be back at its original path/id after rollback: %v", err)
	}
}

func TestConfirmWithoutFilesExecutorKeepsLegacyBehaviour(t *testing.T) {
	steward := NewService() // no files executor attached
	ctx := context.Background()
	if _, err := steward.ReplaceSuggestions(ctx, []Suggestion{{
		ID:         "noop",
		Title:      "演示建议",
		Risk:       RiskMedium,
		Operations: []SuggestionOp{{Type: "delete", FileID: "whatever"}},
	}}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	preview, err := steward.Preview(ctx, "noop", PreviewRequest{ActorID: "admin"})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	result, err := steward.Confirm(ctx, "noop", ConfirmRequest{ActorID: "admin", ConfirmationID: preview.ConfirmationID})
	if err != nil {
		t.Fatalf("confirm should succeed without a files executor: %v", err)
	}
	if len(result.AuditEntry.ExecutedFileIDs) != 0 {
		t.Fatalf("no operations should execute without a files executor: %#v", result.AuditEntry)
	}
}
