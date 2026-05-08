package files

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"higoos/server-go/internal/state"
)

var spaceNames = map[string]string{
	"home-space":       "家庭空间",
	"team-space":       "团队空间",
	"finance-receipts": "财务票据",
	"photos-and-media": "照片与视频",
	"backup-archive":   "备份归档",
	"downloads":        "下载目录",
}

var orderedSpaceDirs = []string{
	"home-space",
	"team-space",
	"finance-receipts",
	"photos-and-media",
	"backup-archive",
	"downloads",
}

type FixtureRepository struct {
	root      string
	nodes     map[string]FileNode
	tree      FileNode
	statePath string
}

func NewFixtureRepository(root string) (*FixtureRepository, error) {
	if strings.TrimSpace(root) == "" {
		var err error
		root, err = defaultFixtureRoot()
		if err != nil {
			return nil, err
		}
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("fixture root is not a directory: %s", abs)
	}

	repo := &FixtureRepository{
		root:  abs,
		nodes: map[string]FileNode{},
		tree: FileNode{
			ID:    stableID("root"),
			Name:  "HiGoNAS",
			Path:  "/",
			Type:  "文件夹",
			Size:  "-",
			IsDir: true,
		},
	}
	if err := repo.scan(); err != nil {
		return nil, err
	}
	return repo, nil
}

func NewFixtureRepositoryWithStateDir(root string, stateDir string) (*FixtureRepository, error) {
	repo, err := NewFixtureRepository(root)
	if err != nil {
		return nil, err
	}
	if stateDir == "" {
		return repo, nil
	}
	repo.statePath = filepath.Join(stateDir, "files.json")
	var persisted []FileNode
	if err := state.LoadJSON(repo.statePath, &persisted); err != nil {
		return nil, err
	}
	for _, node := range persisted {
		current, ok := repo.nodes[node.ID]
		if !ok {
			continue
		}
		current.Tags = append([]string(nil), node.Tags...)
		if node.Permission != "" {
			current.Permission = node.Permission
		}
		if node.Summary != "" {
			current.Summary = node.Summary
		}
		repo.nodes[current.ID] = cloneNode(current)
		if err := repo.replaceTreeNode(current.ID, current); err != nil {
			return nil, err
		}
	}
	return repo, nil
}

func (r *FixtureRepository) Tree(_ context.Context) (FileNode, error) {
	return cloneNode(r.tree), nil
}

func (r *FixtureRepository) List(_ context.Context) ([]FileNode, error) {
	nodes := make([]FileNode, 0, len(r.nodes))
	for _, node := range r.nodes {
		nodes = append(nodes, cloneNode(node))
	}
	sortNodes(nodes)
	return nodes, nil
}

func (r *FixtureRepository) Get(_ context.Context, id string) (FileNode, error) {
	node, ok := r.nodes[id]
	if !ok {
		return FileNode{}, fmt.Errorf("file node not found: %s", id)
	}
	return cloneNode(node), nil
}

func (r *FixtureRepository) Put(_ context.Context, node FileNode) error {
	if node.ID == "" {
		return errors.New("file node id is required")
	}
	r.nodes[node.ID] = cloneNode(node)
	if err := r.replaceTreeNode(node.ID, node); err != nil {
		return err
	}
	return r.saveState()
}

func (r *FixtureRepository) saveState() error {
	if r.statePath == "" {
		return nil
	}
	nodes := make([]FileNode, 0, len(r.nodes))
	for _, node := range r.nodes {
		nodes = append(nodes, cloneNode(node))
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	return state.SaveJSON(r.statePath, nodes)
}

func (r *FixtureRepository) scan() error {
	for _, dirName := range orderedSpaceDirs {
		spacePath := filepath.Join(r.root, dirName)
		info, err := os.Stat(spacePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		if !info.IsDir() {
			continue
		}
		spaceNode, err := r.scanSpace(dirName, info)
		if err != nil {
			return err
		}
		r.tree.Children = append(r.tree.Children, spaceNode)
	}

	entries, err := os.ReadDir(r.root)
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, dirName := range orderedSpaceDirs {
		known[dirName] = true
	}
	for _, entry := range entries {
		if !entry.IsDir() || known[entry.Name()] {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		spaceNode, err := r.scanSpace(entry.Name(), info)
		if err != nil {
			return err
		}
		r.tree.Children = append(r.tree.Children, spaceNode)
	}
	sortNodes(r.tree.Children)
	return nil
}

func (r *FixtureRepository) scanSpace(dirName string, info fs.FileInfo) (FileNode, error) {
	space := spaceNames[dirName]
	if space == "" {
		space = dirName
	}
	rel := filepath.ToSlash(dirName)
	node := FileNode{
		ID:         stableID(rel),
		Name:       space,
		Path:       "/" + space,
		RealPath:   filepath.Join(r.root, dirName),
		Type:       "文件夹",
		Space:      space,
		Size:       "-",
		Modified:   info.ModTime(),
		Permission: inferPermission(space, "", ""),
		IsDir:      true,
	}
	r.nodes[node.ID] = cloneNode(node)

	err := filepath.WalkDir(node.RealPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == node.RealPath {
			return nil
		}
		child, err := r.nodeFromEntry(path, d, dirName, space)
		if err != nil {
			return err
		}
		r.nodes[child.ID] = cloneNode(child)
		parentRel, err := filepath.Rel(node.RealPath, filepath.Dir(path))
		if err != nil {
			return err
		}
		if parentRel == "." {
			node.Children = append(node.Children, child)
			return nil
		}
		parentID := stableID(filepath.ToSlash(filepath.Join(dirName, parentRel)))
		parent := r.nodes[parentID]
		parent.Children = append(parent.Children, child)
		sortNodes(parent.Children)
		r.nodes[parentID] = parent
		return nil
	})
	if err != nil {
		return FileNode{}, err
	}
	sortNodes(node.Children)
	r.nodes[node.ID] = cloneNode(node)
	return node, nil
}

func (r *FixtureRepository) nodeFromEntry(path string, d fs.DirEntry, dirName string, space string) (FileNode, error) {
	info, err := d.Info()
	if err != nil {
		return FileNode{}, err
	}
	rel, err := filepath.Rel(r.root, path)
	if err != nil {
		return FileNode{}, err
	}
	relSlash := filepath.ToSlash(rel)
	spaceRel, err := filepath.Rel(filepath.Join(r.root, dirName), path)
	if err != nil {
		return FileNode{}, err
	}
	spaceRelSlash := filepath.ToSlash(spaceRel)
	content := ""
	if !d.IsDir() {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return FileNode{}, err
		}
		content = string(bytes)
	}
	name := displayName(d.Name(), content)
	node := FileNode{
		ID:         stableID(relSlash),
		Name:       name,
		Path:       "/" + strings.TrimPrefix(filepath.ToSlash(filepath.Join(space, spaceRelSlash)), "/"),
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
	}
	return node, nil
}

func (r *FixtureRepository) replaceTreeNode(id string, updated FileNode) error {
	if r.tree.ID == id {
		r.tree = cloneNode(updated)
		return nil
	}
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

func defaultFixtureRoot() (string, error) {
	candidates := []string{
		filepath.Join("fixtures", "nas-root"),
		filepath.Join("server-go", "fixtures", "nas-root"),
		filepath.Join("..", "..", "fixtures", "nas-root"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("fixture root not found")
}

func stableID(value string) string {
	sum := sha1.Sum([]byte(filepath.ToSlash(value)))
	return hex.EncodeToString(sum[:])[:16]
}

func displayName(filename string, content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	ext := filepath.Ext(filename)
	if ext == "" {
		return filename
	}
	return strings.TrimSuffix(filename, ext)
}

func fileType(filename string, isDir bool) string {
	if isDir {
		return "文件夹"
	}
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if ext == "" {
		return "FILE"
	}
	return strings.ToUpper(ext)
}

func humanSize(size int64, isDir bool) string {
	if isDir {
		return "-"
	}
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(size)
	for _, unit := range units {
		value = value / 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%.1f PB", value/1024)
}

func inferTags(space string, filename string, content string) []string {
	haystack := strings.ToLower(space + " " + filename + " " + content)
	var tags []string
	add := func(tag string) {
		if tag != "" && !stringSliceContains(tags, tag) {
			tags = append(tags, tag)
		}
	}
	if strings.Contains(haystack, "合同") || strings.Contains(haystack, "contract") {
		add("合同")
	}
	if strings.Contains(haystack, "客户 a") || strings.Contains(haystack, "客户a") {
		add("客户A")
	}
	if strings.Contains(haystack, "保密") || strings.Contains(haystack, "权限") {
		add("权限敏感")
	}
	if strings.Contains(haystack, "发票") || strings.Contains(haystack, "invoice") {
		add("发票")
	}
	if strings.Contains(haystack, "重复") || strings.Contains(haystack, "duplicate") {
		add("重复项")
	}
	if strings.Contains(haystack, "未归档") || strings.Contains(haystack, "unarchived") || strings.Contains(haystack, "下载") {
		add("待整理")
	}
	if strings.Contains(haystack, "保修") || strings.Contains(haystack, "保险") || strings.Contains(haystack, "warranty") {
		add("保修")
		add("家庭知识库")
	}
	if strings.Contains(haystack, "到期") || strings.Contains(haystack, "提醒") {
		add("需提醒")
	}
	if strings.Contains(haystack, "照片") || strings.Contains(haystack, "旅行") || strings.Contains(haystack, "回忆") {
		add("旅行")
		add("可生成回忆")
	}
	if strings.Contains(haystack, "备份") || strings.Contains(haystack, "backup") {
		add("备份")
	}
	return tags
}

func inferPermission(space string, filename string, content string) string {
	haystack := strings.ToLower(space + " " + filename + " " + content)
	switch {
	case strings.Contains(haystack, "仅管理员") || space == "财务票据":
		return "仅管理员"
	case strings.Contains(haystack, "项目组") || strings.Contains(haystack, "保密") || space == "团队空间":
		return "项目组"
	case space == "家庭空间" || space == "照片与视频":
		return "家人可见"
	default:
		return "管理员可见"
	}
}

func inferSummary(content string, fallback string) string {
	lines := strings.Split(content, "\n")
	var summary []string
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		summary = append(summary, line)
		if len(strings.Join(summary, " ")) >= 140 {
			break
		}
	}
	if len(summary) == 0 {
		return fallback
	}
	return truncate(strings.Join(summary, " "), 220)
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func cloneNode(node FileNode) FileNode {
	node.Tags = append([]string(nil), node.Tags...)
	node.Children = cloneNodes(node.Children)
	return node
}

func cloneNodes(nodes []FileNode) []FileNode {
	out := make([]FileNode, len(nodes))
	for i, node := range nodes {
		out[i] = cloneNode(node)
	}
	return out
}

func sortNodes(nodes []FileNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Space != nodes[j].Space {
			return spaceOrder(nodes[i].Space) < spaceOrder(nodes[j].Space)
		}
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return nodes[i].Name < nodes[j].Name
	})
}

func spaceOrder(space string) int {
	for i, dir := range orderedSpaceDirs {
		if spaceNames[dir] == space {
			return i
		}
	}
	return len(orderedSpaceDirs)
}

func formatModified(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
