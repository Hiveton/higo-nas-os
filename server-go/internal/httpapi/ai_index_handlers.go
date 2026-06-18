package httpapi

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	"higoos/server-go/internal/files"
	"higoos/server-go/internal/index"
	"higoos/server-go/internal/media"
	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/search"
)

const (
	maxIndexFileBytes = 2 << 20 // 2 MiB per file cap for indexing
	maxIndexFiles     = 2000    // safety cap per reindex run
)

// aiIndexDocument indexes a single document supplied in the request body.
func (a *API) aiIndexDocument(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.index == nil || !a.index.Enabled() {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "index_disabled", "AI 索引未启用（未配置数据库）")
		return
	}
	var body struct {
		SourceURI string         `json:"sourceUri"`
		Domain    string         `json:"domain"`
		Space     string         `json:"space"`
		Title     string         `json:"title"`
		Text      string         `json:"text"`
		Summary   string         `json:"summary"`
		Tags      []string       `json:"tags"`
		Metadata  map[string]any `json:"metadata"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	res, err := a.index.IndexDocument(r.Context(), index.DocInput{
		SourceURI: body.SourceURI, Domain: body.Domain, Space: body.Space,
		Title: body.Title, Text: body.Text, Summary: body.Summary,
		Tags: body.Tags, Metadata: body.Metadata,
	})
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "index_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, res)
}

// aiIndexFiles walks the file tree and indexes every readable text file. It is
// the bulk-indexing entry point for "AI 文件" semantic search.
func (a *API) aiIndexFiles(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.index == nil || !a.index.Enabled() {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "index_disabled", "AI 索引未启用（未配置数据库）")
		return
	}
	if a.files == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "files_unavailable", "files service is unavailable")
		return
	}
	var body struct {
		Space string `json:"space"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}

	root, err := a.files.Tree(r.Context(), body.Space)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "files_tree_failed", err.Error())
		return
	}

	// The root tree is shallow (spaces only); fetch each space's full subtree so
	// nested files are visible. Dedupe by path.
	seen := map[string]bool{}
	var nodes []files.FileNode
	add := func(n files.FileNode) {
		if !seen[n.Path] {
			seen[n.Path] = true
			nodes = append(nodes, n)
		}
	}
	collect := func(tree files.FileNode) {
		var found []files.FileNode
		collectTextFiles(tree, &found)
		for _, n := range found {
			add(n)
		}
	}
	collect(root)
	if body.Space == "" {
		for _, child := range root.Children {
			if !child.IsDir || child.Space == "" {
				continue
			}
			if sub, err := a.files.Tree(r.Context(), child.Space); err == nil {
				collect(sub)
			}
		}
	}

	indexed, skipped, failed := 0, 0, 0
	for _, node := range nodes {
		if indexed+failed >= maxIndexFiles {
			break
		}
		text, ok := a.readFileText(r.Context(), node)
		if !ok {
			skipped++
			continue
		}
		if _, err := a.index.IndexDocument(r.Context(), index.DocInput{
			SourceURI: "file:" + node.Path,
			Domain:    "file",
			Space:     node.Space,
			Title:     node.Name,
			Text:      text,
			Tags:      node.Tags,
			Metadata:  map[string]any{"path": node.Path, "type": node.Type},
		}); err != nil {
			failed++
			continue
		}
		indexed++
	}

	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"indexed": indexed, "skipped": skipped, "failed": failed, "candidates": len(nodes),
	})
}

// aiIndexMedia indexes photo/video metadata into the shared AI index so the
// library is searchable by natural language through /search/semantic. Vision
// captions can enrich this later; today it indexes title/album/people/place/
// device/timeline text, which already makes the library semantically findable.
func (a *API) aiIndexMedia(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.index == nil || !a.index.Enabled() {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "index_disabled", "AI 索引未启用（未配置数据库）")
		return
	}
	if a.media == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "media_unavailable", "media service is unavailable")
		return
	}
	items, err := a.media.Items(r.Context(), media.ItemFilter{})
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "media_items_failed", err.Error())
		return
	}
	indexed, failed := 0, 0
	for _, it := range items {
		domain := "photo"
		if it.Kind == "视频" {
			domain = "video"
		} else if it.Kind == "音乐" {
			domain = "music"
		}
		if _, err := a.index.IndexDocument(r.Context(), index.DocInput{
			SourceURI: "media:" + strconv.Itoa(it.ID),
			Domain:    domain,
			Space:     it.Album,
			Title:     it.Title,
			Text:      mediaText(it),
			Tags:      mediaTags(it),
			Metadata: map[string]any{
				"album": it.Album, "timeline": it.Timeline, "place": it.Place,
				"people": it.People, "device": it.Device, "kind": string(it.Kind),
			},
		}); err != nil {
			failed++
			continue
		}
		indexed++
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"indexed": indexed, "failed": failed, "candidates": len(items),
	})
}

// mediaText flattens a media item's metadata into indexable text, skipping the
// "待 AI 识别" placeholders.
func mediaText(it media.MediaItem) string {
	parts := []string{it.Title, it.Album, it.Timeline}
	for _, v := range []string{it.People, it.Place, it.Device} {
		if v != "" && !strings.Contains(v, "待 AI") {
			parts = append(parts, v)
		}
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}

func mediaTags(it media.MediaItem) []string {
	tags := []string{string(it.Kind)}
	if it.Album != "" {
		tags = append(tags, it.Album)
	}
	return tags
}

// aiIndexStatus reports index size and whether an embedding provider is bound.
func (a *API) aiIndexStatus(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	enabled := a.index != nil && a.index.Enabled()
	out := map[string]any{"enabled": enabled, "embeddingProvider": a.hasEmbeddingProvider()}
	if enabled {
		docs, chunks, embedded, err := a.index.Stats(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "index_status_failed", err.Error())
			return
		}
		out["documents"] = docs
		out["chunks"] = chunks
		out["embedded"] = embedded
	}
	platform.WriteJSON(w, r, http.StatusOK, out)
}

func (a *API) hasEmbeddingProvider() bool {
	if a.llm == nil {
		return false
	}
	_, err := a.llm.DefaultFor("embedding")
	return err == nil
}

// readFileText opens a file node and returns its extracted text (size-capped).
func (a *API) readFileText(ctx context.Context, node files.FileNode) (string, bool) {
	if node.IsDir || !index.IsTextPath(node.Path) {
		return "", false
	}
	if node.SizeBytes > maxIndexFileBytes {
		return "", false
	}
	rc, _, err := a.files.Open(ctx, node.ID)
	if err != nil {
		return "", false
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, maxIndexFileBytes))
	if err != nil {
		return "", false
	}
	return index.ExtractText(node.Path, data)
}

func collectTextFiles(node files.FileNode, out *[]files.FileNode) {
	if !node.IsDir && index.IsTextPath(node.Path) {
		*out = append(*out, node)
	}
	for _, child := range node.Children {
		collectTextFiles(child, out)
	}
}

// runSemanticSearch answers via the real vector index when available, mapping to
// the same response shape the frontend expects. Returns false if it could not
// run (caller falls back to the assistant's keyword search).
func (a *API) runSemanticSearch(w http.ResponseWriter, r *http.Request, query string, scopes []string, limit int) bool {
	if a.search == nil || !a.search.Enabled() {
		return false
	}
	resp, err := a.search.Semantic(r.Context(), search.Request{Query: query, Spaces: scopes, Limit: limit})
	if err != nil {
		return false
	}
	items := make([]map[string]any, 0, len(resp.Hits))
	citations := make([]map[string]any, 0, len(resp.Hits))
	for i, h := range resp.Hits {
		items = append(items, map[string]any{
			"id": h.SourceURI, "name": h.Title, "title": h.Title, "type": h.Domain,
			"space": h.Space, "size": "", "modified": "", "tags": []string{"semantic"},
			"permission": h.Space, "aiSummary": h.Summary, "path": h.SourceURI,
			"snippet": h.Snippet, "score": h.Score,
		})
		if i < 3 {
			citations = append(citations, map[string]any{
				"itemId": h.SourceURI, "title": h.Title, "path": h.SourceURI,
				"scope": h.Space, "quote": h.Snippet,
			})
		}
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"answer": resp.Answer, "items": items, "citations": citations, "vector": resp.Vector,
	})
	return true
}
