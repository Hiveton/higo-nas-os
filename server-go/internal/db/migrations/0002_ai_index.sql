-- AI index backbone, decoupled from the org/ACL schema so it can coexist with
-- the JSON-state domain services. Content identity is the source URI (file path
-- or media id). Embeddings use a dimension-flexible `vector` column so any
-- OpenAI-compatible embedding model works without a schema change; search uses
-- exact KNN (no fixed-dim ANN index required).
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS ai_documents (
  id          BIGSERIAL PRIMARY KEY,
  source_uri  TEXT NOT NULL UNIQUE,
  domain      TEXT NOT NULL DEFAULT 'file',   -- file | photo | video | music
  space       TEXT NOT NULL DEFAULT '',
  title       TEXT NOT NULL DEFAULT '',
  summary     TEXT NOT NULL DEFAULT '',
  checksum    TEXT NOT NULL DEFAULT '',
  tags        TEXT[] NOT NULL DEFAULT '{}',
  metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,
  indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ai_documents_domain_idx ON ai_documents (domain);
CREATE INDEX IF NOT EXISTS ai_documents_space_idx ON ai_documents (space);

CREATE TABLE IF NOT EXISTS ai_chunks (
  id        BIGSERIAL PRIMARY KEY,
  doc_id    BIGINT NOT NULL REFERENCES ai_documents(id) ON DELETE CASCADE,
  chunk_no  INTEGER NOT NULL,
  content   TEXT NOT NULL,
  embedding vector,
  UNIQUE (doc_id, chunk_no)
);

CREATE INDEX IF NOT EXISTS ai_chunks_doc_idx ON ai_chunks (doc_id);
