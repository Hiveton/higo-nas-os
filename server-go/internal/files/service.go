package files

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Repository interface {
	Tree(context.Context) (FileNode, error)
	List(context.Context) ([]FileNode, error)
	Get(context.Context, string) (FileNode, error)
	Put(context.Context, FileNode) error
}

type MutableRepository interface {
	Repository
	CreateFolder(context.Context, CreateFolderRequest) (FileNode, error)
	CreateFile(context.Context, CreateFileRequest) (FileNode, error)
	UploadFile(context.Context, UploadFileRequest) (FileNode, error)
	Rename(context.Context, string, RenameRequest) (FileNode, error)
	Move(context.Context, string, MoveRequest) (FileNode, error)
	Delete(context.Context, string, string) (FileNode, error)
	Restore(context.Context, string) (FileNode, error)
	Open(context.Context, string) (io.ReadCloser, FileNode, error)
}

type Service struct {
	repo Repository
	now  func() time.Time

	mu      sync.Mutex
	pending map[string]pendingBatch
}

// pendingBatch keeps a planned batch task and the operation needed to execute it
// once confirmed, completing the plan→confirm→execute governance loop.
type pendingBatch struct {
	task Task
	op   BatchOperation
}

func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("files repository is required")
	}
	return &Service{
		repo:    repo,
		now:     time.Now,
		pending: map[string]pendingBatch{},
	}, nil
}

func (s *Service) Tree(ctx context.Context, space string) (FileNode, error) {
	tree, err := s.repo.Tree(ctx)
	if err != nil {
		return FileNode{}, err
	}
	space = strings.TrimSpace(space)
	if space == "" {
		return tree, nil
	}
	for _, child := range tree.Children {
		if child.Space == space || child.Name == space {
			return child, nil
		}
	}
	return FileNode{}, fmt.Errorf("space not found: %s", space)
}

func (s *Service) Search(ctx context.Context, query SearchQuery) ([]FileRow, error) {
	nodes, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	type scoredRow struct {
		row   FileRow
		score int
	}
	var rows []scoredRow
	for _, node := range nodes {
		if node.IsDir {
			continue
		}
		if query.Space != "" && node.Space != query.Space {
			continue
		}
		if query.Type != "" && !strings.EqualFold(node.Type, query.Type) {
			continue
		}
		if !hasAllTags(node.Tags, query.Tags) {
			continue
		}
		score := matchScore(node, query.Query)
		if strings.TrimSpace(query.Query) != "" && score == 0 {
			continue
		}
		rows = append(rows, scoredRow{row: rowFromNode(node), score: score})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		if rows[i].row.Space != rows[j].row.Space {
			return spaceOrder(rows[i].row.Space) < spaceOrder(rows[j].row.Space)
		}
		return rows[i].row.Name < rows[j].row.Name
	})
	limit := query.Limit
	if limit <= 0 || limit > len(rows) {
		limit = len(rows)
	}
	out := make([]FileRow, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, rows[i].row)
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id string) (FileRow, error) {
	node, err := s.repo.Get(ctx, id)
	if err != nil {
		return FileRow{}, err
	}
	return rowFromNode(node), nil
}

func (s *Service) Preview(ctx context.Context, id string) (Preview, error) {
	node, err := s.repo.Get(ctx, id)
	if err != nil {
		return Preview{}, err
	}
	preview := Preview{
		FileID: node.ID,
		Name:   node.Name,
		Type:   node.Type,
	}
	switch strings.ToLower(node.Type) {
	case "txt", "md":
		bytes, err := os.ReadFile(node.RealPath)
		if err != nil {
			return Preview{}, err
		}
		text := truncate(string(bytes), 1200)
		return Preview{
			FileID:    node.ID,
			Name:      node.Name,
			Type:      node.Type,
			Supported: true,
			Kind:      "text",
			Summary:   inferSummary(text, node.Name),
			Text:      text,
		}, nil
	default:
		preview.Supported = false
		preview.Kind = strings.ToLower(node.Type)
		preview.Reason = "unsupported preview"
		return preview, nil
	}
}

func (s *Service) CreateFolder(ctx context.Context, request CreateFolderRequest) (FileRow, error) {
	repo, ok := s.repo.(MutableRepository)
	if !ok {
		return FileRow{}, fmt.Errorf("file repository does not support mutations")
	}
	node, err := repo.CreateFolder(ctx, request)
	if err != nil {
		return FileRow{}, err
	}
	return rowFromNode(node), nil
}

func (s *Service) CreateFile(ctx context.Context, request CreateFileRequest) (FileRow, error) {
	repo, ok := s.repo.(MutableRepository)
	if !ok {
		return FileRow{}, fmt.Errorf("file repository does not support mutations")
	}
	node, err := repo.CreateFile(ctx, request)
	if err != nil {
		return FileRow{}, err
	}
	return rowFromNode(node), nil
}

func (s *Service) UploadFile(ctx context.Context, request UploadFileRequest) (FileRow, error) {
	repo, ok := s.repo.(MutableRepository)
	if !ok {
		return FileRow{}, fmt.Errorf("file repository does not support mutations")
	}
	node, err := repo.UploadFile(ctx, request)
	if err != nil {
		return FileRow{}, err
	}
	return rowFromNode(node), nil
}

func (s *Service) Rename(ctx context.Context, id string, request RenameRequest) (FileRow, error) {
	repo, ok := s.repo.(MutableRepository)
	if !ok {
		return FileRow{}, fmt.Errorf("file repository does not support mutations")
	}
	node, err := repo.Rename(ctx, id, request)
	if err != nil {
		return FileRow{}, err
	}
	return rowFromNode(node), nil
}

func (s *Service) Move(ctx context.Context, id string, request MoveRequest) (FileRow, error) {
	repo, ok := s.repo.(MutableRepository)
	if !ok {
		return FileRow{}, fmt.Errorf("file repository does not support mutations")
	}
	node, err := repo.Move(ctx, id, request)
	if err != nil {
		return FileRow{}, err
	}
	return rowFromNode(node), nil
}

func (s *Service) Delete(ctx context.Context, id string, actor string) (FileRow, error) {
	repo, ok := s.repo.(MutableRepository)
	if !ok {
		return FileRow{}, fmt.Errorf("file repository does not support mutations")
	}
	node, err := repo.Delete(ctx, id, actor)
	if err != nil {
		return FileRow{}, err
	}
	return rowFromNode(node), nil
}

func (s *Service) Restore(ctx context.Context, id string) (FileRow, error) {
	repo, ok := s.repo.(MutableRepository)
	if !ok {
		return FileRow{}, fmt.Errorf("file repository does not support mutations")
	}
	node, err := repo.Restore(ctx, id)
	if err != nil {
		return FileRow{}, err
	}
	return rowFromNode(node), nil
}

func (s *Service) Open(ctx context.Context, id string) (io.ReadCloser, FileNode, error) {
	repo, ok := s.repo.(MutableRepository)
	if !ok {
		return nil, FileNode{}, fmt.Errorf("file repository does not support downloads")
	}
	return repo.Open(ctx, id)
}

func (s *Service) AddTags(ctx context.Context, mutation TagMutation) (FileRow, error) {
	node, err := s.repo.Get(ctx, mutation.FileID)
	if err != nil {
		return FileRow{}, err
	}
	for _, tag := range mutation.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || stringSliceContains(node.Tags, tag) {
			continue
		}
		node.Tags = append(node.Tags, tag)
	}
	if mutation.Audit == "" {
		mutation.Audit = fmt.Sprintf("%s added %d tag(s) via %s", emptyDefault(mutation.Actor, "system"), len(mutation.Tags), emptyDefault(mutation.Source, "manual"))
	}
	if err := s.repo.Put(ctx, node); err != nil {
		return FileRow{}, err
	}
	return rowFromNode(node), nil
}

func (s *Service) CreateShare(ctx context.Context, request ShareLink) (ShareLink, error) {
	if _, err := s.repo.Get(ctx, request.FileID); err != nil {
		return ShareLink{}, err
	}
	if request.ID == "" {
		request.ID = "share_" + randomHex(8)
	}
	if request.ExpiresAt.IsZero() {
		request.ExpiresAt = s.now().Add(7 * 24 * time.Hour)
	}
	if request.URL == "" {
		request.URL = "/shares/" + request.ID
	}
	if request.Audit == "" {
		request.Audit = fmt.Sprintf("share link %s created for file %s", request.ID, request.FileID)
	}
	return request, nil
}

func (s *Service) BatchMove(ctx context.Context, op BatchOperation) (Task, error) {
	op.Type = "move"
	return s.planBatch(ctx, op, func(node FileNode) (string, string, string) {
		target := "/" + strings.Trim(strings.TrimSpace(op.Destination), "/")
		if target == "/" {
			target = filepath.ToSlash(filepath.Dir(node.Path))
		}
		return "move", node.Path, filepath.ToSlash(filepath.Join(target, filepath.Base(node.Path)))
	})
}

func (s *Service) BatchRename(ctx context.Context, op BatchOperation) (Task, error) {
	op.Type = "rename"
	return s.planBatch(ctx, op, func(node FileNode) (string, string, string) {
		newName := strings.TrimSpace(op.Rename[node.ID])
		if newName == "" {
			newName = node.Name
		}
		return "rename", node.Path, filepath.ToSlash(filepath.Join(filepath.Dir(node.Path), newName))
	})
}

func (s *Service) BatchDelete(ctx context.Context, op BatchOperation) (Task, error) {
	op.Type = "delete"
	task, err := s.planBatch(ctx, op, func(node FileNode) (string, string, string) {
		return "delete", node.Path, filepath.ToSlash(filepath.Join("/回收站", strings.TrimPrefix(node.Path, "/")))
	})
	if err != nil {
		return Task{}, err
	}
	task.RequiresConfirmation = true
	return task, nil
}

func (s *Service) planBatch(ctx context.Context, op BatchOperation, build func(FileNode) (string, string, string)) (Task, error) {
	if len(op.FileIDs) == 0 {
		return Task{}, fmt.Errorf("batch operation needs at least one file id")
	}
	now := s.now()
	task := Task{
		ID:             "task_" + randomHex(8),
		Type:           op.Type,
		Status:         "planned",
		Actor:          emptyDefault(op.Actor, "system"),
		FileIDs:        append([]string(nil), op.FileIDs...),
		CreatedAt:      now,
		RollbackPlan:   RollbackPlan{ID: "rollback_" + randomHex(8), CreatedAt: now},
		PlannedActions: make([]PlannedAction, 0, len(op.FileIDs)),
	}
	for _, id := range op.FileIDs {
		node, err := s.repo.Get(ctx, id)
		if err != nil {
			return Task{}, err
		}
		action, from, to := build(node)
		task.PlannedActions = append(task.PlannedActions, PlannedAction{
			FileID: node.ID,
			From:   from,
			To:     to,
			Action: action,
		})
		task.RollbackPlan.Steps = append(task.RollbackPlan.Steps, RollbackStep{
			FileID: node.ID,
			From:   to,
			To:     from,
			Action: "rollback_" + action,
		})
	}
	s.mu.Lock()
	s.pending[task.ID] = pendingBatch{task: task, op: op}
	s.mu.Unlock()
	return task, nil
}

// ExecuteBatch runs a previously planned batch task, performing the real file
// operations (reusing the single-file Move/Rename/Delete paths). This is the
// confirm→execute half of the batch governance loop; planning alone never
// mutates files. The task is consumed on success.
func (s *Service) ExecuteBatch(ctx context.Context, taskID string, actor string) (Task, error) {
	s.mu.Lock()
	pending, ok := s.pending[taskID]
	s.mu.Unlock()
	if !ok {
		return Task{}, fmt.Errorf("planned batch task not found: %s", taskID)
	}
	if strings.TrimSpace(actor) == "" {
		actor = pending.op.Actor
	}
	for _, fileID := range pending.op.FileIDs {
		var err error
		switch pending.op.Type {
		case "move":
			_, err = s.Move(ctx, fileID, MoveRequest{Destination: pending.op.Destination})
		case "rename":
			name := strings.TrimSpace(pending.op.Rename[fileID])
			if name == "" {
				continue
			}
			_, err = s.Rename(ctx, fileID, RenameRequest{Name: name})
		case "delete":
			_, err = s.Delete(ctx, fileID, emptyDefault(actor, "system"))
		default:
			return Task{}, fmt.Errorf("unsupported batch type: %s", pending.op.Type)
		}
		if err != nil {
			return Task{}, fmt.Errorf("batch execute failed for %s: %w", fileID, err)
		}
	}
	task := pending.task
	task.Status = "completed"
	s.mu.Lock()
	delete(s.pending, taskID)
	s.mu.Unlock()
	return task, nil
}

func rowFromNode(node FileNode) FileRow {
	return FileRow{
		ID:         node.ID,
		Name:       node.Name,
		Path:       node.Path,
		Type:       node.Type,
		Space:      node.Space,
		Size:       node.Size,
		Modified:   formatModified(node.Modified),
		Tags:       append([]string(nil), node.Tags...),
		Permission: node.Permission,
		AISummary:  node.Summary,
		IsDir:      node.IsDir,
	}
}

func matchScore(node FileNode, query string) int {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		return 1
	}
	haystack := strings.ToLower(strings.Join([]string{
		node.Name,
		node.Path,
		node.RealPath,
		node.Type,
		node.Space,
		strings.Join(node.Tags, " "),
		node.Summary,
	}, " "))
	score := 0
	for _, term := range terms {
		if strings.Contains(haystack, term) {
			score++
		}
	}
	return score
}

func hasAllTags(have []string, want []string) bool {
	for _, tag := range want {
		if !stringSliceContains(have, tag) {
			return false
		}
	}
	return true
}

func emptyDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func randomHex(bytes int) string {
	if bytes <= 0 {
		bytes = 8
	}
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
