package index

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"higoos/server-go/internal/db"
	"higoos/server-go/internal/llm"
)

const embedBatch = 64

// Service indexes content into Postgres (pgvector) and is the write side of the
// shared AI backbone. It degrades gracefully: with no DB it is disabled; with no
// embedding provider it still stores documents/chunks (text search only).
type Service struct {
	pool      *db.Pool
	providers *llm.Store
}

// New builds an index service. pool may be nil (feature disabled).
func New(pool *db.Pool, providers *llm.Store) *Service {
	return &Service{pool: pool, providers: providers}
}

// Enabled reports whether a database backbone is available.
func (s *Service) Enabled() bool { return s != nil && s.pool != nil }

// DocInput describes one indexable item.
type DocInput struct {
	SourceURI string         // unique content id (file path or media id)
	Domain    string         // file | photo | video | music
	Space     string         // owning space/scope
	Title     string         // display title
	Text      string         // full extracted text (chunked + embedded)
	Summary   string         // optional precomputed summary
	Checksum  string         // content hash for change detection
	Tags      []string       // derived tags
	Metadata  map[string]any // arbitrary extra fields
}

// Result reports what an index operation produced.
type Result struct {
	DocID    int64 `json:"docId"`
	Chunks   int   `json:"chunks"`
	Embedded bool  `json:"embedded"`
}

// IndexDocument upserts a document by source URI and (re)builds its chunks and
// embeddings. Re-indexing the same URI replaces prior chunks atomically.
func (s *Service) IndexDocument(ctx context.Context, in DocInput) (Result, error) {
	if !s.Enabled() {
		return Result{}, fmt.Errorf("index: database not configured")
	}
	if strings.TrimSpace(in.SourceURI) == "" {
		return Result{}, fmt.Errorf("index: sourceUri is required")
	}
	if in.Domain == "" {
		in.Domain = "file"
	}
	chunks := ChunkText(in.Text, defaultChunkRunes, defaultChunkOverlap)
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		summary = deriveSummary(in.Title, in.Text)
	}

	// Embed chunks when an embedding provider is configured.
	var vectors [][]float32
	embedded := false
	if len(chunks) > 0 {
		if provider, err := s.providers.DefaultFor(llm.PurposeEmbedding); err == nil {
			vecs, err := embedAll(ctx, provider, chunks)
			if err != nil {
				return Result{}, fmt.Errorf("index: embed: %w", err)
			}
			vectors = vecs
			embedded = true
		}
	}

	metaJSON, err := json.Marshal(orEmpty(in.Metadata))
	if err != nil {
		return Result{}, fmt.Errorf("index: encode metadata: %w", err)
	}
	if in.Tags == nil {
		in.Tags = []string{}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback(ctx)

	var docID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO ai_documents (source_uri, domain, space, title, summary, checksum, tags, metadata, indexed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now())
		ON CONFLICT (source_uri) DO UPDATE SET
			domain=EXCLUDED.domain, space=EXCLUDED.space, title=EXCLUDED.title,
			summary=EXCLUDED.summary, checksum=EXCLUDED.checksum, tags=EXCLUDED.tags,
			metadata=EXCLUDED.metadata, indexed_at=now()
		RETURNING id`,
		in.SourceURI, in.Domain, in.Space, in.Title, summary, in.Checksum, in.Tags, metaJSON,
	).Scan(&docID)
	if err != nil {
		return Result{}, fmt.Errorf("index: upsert document: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM ai_chunks WHERE doc_id = $1`, docID); err != nil {
		return Result{}, fmt.Errorf("index: clear chunks: %w", err)
	}

	for i, chunk := range chunks {
		if embedded && i < len(vectors) && len(vectors[i]) > 0 {
			if _, err := tx.Exec(ctx,
				`INSERT INTO ai_chunks (doc_id, chunk_no, content, embedding) VALUES ($1,$2,$3,$4::vector)`,
				docID, i, chunk, encodeVector(vectors[i]),
			); err != nil {
				return Result{}, fmt.Errorf("index: insert chunk: %w", err)
			}
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO ai_chunks (doc_id, chunk_no, content) VALUES ($1,$2,$3)`,
			docID, i, chunk,
		); err != nil {
			return Result{}, fmt.Errorf("index: insert chunk: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	return Result{DocID: docID, Chunks: len(chunks), Embedded: embedded}, nil
}

// DeleteDocument removes a document and its chunks by source URI.
func (s *Service) DeleteDocument(ctx context.Context, sourceURI string) error {
	if !s.Enabled() {
		return fmt.Errorf("index: database not configured")
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM ai_documents WHERE source_uri = $1`, sourceURI)
	return err
}

// Stats reports index size for status surfaces.
func (s *Service) Stats(ctx context.Context) (docs, chunks, embedded int, err error) {
	if !s.Enabled() {
		return 0, 0, 0, fmt.Errorf("index: database not configured")
	}
	err = s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM ai_documents),
			(SELECT count(*) FROM ai_chunks),
			(SELECT count(*) FROM ai_chunks WHERE embedding IS NOT NULL)`).Scan(&docs, &chunks, &embedded)
	return docs, chunks, embedded, err
}

// embedAll embeds chunks in batches and returns one vector per chunk.
func embedAll(ctx context.Context, provider llm.Provider, chunks []string) ([][]float32, error) {
	out := make([][]float32, 0, len(chunks))
	for start := 0; start < len(chunks); start += embedBatch {
		end := start + embedBatch
		if end > len(chunks) {
			end = len(chunks)
		}
		vecs, err := llm.Embed(ctx, provider, chunks[start:end])
		if err != nil {
			return nil, err
		}
		out = append(out, vecs...)
	}
	return out, nil
}

func deriveSummary(title, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return strings.TrimSpace(title)
	}
	runes := []rune(text)
	if len(runes) > 200 {
		return strings.TrimSpace(string(runes[:200])) + "…"
	}
	return text
}

func orEmpty(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}
