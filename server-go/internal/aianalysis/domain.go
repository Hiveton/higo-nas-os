package aianalysis

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"higoos/server-go/internal/files"
	"higoos/server-go/internal/media"
	"higoos/server-go/internal/video"
)

// DomainSource abstracts enumerating a domain's items and writing analysis back.
// The engine depends on this interface; the concrete adapters depend on the
// domain services. The domain services never depend on aianalysis (one-way),
// which is why Apply translates AnalyzerResult into plain method arguments.
type DomainSource interface {
	Enumerate(ctx context.Context) ([]ItemRef, error)
	Apply(ctx context.Context, key string, res AnalyzerResult) error
}

// --- media (photos) ----------------------------------------------------------

type mediaSource struct{ svc *media.Service }

func (m mediaSource) Enumerate(ctx context.Context) ([]ItemRef, error) {
	items, err := m.svc.Items(ctx, media.ItemFilter{})
	if err != nil {
		return nil, err
	}
	out := make([]ItemRef, 0, len(items))
	for _, it := range items {
		if it.Kind == media.MediaKindMusic {
			continue
		}
		out = append(out, ItemRef{
			Key:        fmt.Sprintf("media:%d", it.ID),
			Domain:     DomainMedia,
			Title:      it.Title,
			SourcePath: it.SourcePath,
			Kind:       string(it.Kind),
			Sig:        statSig(it.SourcePath),
		})
	}
	return out, nil
}

func (m mediaSource) Apply(ctx context.Context, key string, res AnalyzerResult) error {
	id, err := strconv.Atoi(strings.TrimPrefix(key, "media:"))
	if err != nil {
		return fmt.Errorf("aianalysis: bad media key %q: %w", key, err)
	}
	return m.svc.ApplyAnalysis(id, res.People, res.Place, res.Device, summaryOrCaption(res))
}

// --- files -------------------------------------------------------------------

type fileSource struct{ svc *files.Service }

func (f fileSource) Enumerate(ctx context.Context) ([]ItemRef, error) {
	root, err := f.svc.Tree(ctx, "")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var refs []ItemRef
	collect := func(node files.FileNode) {
		var found []files.FileNode
		collectFileNodes(node, &found)
		for _, n := range found {
			if seen[n.ID] {
				continue
			}
			seen[n.ID] = true
			refs = append(refs, ItemRef{
				Key:        "file:" + n.ID,
				Domain:     DomainFile,
				Title:      n.Name,
				SourcePath: n.RealPath,
				Kind:       n.Type,
				Sig:        fmt.Sprintf("%d:%d", n.Modified.UnixNano(), n.SizeBytes),
			})
		}
	}
	collect(root)
	// The root tree is shallow (spaces only); descend each space for nested files.
	for _, child := range root.Children {
		if !child.IsDir || child.Space == "" {
			continue
		}
		if sub, err := f.svc.Tree(ctx, child.Space); err == nil {
			collect(sub)
		}
	}
	return refs, nil
}

func (f fileSource) Apply(ctx context.Context, key string, res AnalyzerResult) error {
	id := strings.TrimPrefix(key, "file:")
	_, err := f.svc.ApplyAnalysis(ctx, id, summaryOrCaption(res), res.Tags)
	return err
}

func collectFileNodes(node files.FileNode, out *[]files.FileNode) {
	if !node.IsDir {
		*out = append(*out, node)
	}
	for _, child := range node.Children {
		collectFileNodes(child, out)
	}
}

// --- video -------------------------------------------------------------------

type videoSource struct{ svc *video.Service }

func (v videoSource) Enumerate(ctx context.Context) ([]ItemRef, error) {
	items, err := v.svc.Items(ctx, "", "", "")
	if err != nil {
		return nil, err
	}
	out := make([]ItemRef, 0, len(items))
	for _, it := range items {
		out = append(out, ItemRef{
			Key:        "video:" + it.ID,
			Domain:     DomainVideo,
			Title:      it.Title,
			SourcePath: it.Path,
			Kind:       it.Kind,
			Sig:        fmt.Sprintf("%s:%d", it.ModifiedAt, it.SizeBytes),
		})
	}
	return out, nil
}

func (v videoSource) Apply(ctx context.Context, key string, res AnalyzerResult) error {
	id := strings.TrimPrefix(key, "video:")
	var container, codec, resolution string
	var duration int
	if res.TechMeta != nil {
		container = toStr(res.TechMeta["container"])
		codec = toStr(res.TechMeta["codec"])
		resolution = toStr(res.TechMeta["resolution"])
		duration = toInt(res.TechMeta["durationSeconds"])
	}
	return v.svc.ApplyAnalysis(id, summaryOrCaption(res), res.Tags, container, codec, resolution, duration)
}

// --- shared helpers ----------------------------------------------------------

func summaryOrCaption(res AnalyzerResult) string {
	if strings.TrimSpace(res.Summary) != "" {
		return res.Summary
	}
	return strings.TrimSpace(res.Caption)
}

// statSig builds a cheap content signature (mtime+size) for a path on disk, or
// "" when it cannot be stat'd (e.g. a fixture item with no backing file).
func statSig(path string) string {
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), info.Size())
}
