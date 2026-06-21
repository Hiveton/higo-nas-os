package files

import (
	"context"
	"fmt"
	"strings"
)

// Viewer is the file-visibility scope of the requesting user. The HTTP layer
// builds it from the authenticated principal + accounts entitlements and passes
// it to TreeFor / CanAccess so each user sees and reads only their own personal
// folder, their group folders, and spaces explicitly granted to them. Admins see
// everything.
type Viewer struct {
	Admin         bool
	Username      string   // personal folder name under homes/
	GroupDirs     []string // group folder names under groups/
	GrantedSpaces []string // shared-space directory names the user may see
	DeniedSpaces  []string // spaces explicitly denied (hidden + read-blocked)
}

func (v Viewer) denied(dir string) bool {
	for _, d := range v.DeniedSpaces {
		if d == dir || displayForDir(d) == dir {
			return true
		}
	}
	return false
}

// TreeFor returns the file tree scoped to the viewer. Admins get the full tree
// (optionally filtered to one space); everyone else gets a synthetic root with
// only their personal folder, group folders, and granted shared spaces.
func (s *Service) TreeFor(ctx context.Context, space string, v Viewer) (FileNode, error) {
	if v.Admin {
		return s.Tree(ctx, space)
	}
	tree, err := s.repo.Tree(ctx)
	if err != nil {
		return FileNode{}, err
	}
	scoped := scopeTree(tree, v)
	space = strings.TrimSpace(space)
	if space == "" {
		return scoped, nil
	}
	for _, child := range scoped.Children {
		if child.Space == space || child.Name == space {
			return child, nil
		}
	}
	return FileNode{}, fmt.Errorf("space not found: %s", space)
}

// scopeTree rebuilds the top level of the tree to only the viewer's entitlements.
func scopeTree(tree FileNode, v Viewer) FileNode {
	out := FileNode{
		ID:    tree.ID,
		Name:  tree.Name,
		Path:  tree.Path,
		Type:  tree.Type,
		Size:  tree.Size,
		IsDir: true,
	}
	groups := map[string]bool{}
	for _, g := range v.GroupDirs {
		groups[g] = true
	}
	granted := map[string]bool{}
	for _, dir := range v.GrantedSpaces {
		granted[dir] = true
		granted[displayForDir(dir)] = true
	}
	for _, child := range tree.Children {
		switch child.Space {
		case "homes":
			for _, sub := range child.Children {
				if sub.Name == v.Username {
					personal := sub
					personal.Name = "我的文件"
					out.Children = append(out.Children, personal)
				}
			}
		case "groups":
			for _, sub := range child.Children {
				if groups[sub.Name] {
					out.Children = append(out.Children, sub)
				}
			}
		default:
			if granted[child.Space] && !v.denied(child.Space) {
				out.Children = append(out.Children, child)
			}
		}
	}
	return out
}

// CanAccess reports whether the viewer may read the given node, based on its
// logical path. Mirrors the on-disk ownership/ACL model for the web layer (which
// runs as root and would otherwise bypass filesystem permissions).
func (s *Service) CanAccess(node FileNode, v Viewer) bool {
	if v.Admin {
		return true
	}
	segs := strings.Split(strings.Trim(node.Path, "/"), "/")
	if len(segs) == 0 || segs[0] == "" {
		return false
	}
	switch segs[0] {
	case "homes":
		return len(segs) >= 2 && segs[1] == v.Username
	case "groups":
		if len(segs) < 2 {
			return false
		}
		for _, g := range v.GroupDirs {
			if g == segs[1] {
				return true
			}
		}
		return false
	default:
		if v.denied(segs[0]) {
			return false
		}
		dir := dirForDisplay(segs[0])
		for _, gd := range v.GrantedSpaces {
			if gd == dir || gd == segs[0] {
				return true
			}
		}
		return false
	}
}

// CanAccessID resolves a file by ID and reports whether the viewer may read it.
func (s *Service) CanAccessID(ctx context.Context, id string, v Viewer) bool {
	if v.Admin {
		return true
	}
	node, err := s.repo.Get(ctx, id)
	if err != nil {
		return false
	}
	return s.CanAccess(node, v)
}

// displayForDir maps a NAS-root directory to its display space name.
func displayForDir(dir string) string {
	if name, ok := spaceNames[dir]; ok {
		return name
	}
	return dir
}

// dirForDisplay maps a display space name back to its directory.
func dirForDisplay(display string) string {
	for dir, name := range spaceNames {
		if name == display {
			return dir
		}
	}
	return display
}
