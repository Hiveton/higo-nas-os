package search

import (
	"strings"
	"testing"
)

func TestFiltersEmpty(t *testing.T) {
	clause, args := filters(Request{})
	if clause != "" || len(args) != 0 {
		t.Fatalf("empty filters should be empty, got %q args=%v", clause, args)
	}
}

func TestFiltersDomainAndSpace(t *testing.T) {
	clause, args := filters(Request{Domains: []string{"file"}, Spaces: []string{"home", "team"}})
	if !strings.Contains(clause, "d.domain = ANY($1)") {
		t.Fatalf("expected domain filter, got %q", clause)
	}
	if !strings.Contains(clause, "d.space = ANY($2)") {
		t.Fatalf("expected space filter, got %q", clause)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
}

func TestEncodeVector(t *testing.T) {
	if got := encodeVector([]float32{1, 2.5}); got != "[1,2.5]" {
		t.Fatalf("encodeVector = %q", got)
	}
}

func TestSnippetTruncates(t *testing.T) {
	long := strings.Repeat("汉", 300)
	got := snippet(long)
	if !strings.HasSuffix(got, "…") || len([]rune(got)) != 241 {
		t.Fatalf("snippet length unexpected: %d", len([]rune(got)))
	}
}

func TestAnswerLine(t *testing.T) {
	if got := answerLine(nil); !strings.Contains(got, "没有") {
		t.Fatalf("empty answer line = %q", got)
	}
	got := answerLine([]Hit{{Title: "合同"}})
	if !strings.Contains(got, "合同") {
		t.Fatalf("answer line should name top hit, got %q", got)
	}
}

func TestDisabledWithoutDB(t *testing.T) {
	s := New(nil, nil)
	if s.Enabled() {
		t.Fatal("nil pool should be disabled")
	}
	if _, err := s.Semantic(nil, Request{Query: "x"}); err == nil {
		t.Fatal("expected error when DB not configured")
	}
}

func TestTokenize(t *testing.T) {
	terms := tokenize("用语义搜索找客户A的合同")
	hasHetong := false
	for _, x := range terms {
		if x == "合同" {
			hasHetong = true
		}
	}
	if !hasHetong {
		t.Fatalf("expected '合同' among terms, got %v", terms)
	}
}
