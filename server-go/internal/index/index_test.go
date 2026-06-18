package index

import (
	"strings"
	"testing"
)

func TestChunkTextShortReturnsSingle(t *testing.T) {
	got := ChunkText("hello world", 1200, 150)
	if len(got) != 1 || got[0] != "hello world" {
		t.Fatalf("expected single chunk, got %#v", got)
	}
}

func TestChunkTextEmpty(t *testing.T) {
	if got := ChunkText("   ", 100, 10); got != nil {
		t.Fatalf("expected nil for blank, got %#v", got)
	}
}

func TestChunkTextOverlapAndCoverage(t *testing.T) {
	// 50 numbered lines; small window forces multiple chunks.
	var b strings.Builder
	for i := 0; i < 50; i++ {
		b.WriteString("line ")
		b.WriteByte(byte('A' + i%26))
		b.WriteByte('\n')
	}
	text := b.String()
	chunks := ChunkText(text, 80, 20)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	// Every chunk non-empty and within a reasonable bound of the window size.
	for i, c := range chunks {
		if strings.TrimSpace(c) == "" {
			t.Fatalf("chunk %d empty", i)
		}
		if len([]rune(c)) > 80+20 {
			t.Fatalf("chunk %d too large: %d runes", i, len([]rune(c)))
		}
	}
}

func TestChunkTextBreaksOnBoundary(t *testing.T) {
	text := strings.Repeat("a", 60) + ". " + strings.Repeat("b", 60)
	chunks := ChunkText(text, 64, 8)
	if len(chunks) < 2 {
		t.Fatalf("expected split, got %d", len(chunks))
	}
	// First chunk should end at the sentence boundary (the period).
	if !strings.HasSuffix(strings.TrimSpace(chunks[0]), ".") {
		t.Fatalf("expected first chunk to end on boundary, got %q", chunks[0])
	}
}

func TestIsTextPath(t *testing.T) {
	cases := map[string]bool{
		"/a/b/notes.md":  true,
		"report.TXT":     true,
		"data.csv":       true,
		"photo.jpg":      false,
		"movie.mp4":      false,
		"archive.tar.gz": false,
		"noext":          false,
	}
	for path, want := range cases {
		if got := IsTextPath(path); got != want {
			t.Errorf("IsTextPath(%q)=%v want %v", path, got, want)
		}
	}
}

func TestExtractText(t *testing.T) {
	if _, ok := ExtractText("a.bin", []byte("data")); ok {
		t.Fatal("binary path should not extract")
	}
	text, ok := ExtractText("a.md", []byte("  # Title\n\nbody  "))
	if !ok || !strings.Contains(text, "Title") {
		t.Fatalf("expected extracted text, got %q ok=%v", text, ok)
	}
}

func TestEncodeVector(t *testing.T) {
	if got := encodeVector([]float32{0.5, -1, 2}); got != "[0.5,-1,2]" {
		t.Fatalf("encodeVector = %q", got)
	}
	if got := encodeVector(nil); got != "" {
		t.Fatalf("empty vector should encode empty, got %q", got)
	}
}

func TestDeriveSummary(t *testing.T) {
	if got := deriveSummary("T", ""); got != "T" {
		t.Fatalf("blank text should fall back to title, got %q", got)
	}
	long := strings.Repeat("x", 300)
	if got := deriveSummary("T", long); !strings.HasSuffix(got, "…") {
		t.Fatalf("long text should be truncated with ellipsis, got len %d", len([]rune(got)))
	}
}

func TestIndexDisabledWithoutDB(t *testing.T) {
	s := New(nil, nil)
	if s.Enabled() {
		t.Fatal("nil pool should be disabled")
	}
	if _, err := s.IndexDocument(nil, DocInput{SourceURI: "x", Text: "y"}); err == nil {
		t.Fatal("expected error when DB not configured")
	}
}
