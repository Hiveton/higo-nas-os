package files

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func fixtureRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "fixtures", "nas-root"))
	if err != nil {
		t.Fatalf("fixture root: %v", err)
	}
	return root
}

func newFixtureService(t *testing.T) *Service {
	t.Helper()
	repo, err := NewFixtureRepository(fixtureRoot(t))
	if err != nil {
		t.Fatalf("repository: %v", err)
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	return service
}

func TestFixtureRepositoryScansChineseSpaces(t *testing.T) {
	service := newFixtureService(t)

	tree, err := service.Tree(context.Background(), "")
	if err != nil {
		t.Fatalf("tree: %v", err)
	}

	wantSpaces := []string{"家庭空间", "团队空间", "财务票据", "照片与视频", "备份归档", "下载目录"}
	if len(tree.Children) != len(wantSpaces) {
		t.Fatalf("space roots = %d, want %d", len(tree.Children), len(wantSpaces))
	}
	seen := map[string]bool{}
	for _, node := range tree.Children {
		seen[node.Space] = true
		if !node.IsDir || node.Type != "文件夹" {
			t.Fatalf("space node %q is dir=%v type=%q", node.Space, node.IsDir, node.Type)
		}
	}
	for _, space := range wantSpaces {
		if !seen[space] {
			t.Fatalf("missing mapped space %q in tree %#v", space, tree.Children)
		}
	}
}

func TestSearchMatchesNamePathTypeSpaceTagsAndSummary(t *testing.T) {
	service := newFixtureService(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		query      SearchQuery
		wantName   string
		wantSpace  string
		wantTag    string
		wantSubstr string
	}{
		{
			name:      "contract by query and team space",
			query:     SearchQuery{Query: "合同", Space: "团队空间"},
			wantName:  "客户 A 合同最终版",
			wantSpace: "团队空间",
			wantTag:   "合同",
		},
		{
			name:      "invoice by summary and finance space",
			query:     SearchQuery{Query: "发票", Space: "财务票据"},
			wantName:  "未归档发票",
			wantSpace: "财务票据",
			wantTag:   "发票",
		},
		{
			name:      "warranty by tags and summary",
			query:     SearchQuery{Query: "保修", Tags: []string{"保修"}},
			wantName:  "2026 家庭保险与保修资料",
			wantSpace: "家庭空间",
			wantTag:   "保修",
		},
		{
			name:       "text type by path and extension",
			query:      SearchQuery{Query: "downloads", Type: "TXT"},
			wantName:   "unarchived-invoices",
			wantSpace:  "下载目录",
			wantSubstr: "invoice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := service.Search(ctx, tt.query)
			if err != nil {
				t.Fatalf("search: %v", err)
			}
			if len(rows) == 0 {
				t.Fatalf("no results for %#v", tt.query)
			}
			got := rows[0]
			if got.Name != tt.wantName || got.Space != tt.wantSpace {
				t.Fatalf("top result = %q/%q, want %q/%q", got.Space, got.Name, tt.wantSpace, tt.wantName)
			}
			if tt.wantTag != "" && !contains(got.Tags, tt.wantTag) {
				t.Fatalf("tags %v missing %q", got.Tags, tt.wantTag)
			}
			if tt.wantSubstr != "" && got.AISummary == "" {
				t.Fatalf("summary missing substring context %q in %#v", tt.wantSubstr, got)
			}
		})
	}
}

func TestPreviewReturnsTextSummaryAndUnsupported(t *testing.T) {
	service := newFixtureService(t)
	ctx := context.Background()

	rows, err := service.Search(ctx, SearchQuery{Query: "保修"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	preview, err := service.Preview(ctx, rows[0].ID)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if !preview.Supported || preview.Kind != "text" {
		t.Fatalf("preview supported=%v kind=%q", preview.Supported, preview.Kind)
	}
	if preview.Summary == "" || preview.Text == "" {
		t.Fatalf("preview missing text summary: %#v", preview)
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "home-space"), 0o755); err != nil {
		t.Fatalf("mkdir temp space: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "home-space", "report.pdf"), []byte("%PDF fixture"), 0o644); err != nil {
		t.Fatalf("write temp pdf: %v", err)
	}
	pdfRepo, err := NewFixtureRepository(root)
	if err != nil {
		t.Fatalf("pdf repo: %v", err)
	}
	pdfService, err := NewService(pdfRepo)
	if err != nil {
		t.Fatalf("pdf service: %v", err)
	}
	pdfRows, err := pdfService.Search(ctx, SearchQuery{Type: "PDF"})
	if err != nil {
		t.Fatalf("pdf search: %v", err)
	}
	unsupported, err := pdfService.Preview(ctx, pdfRows[0].ID)
	if err != nil {
		t.Fatalf("unsupported preview should return payload without error: %v", err)
	}
	if unsupported.Supported || unsupported.Reason == "" {
		t.Fatalf("unsupported preview missing safe response: %#v", unsupported)
	}
}

func TestAddTagsAppendsWithoutDuplicates(t *testing.T) {
	service := newFixtureService(t)
	ctx := context.Background()
	rows, err := service.Search(ctx, SearchQuery{Query: "合同"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	updated, err := service.AddTags(ctx, TagMutation{
		FileID: rows[0].ID,
		Tags:   []string{"AI 已处理", "合同"},
		Actor:  "worker-g-test",
		Source: "test",
	})
	if err != nil {
		t.Fatalf("add tags: %v", err)
	}
	if !contains(updated.Tags, "AI 已处理") {
		t.Fatalf("updated tags missing appended tag: %v", updated.Tags)
	}
	if count(updated.Tags, "合同") != 1 {
		t.Fatalf("duplicate contract tag in %v", updated.Tags)
	}

	got, err := service.Get(ctx, rows[0].ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !contains(got.Tags, "AI 已处理") {
		t.Fatalf("persisted tags missing appended tag: %v", got.Tags)
	}
}

func TestCreateShareReturnsAuditedLink(t *testing.T) {
	service := newFixtureService(t)
	ctx := context.Background()
	rows, err := service.Search(ctx, SearchQuery{Query: "合同"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	link, err := service.CreateShare(ctx, ShareLink{
		FileID:        rows[0].ID,
		Password:      "123456",
		DownloadLimit: 3,
	})
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	if link.ID == "" || link.URL == "" || link.Audit == "" {
		t.Fatalf("share link missing generated fields: %#v", link)
	}
	if link.FileID != rows[0].ID || link.DownloadLimit != 3 || link.Revoked {
		t.Fatalf("share link did not preserve request fields: %#v", link)
	}
}

func TestBatchOperationsReturnRollbackPlansWithoutMovingFiles(t *testing.T) {
	service := newFixtureService(t)
	ctx := context.Background()
	rows, err := service.Search(ctx, SearchQuery{Query: "发票"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(rows) < 1 {
		t.Fatal("need invoice file for batch operation")
	}
	before, err := service.Get(ctx, rows[0].ID)
	if err != nil {
		t.Fatalf("get before: %v", err)
	}

	move, err := service.BatchMove(ctx, BatchOperation{
		FileIDs:     []string{rows[0].ID},
		Destination: "财务票据/2026/04",
		Actor:       "worker-g-test",
	})
	if err != nil {
		t.Fatalf("batch move: %v", err)
	}
	if move.ID == "" || len(move.RollbackPlan.Steps) == 0 {
		t.Fatalf("move task missing rollback plan: %#v", move)
	}

	rename, err := service.BatchRename(ctx, BatchOperation{
		FileIDs: []string{rows[0].ID},
		Rename:  map[string]string{rows[0].ID: "invoice-archive.md"},
		Actor:   "worker-g-test",
	})
	if err != nil {
		t.Fatalf("batch rename: %v", err)
	}
	if rename.RollbackPlan.Steps[0].From == "" || rename.RollbackPlan.Steps[0].To == "" {
		t.Fatalf("rename rollback step missing paths: %#v", rename.RollbackPlan.Steps[0])
	}

	deleteTask, err := service.BatchDelete(ctx, BatchOperation{
		FileIDs: []string{rows[0].ID},
		Actor:   "worker-g-test",
	})
	if err != nil {
		t.Fatalf("batch delete: %v", err)
	}
	if !deleteTask.RequiresConfirmation || deleteTask.RollbackPlan.ID == "" {
		t.Fatalf("delete task should require confirmation and rollback: %#v", deleteTask)
	}

	after, err := service.Get(ctx, rows[0].ID)
	if err != nil {
		t.Fatalf("get after: %v", err)
	}
	if after.Path != before.Path || after.Name != before.Name {
		t.Fatalf("batch preview mutated file: before=%#v after=%#v", before, after)
	}
}

func TestRootRepositoryCreatesMovesRenamesDeletesAndDownloads(t *testing.T) {
	root := t.TempDir()
	repo, err := NewRootRepository(root)
	if err != nil {
		t.Fatalf("root repo: %v", err)
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	ctx := context.Background()

	folder, err := service.CreateFolder(ctx, CreateFolderRequest{Space: "家庭空间", Name: "documents"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if !folder.IsDir || folder.Space != "家庭空间" {
		t.Fatalf("unexpected folder row: %#v", folder)
	}

	file, err := service.CreateFile(ctx, CreateFileRequest{Path: "家庭空间/documents", Name: "note.md", Content: "# NAS Note\nhello"})
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	uploaded, err := service.UploadFile(ctx, UploadFileRequest{
		Path:    "家庭空间/documents",
		Name:    "photo.bin",
		Content: bytes.NewReader([]byte{0x00, 0x01, 0x02, 0xff}),
	})
	if err != nil {
		t.Fatalf("upload file: %v", err)
	}
	if uploaded.Size != "4 B" || uploaded.Type != "BIN" {
		t.Fatalf("unexpected uploaded row: %#v", uploaded)
	}
	tree, err := service.Tree(ctx, "")
	if err != nil {
		t.Fatalf("tree after upload: %v", err)
	}
	if !treeContainsPath(tree, "/家庭空间/documents/photo.bin") {
		t.Fatalf("uploaded file missing from nested tree: %#v", tree)
	}
	renamed, err := service.Rename(ctx, file.ID, RenameRequest{Name: "renamed.md"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Name != "NAS Note" || renamed.Type != "MD" {
		t.Fatalf("unexpected renamed row: %#v", renamed)
	}
	moved, err := service.Move(ctx, renamed.ID, MoveRequest{Destination: "下载目录"})
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if moved.Space != "下载目录" {
		t.Fatalf("expected moved file in downloads, got %#v", moved)
	}
	reader, node, err := service.Open(ctx, moved.ID)
	if err != nil {
		t.Fatalf("open moved file: %v", err)
	}
	_ = reader.Close()
	if node.SizeBytes == 0 {
		t.Fatalf("download node missing size: %#v", node)
	}
	deleted, err := service.Delete(ctx, moved.ID, "test")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted.Path == moved.Path {
		t.Fatalf("delete should move to recycle path: before=%s after=%s", moved.Path, deleted.Path)
	}
}

func TestRootRepositoryRestoreReturnsDeletedFileToOriginalLocation(t *testing.T) {
	root := t.TempDir()
	repo, err := NewRootRepository(root)
	if err != nil {
		t.Fatalf("root repo: %v", err)
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	ctx := context.Background()

	if _, err := service.CreateFolder(ctx, CreateFolderRequest{Space: "家庭空间", Name: "documents"}); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	file, err := service.CreateFile(ctx, CreateFileRequest{Path: "家庭空间/documents", Name: "note.md", Content: "hello"})
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	originalID, originalPath := file.ID, file.Path

	if _, err := service.Delete(ctx, file.ID, "test"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := service.Get(ctx, originalID); err == nil {
		t.Fatalf("file should be gone from tree after delete")
	}

	restored, err := service.Restore(ctx, originalID)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.ID != originalID {
		t.Fatalf("restore should recover original id: got %s want %s", restored.ID, originalID)
	}
	if restored.Path != originalPath {
		t.Fatalf("restore should return file to original path: got %s want %s", restored.Path, originalPath)
	}
	if _, err := service.Get(ctx, originalID); err != nil {
		t.Fatalf("restored file should be back in tree: %v", err)
	}

	// Restoring again must fail cleanly: the manifest is consumed.
	if _, err := service.Restore(ctx, originalID); err == nil {
		t.Fatalf("second restore should fail, manifest already consumed")
	}
}

func TestRootRepositoryRestoreDedupesWhenOriginalPathOccupied(t *testing.T) {
	root := t.TempDir()
	repo, err := NewRootRepository(root)
	if err != nil {
		t.Fatalf("root repo: %v", err)
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	ctx := context.Background()

	if _, err := service.CreateFolder(ctx, CreateFolderRequest{Space: "家庭空间", Name: "documents"}); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	file, err := service.CreateFile(ctx, CreateFileRequest{Path: "家庭空间/documents", Name: "note.md", Content: "v1"})
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	if _, err := service.Delete(ctx, file.ID, "test"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// Re-create a file at the same original location before restoring.
	if _, err := service.CreateFile(ctx, CreateFileRequest{Path: "家庭空间/documents", Name: "note.md", Content: "v2"}); err != nil {
		t.Fatalf("recreate file: %v", err)
	}

	restored, err := service.Restore(ctx, file.ID)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.Path == file.Path {
		t.Fatalf("restore should not clobber existing file; expected deduped path, got %s", restored.Path)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func count(values []string, want string) int {
	var total int
	for _, value := range values {
		if value == want {
			total++
		}
	}
	return total
}

func treeContainsPath(node FileNode, want string) bool {
	if node.Path == want {
		return true
	}
	for _, child := range node.Children {
		if treeContainsPath(child, want) {
			return true
		}
	}
	return false
}
