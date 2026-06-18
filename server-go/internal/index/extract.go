// Package index builds the AI content index: it extracts text from content
// sources, chunks it, embeds the chunks via the configured embedding provider,
// and persists documents/chunks/embeddings to Postgres (pgvector). The same
// index powers semantic search across files, photos, audio and video.
package index

import (
	"path/filepath"
	"strings"
)

// textExtensions are file extensions whose bytes are indexable as plain text.
var textExtensions = map[string]bool{
	".txt": true, ".md": true, ".markdown": true, ".rst": true,
	".log": true, ".csv": true, ".tsv": true, ".json": true, ".yaml": true,
	".yml": true, ".toml": true, ".ini": true, ".conf": true,
	".go": true, ".py": true, ".js": true, ".ts": true, ".tsx": true,
	".vue": true, ".java": true, ".c": true, ".h": true, ".cpp": true,
	".rs": true, ".rb": true, ".php": true, ".sh": true, ".sql": true,
	".html": true, ".css": true, ".xml": true,
}

// IsTextPath reports whether the path looks like an indexable text file.
func IsTextPath(path string) bool {
	return textExtensions[strings.ToLower(filepath.Ext(path))]
}

// ExtractText returns indexable UTF-8 text from raw bytes for a text file, plus
// whether extraction succeeded. Binary/unsupported types return ok=false; richer
// extractors (PDF/docx/vision/ASR) are layered on by domain callers.
func ExtractText(path string, data []byte) (string, bool) {
	if !IsTextPath(path) {
		return "", false
	}
	text := strings.TrimSpace(strings.ToValidUTF8(string(data), ""))
	if text == "" {
		return "", false
	}
	return text, true
}
