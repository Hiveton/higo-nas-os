package index

import "strings"

const (
	defaultChunkRunes   = 1200 // ~ a few hundred tokens per chunk
	defaultChunkOverlap = 150  // carried-over context between chunks
	maxChunksPerDoc     = 200  // cap work/cost for very large files
)

// ChunkText splits text into overlapping rune windows. It prefers to break on a
// paragraph/sentence boundary near the window end so chunks stay coherent.
func ChunkText(text string, size, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if size <= 0 {
		size = defaultChunkRunes
	}
	if overlap < 0 || overlap >= size {
		overlap = size / 8
	}
	runes := []rune(text)
	if len(runes) <= size {
		return []string{text}
	}

	var chunks []string
	start := 0
	for start < len(runes) && len(chunks) < maxChunksPerDoc {
		end := start + size
		if end >= len(runes) {
			chunks = append(chunks, strings.TrimSpace(string(runes[start:])))
			break
		}
		// Try to end on a natural boundary within the last 20% of the window.
		cut := boundary(runes, start+(size*4/5), end)
		chunk := strings.TrimSpace(string(runes[start:cut]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		next := cut - overlap
		if next <= start {
			next = cut
		}
		start = next
	}
	return chunks
}

// boundary returns the index of a good break point in (lo, hi], preferring the
// last newline, then sentence punctuation, else hi.
func boundary(runes []rune, lo, hi int) int {
	if lo < 0 {
		lo = 0
	}
	for i := hi - 1; i > lo; i-- {
		if runes[i] == '\n' {
			return i + 1
		}
	}
	for i := hi - 1; i > lo; i-- {
		switch runes[i] {
		case '.', '。', '!', '！', '?', '？', ';', '；':
			return i + 1
		}
	}
	return hi
}
