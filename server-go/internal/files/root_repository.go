package files

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// recycleMeta is the sidecar manifest written next to each recycled entry so a
// later Restore can move it back to its original location and recover its ID.
type recycleMeta struct {
	OriginalID   string    `json:"originalId"`
	OriginalPath string    `json:"originalPath"`
	OriginalReal string    `json:"originalRealPath"`
	Space        string    `json:"space"`
	RecycleName  string    `json:"recycleName"`
	DeletedAt    time.Time `json:"deletedAt"`
}

const recycleMetaSuffix = ".meta.json"

type RootRepository struct {
	root string

	mu    sync.RWMutex
	nodes map[string]FileNode
	tree  FileNode
}

func NewRootRepository(root string) (*RootRepository, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("nas root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	repo := &RootRepository{root: abs}
	if err := repo.ensureDefaultSpaces(); err != nil {
		return nil, err
	}
	if err := repo.rescanLocked(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *RootRepository) Tree(ctx context.Context) (FileNode, error) {
	if err := ctx.Err(); err != nil {
		return FileNode{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneNode(r.tree), nil
}

func (r *RootRepository) List(ctx context.Context) ([]FileNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	nodes := make([]FileNode, 0, len(r.nodes))
	for _, node := range r.nodes {
		nodes = append(nodes, cloneNode(node))
	}
	sortNodes(nodes)
	return nodes, nil
}

func (r *RootRepository) Get(ctx context.Context, id string) (FileNode, error) {
	if err := ctx.Err(); err != nil {
		return FileNode{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	node, ok := r.nodes[id]
	if !ok {
		return FileNode{}, fmt.Errorf("file node not found: %s", id)
	}
	return cloneNode(node), nil
}

func (r *RootRepository) Put(ctx context.Context, node FileNode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.nodes[node.ID]
	if !ok {
		return fmt.Errorf("file node not found: %s", node.ID)
	}
	current.Tags = append([]string(nil), node.Tags...)
	current.Permission = node.Permission
	current.Summary = node.Summary
	r.nodes[current.ID] = cloneNode(current)
	return r.replaceTreeNodeLocked(current.ID, current)
}

func (r *RootRepository) CreateFolder(ctx context.Context, request CreateFolderRequest) (FileNode, error) {
	if err := ctx.Err(); err != nil {
		return FileNode{}, err
	}
	if strings.TrimSpace(request.Name) == "" {
		return FileNode{}, errors.New("folder name is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	parent, err := r.resolveDestinationLocked(request.Space, request.Path)
	if err != nil {
		return FileNode{}, err
	}
	target, err := r.safeJoin(parent, request.Name)
	if err != nil {
		return FileNode{}, err
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		return FileNode{}, err
	}
	if err := r.rescanLocked(); err != nil {
		return FileNode{}, err
	}
	return r.nodeByRealPathLocked(target)
}

func (r *RootRepository) CreateFile(ctx context.Context, request CreateFileRequest) (FileNode, error) {
	if err := ctx.Err(); err != nil {
		return FileNode{}, err
	}
	if strings.TrimSpace(request.Name) == "" {
		return FileNode{}, errors.New("file name is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	parent, err := r.resolveDestinationLocked(request.Space, request.Path)
	if err != nil {
		return FileNode{}, err
	}
	target, err := r.safeJoin(parent, request.Name)
	if err != nil {
		return FileNode{}, err
	}
	if err := os.WriteFile(target, []byte(request.Content), 0o644); err != nil {
		return FileNode{}, err
	}
	if err := r.rescanLocked(); err != nil {
		return FileNode{}, err
	}
	return r.nodeByRealPathLocked(target)
}

func (r *RootRepository) UploadFile(ctx context.Context, request UploadFileRequest) (FileNode, error) {
	if err := ctx.Err(); err != nil {
		return FileNode{}, err
	}
	name := cleanLeafName(request.Name)
	if name == "" {
		return FileNode{}, errors.New("file name is required")
	}
	if request.Content == nil {
		return FileNode{}, errors.New("file content is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	parent, err := r.resolveDestinationLocked(request.Space, request.Path)
	if err != nil {
		return FileNode{}, err
	}
	target, err := r.safeJoin(parent, name)
	if err != nil {
		return FileNode{}, err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return FileNode{}, err
	}
	_, copyErr := io.Copy(file, request.Content)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(target)
		return FileNode{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(target)
		return FileNode{}, closeErr
	}
	if err := r.rescanLocked(); err != nil {
		return FileNode{}, err
	}
	return r.nodeByRealPathLocked(target)
}

func (r *RootRepository) Rename(ctx context.Context, id string, request RenameRequest) (FileNode, error) {
	if err := ctx.Err(); err != nil {
		return FileNode{}, err
	}
	if strings.TrimSpace(request.Name) == "" {
		return FileNode{}, errors.New("new name is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	node, err := r.getLocked(id)
	if err != nil {
		return FileNode{}, err
	}
	target, err := r.safeJoin(filepath.Dir(node.RealPath), request.Name)
	if err != nil {
		return FileNode{}, err
	}
	if err := os.Rename(node.RealPath, target); err != nil {
		return FileNode{}, err
	}
	if err := r.rescanLocked(); err != nil {
		return FileNode{}, err
	}
	return r.nodeByRealPathLocked(target)
}

func (r *RootRepository) Move(ctx context.Context, id string, request MoveRequest) (FileNode, error) {
	if err := ctx.Err(); err != nil {
		return FileNode{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	node, err := r.getLocked(id)
	if err != nil {
		return FileNode{}, err
	}
	parent, err := r.resolveDestinationLocked("", request.Destination)
	if err != nil {
		return FileNode{}, err
	}
	target, err := r.safeJoin(parent, filepath.Base(node.RealPath))
	if err != nil {
		return FileNode{}, err
	}
	if err := os.Rename(node.RealPath, target); err != nil {
		return FileNode{}, err
	}
	if err := r.rescanLocked(); err != nil {
		return FileNode{}, err
	}
	return r.nodeByRealPathLocked(target)
}

func (r *RootRepository) Delete(ctx context.Context, id string, actor string) (FileNode, error) {
	if err := ctx.Err(); err != nil {
		return FileNode{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	node, err := r.getLocked(id)
	if err != nil {
		return FileNode{}, err
	}
	recycleRoot := filepath.Join(r.root, ".recycle")
	if err := os.MkdirAll(recycleRoot, 0o755); err != nil {
		return FileNode{}, err
	}
	recycleName := time.Now().UTC().Format("20060102T150405") + "-" + filepath.Base(node.RealPath)
	target, err := r.safeJoin(recycleRoot, recycleName)
	if err != nil {
		return FileNode{}, err
	}
	originalReal := node.RealPath
	if err := os.Rename(originalReal, target); err != nil {
		return FileNode{}, err
	}
	// Persist a sidecar manifest so the entry can be restored later. A failure
	// here is non-fatal: the file is already recycled, it just won't be
	// restorable, so we surface it rather than rolling back the delete.
	meta := recycleMeta{
		OriginalID:   node.ID,
		OriginalPath: node.Path,
		OriginalReal: originalReal,
		Space:        node.Space,
		RecycleName:  filepath.Base(target),
		DeletedAt:    time.Now().UTC(),
	}
	if payload, mErr := json.MarshalIndent(meta, "", "  "); mErr == nil {
		_ = os.WriteFile(target+recycleMetaSuffix, payload, 0o644)
	}
	if err := r.rescanLocked(); err != nil {
		return FileNode{}, err
	}
	node.Path = "/回收站/" + filepath.Base(target)
	node.RealPath = target
	return node, nil
}

// Restore moves a previously deleted entry from the recycle bin back to its
// original location, recovering its original ID. When the original location is
// already occupied the entry is restored under a deduplicated name so nothing
// is clobbered.
func (r *RootRepository) Restore(ctx context.Context, id string) (FileNode, error) {
	if err := ctx.Err(); err != nil {
		return FileNode{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return FileNode{}, errors.New("file id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	recycleRoot := filepath.Join(r.root, ".recycle")
	entries, err := os.ReadDir(recycleRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return FileNode{}, fmt.Errorf("nothing to restore: %s", id)
		}
		return FileNode{}, err
	}

	// Find the most recent recycle manifest matching this original id.
	var (
		match     recycleMeta
		matchMeta string
		found     bool
	)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), recycleMetaSuffix) {
			continue
		}
		metaPath := filepath.Join(recycleRoot, entry.Name())
		payload, readErr := os.ReadFile(metaPath)
		if readErr != nil {
			continue
		}
		var meta recycleMeta
		if json.Unmarshal(payload, &meta) != nil {
			continue
		}
		if meta.OriginalID != id {
			continue
		}
		if !found || meta.DeletedAt.After(match.DeletedAt) {
			match, matchMeta, found = meta, metaPath, true
		}
	}
	if !found {
		return FileNode{}, fmt.Errorf("no recoverable entry found for id: %s", id)
	}

	source := filepath.Join(recycleRoot, match.RecycleName)
	if _, err := os.Stat(source); err != nil {
		return FileNode{}, fmt.Errorf("recycled entry missing on disk: %w", err)
	}
	dest := match.OriginalReal
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return FileNode{}, err
	}
	dest = dedupeRestorePath(dest)
	if err := os.Rename(source, dest); err != nil {
		return FileNode{}, err
	}
	_ = os.Remove(matchMeta)
	if err := r.rescanLocked(); err != nil {
		return FileNode{}, err
	}
	return r.nodeByRealPathLocked(dest)
}

// dedupeRestorePath returns path unchanged when nothing exists there, otherwise
// appends a numeric suffix (before any extension) until a free path is found.
func dedupeRestorePath(path string) string {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-恢复%d%s", base, i, ext)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
	return path
}

func (r *RootRepository) Open(ctx context.Context, id string) (io.ReadCloser, FileNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, FileNode{}, err
	}
	r.mu.RLock()
	node, err := r.getLocked(id)
	r.mu.RUnlock()
	if err != nil {
		return nil, FileNode{}, err
	}
	if node.IsDir {
		return nil, FileNode{}, errors.New("cannot download a directory")
	}
	file, err := os.Open(node.RealPath)
	if err != nil {
		return nil, FileNode{}, err
	}
	return file, node, nil
}

func (r *RootRepository) ensureDefaultSpaces() error {
	for _, dir := range orderedSpaceDirs {
		if err := os.MkdirAll(filepath.Join(r.root, dir), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (r *RootRepository) rescanLocked() error {
	nodes := map[string]FileNode{}
	tree := FileNode{
		ID:    stableID("root"),
		Name:  "HiGoNAS",
		Path:  "/",
		Type:  "文件夹",
		Size:  "-",
		IsDir: true,
	}
	entries, err := os.ReadDir(r.root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		spaceNode, err := r.scanSpaceLocked(entry.Name(), nodes)
		if err != nil {
			return err
		}
		tree.Children = append(tree.Children, spaceNode)
	}
	sortNodes(tree.Children)
	r.nodes = nodes
	r.tree = tree
	return nil
}

func (r *RootRepository) scanSpaceLocked(dirName string, nodes map[string]FileNode) (FileNode, error) {
	space := spaceNames[dirName]
	if space == "" {
		space = dirName
	}
	spacePath := filepath.Join(r.root, dirName)
	info, err := os.Stat(spacePath)
	if err != nil {
		return FileNode{}, err
	}
	node := FileNode{
		ID:         stableID(dirName),
		Name:       space,
		Path:       "/" + space,
		RealPath:   spacePath,
		Type:       "文件夹",
		Space:      space,
		Size:       "-",
		Modified:   info.ModTime(),
		Permission: inferPermission(space, "", ""),
		IsDir:      true,
	}
	nodes[node.ID] = cloneNode(node)
	err = filepath.WalkDir(spacePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == spacePath {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		child, err := r.nodeFromEntry(path, d, dirName, space)
		if err != nil {
			return err
		}
		nodes[child.ID] = cloneNode(child)
		return nil
	})
	if err != nil {
		return FileNode{}, err
	}

	childrenByParent := map[string][]string{}
	for childID, child := range nodes {
		if childID == node.ID || child.RealPath == "" {
			continue
		}
		relToSpace, err := filepath.Rel(spacePath, child.RealPath)
		if err != nil || relToSpace == "." || relToSpace == ".." || strings.HasPrefix(relToSpace, ".."+string(filepath.Separator)) {
			continue
		}
		parentPath := filepath.Dir(child.RealPath)
		parentID := node.ID
		if filepath.Clean(parentPath) != filepath.Clean(spacePath) {
			parentRel, err := filepath.Rel(r.root, parentPath)
			if err != nil {
				return FileNode{}, err
			}
			parentID = stableID(filepath.ToSlash(parentRel))
		}
		childrenByParent[parentID] = append(childrenByParent[parentID], childID)
	}

	var hydrate func(id string) FileNode
	hydrate = func(id string) FileNode {
		current := cloneNode(nodes[id])
		current.Children = nil
		for _, childID := range childrenByParent[id] {
			current.Children = append(current.Children, hydrate(childID))
		}
		sortNodes(current.Children)
		nodes[id] = cloneNode(current)
		return current
	}
	return hydrate(node.ID), nil
}

func (r *RootRepository) nodeFromEntry(path string, d fs.DirEntry, dirName string, space string) (FileNode, error) {
	info, err := d.Info()
	if err != nil {
		return FileNode{}, err
	}
	rel, err := filepath.Rel(r.root, path)
	if err != nil {
		return FileNode{}, err
	}
	spaceRel, err := filepath.Rel(filepath.Join(r.root, dirName), path)
	if err != nil {
		return FileNode{}, err
	}
	content := ""
	if !d.IsDir() && info.Size() <= 256*1024 {
		if bytes, err := os.ReadFile(path); err == nil {
			content = string(bytes)
		}
	}
	name := displayName(d.Name(), content)
	return FileNode{
		ID:         stableID(filepath.ToSlash(rel)),
		Name:       name,
		Path:       "/" + strings.TrimPrefix(filepath.ToSlash(filepath.Join(space, spaceRel)), "/"),
		RealPath:   path,
		Type:       fileType(d.Name(), d.IsDir()),
		Space:      space,
		Size:       humanSize(info.Size(), d.IsDir()),
		SizeBytes:  info.Size(),
		Modified:   info.ModTime(),
		Tags:       inferTags(space, d.Name(), content),
		Permission: inferPermission(space, d.Name(), content),
		Summary:    inferSummary(content, name),
		IsDir:      d.IsDir(),
	}, nil
}

func (r *RootRepository) getLocked(id string) (FileNode, error) {
	node, ok := r.nodes[id]
	if !ok {
		return FileNode{}, fmt.Errorf("file node not found: %s", id)
	}
	return cloneNode(node), nil
}

func (r *RootRepository) resolveDestinationLocked(space string, pathValue string) (string, error) {
	cleanPath := strings.Trim(strings.TrimSpace(pathValue), "/")
	if cleanPath == "" && strings.TrimSpace(space) != "" {
		cleanPath = strings.Trim(strings.TrimSpace(space), "/")
	}
	if cleanPath == "" {
		return r.root, nil
	}
	for dir, display := range spaceNames {
		if cleanPath == display || strings.HasPrefix(cleanPath, display+"/") {
			cleanPath = dir + strings.TrimPrefix(cleanPath, display)
			break
		}
	}
	target, err := r.safeJoin(r.root, cleanPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("destination is not a directory: %s", pathValue)
	}
	return target, nil
}

func (r *RootRepository) safeJoin(base string, parts ...string) (string, error) {
	candidate := filepath.Clean(filepath.Join(append([]string{base}, parts...)...))
	rel, err := filepath.Rel(r.root, candidate)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path escapes NAS root: %s", candidate)
	}
	return candidate, nil
}

func (r *RootRepository) nodeByRealPathLocked(path string) (FileNode, error) {
	cleaned := filepath.Clean(path)
	for _, node := range r.nodes {
		if filepath.Clean(node.RealPath) == cleaned {
			return cloneNode(node), nil
		}
	}
	return FileNode{}, fmt.Errorf("file node not found for path: %s", path)
}

func cleanLeafName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = filepath.Base(name)
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	return name
}

func (r *RootRepository) replaceTreeNodeLocked(id string, updated FileNode) error {
	var replace func(nodes []FileNode) bool
	replace = func(nodes []FileNode) bool {
		for i := range nodes {
			if nodes[i].ID == id {
				nodes[i] = cloneNode(updated)
				return true
			}
			if replace(nodes[i].Children) {
				return true
			}
		}
		return false
	}
	if replace(r.tree.Children) {
		return nil
	}
	return fmt.Errorf("file node not found in tree: %s", id)
}
