// Package search is the read side of the AI backbone: it answers semantic
// queries over the pgvector index built by internal/index. When an embedding
// provider is configured it does vector KNN; otherwise it falls back to keyword
// (ILIKE) matching so the feature still works degraded.
package search

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"higoos/server-go/internal/db"
	"higoos/server-go/internal/llm"
)

// Service runs semantic/keyword queries against the AI index.
type Service struct {
	pool      *db.Pool
	providers *llm.Store
}

// New builds a search service. pool may be nil (feature disabled).
func New(pool *db.Pool, providers *llm.Store) *Service {
	return &Service{pool: pool, providers: providers}
}

// Enabled reports whether the database backbone is available.
func (s *Service) Enabled() bool { return s != nil && s.pool != nil }

// Request is a semantic query. Domains/Spaces filter results when non-empty.
type Request struct {
	Query   string
	Domains []string
	Spaces  []string
	Limit   int
}

// Hit is one ranked result.
type Hit struct {
	SourceURI string  `json:"sourceUri"`
	Domain    string  `json:"domain"`
	Space     string  `json:"space"`
	Title     string  `json:"title"`
	Summary   string  `json:"summary"`
	Snippet   string  `json:"snippet"`
	Score     float64 `json:"score"`
	Vector    bool    `json:"vector"`
}

// Response bundles hits with a natural-language answer line.
type Response struct {
	Answer string `json:"answer"`
	Vector bool   `json:"vector"`
	Hits   []Hit  `json:"hits"`
}

// Semantic answers a query, preferring vector KNN and falling back to keyword
// matching when no embedding provider or no embeddings are available.
func (s *Service) Semantic(ctx context.Context, req Request) (Response, error) {
	if !s.Enabled() {
		return Response{}, fmt.Errorf("search: database not configured")
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return Response{}, fmt.Errorf("search: query is required")
	}
	limit := req.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// Try vector search when an embedding provider is configured.
	if provider, err := s.providers.DefaultFor(llm.PurposeEmbedding); err == nil {
		vecs, err := llm.Embed(ctx, provider, []string{query})
		if err == nil && len(vecs) == 1 && len(vecs[0]) > 0 {
			hits, err := s.vectorSearch(ctx, vecs[0], req, limit)
			if err != nil {
				return Response{}, err
			}
			if len(hits) > 0 {
				return Response{Answer: answerLine(hits), Vector: true, Hits: hits}, nil
			}
		}
	}

	hits, err := s.keywordSearch(ctx, query, req, limit)
	if err != nil {
		return Response{}, err
	}
	return Response{Answer: answerLine(hits), Vector: false, Hits: hits}, nil
}

func (s *Service) vectorSearch(ctx context.Context, vec []float32, req Request, limit int) ([]Hit, error) {
	where, args := filters(req)
	args = append(args, encodeVector(vec))
	vecArg := "$" + strconv.Itoa(len(args))
	args = append(args, limit)
	limArg := "$" + strconv.Itoa(len(args))

	q := `
		SELECT d.source_uri, d.domain, d.space, d.title, d.summary, c.content,
		       1 - (c.embedding <=> ` + vecArg + `::vector) AS score
		FROM ai_chunks c
		JOIN ai_documents d ON d.id = c.doc_id
		WHERE c.embedding IS NOT NULL ` + where + `
		ORDER BY c.embedding <=> ` + vecArg + `::vector ASC
		LIMIT ` + limArg
	return s.scanHits(ctx, q, args, true)
}

func (s *Service) keywordSearch(ctx context.Context, query string, req Request, limit int) ([]Hit, error) {
	where, args := filters(req)
	args = append(args, "%"+query+"%")
	likeArg := "$" + strconv.Itoa(len(args))
	args = append(args, limit)
	limArg := "$" + strconv.Itoa(len(args))

	q := `
		SELECT d.source_uri, d.domain, d.space, d.title, d.summary, c.content, 0.0 AS score
		FROM ai_chunks c
		JOIN ai_documents d ON d.id = c.doc_id
		WHERE (c.content ILIKE ` + likeArg + ` OR d.title ILIKE ` + likeArg + ` OR d.summary ILIKE ` + likeArg + `) ` + where + `
		LIMIT ` + limArg
	return s.scanHits(ctx, q, args, false)
}

func (s *Service) scanHits(ctx context.Context, q string, args []any, vector bool) ([]Hit, error) {
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search: query: %w", err)
	}
	defer rows.Close()

	var hits []Hit
	for rows.Next() {
		var h Hit
		var content string
		if err := rows.Scan(&h.SourceURI, &h.Domain, &h.Space, &h.Title, &h.Summary, &content, &h.Score); err != nil {
			return nil, fmt.Errorf("search: scan: %w", err)
		}
		h.Snippet = snippet(content)
		h.Vector = vector
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// filters builds an AND-clause for domain/space filters and returns the args.
func filters(req Request) (string, []any) {
	var clauses []string
	var args []any
	if len(req.Domains) > 0 {
		args = append(args, req.Domains)
		clauses = append(clauses, "d.domain = ANY($"+strconv.Itoa(len(args))+")")
	}
	if len(req.Spaces) > 0 {
		args = append(args, req.Spaces)
		clauses = append(clauses, "d.space = ANY($"+strconv.Itoa(len(args))+")")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " AND " + strings.Join(clauses, " AND "), args
}

func snippet(content string) string {
	content = strings.TrimSpace(content)
	runes := []rune(content)
	if len(runes) > 240 {
		return strings.TrimSpace(string(runes[:240])) + "…"
	}
	return content
}

func answerLine(hits []Hit) string {
	if len(hits) == 0 {
		return "没有找到相关内容。"
	}
	return fmt.Sprintf("找到 %d 条相关内容，最相关的是「%s」。", len(hits), hits[0].Title)
}

// encodeVector renders a float slice as a pgvector text literal.
func encodeVector(vec []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
