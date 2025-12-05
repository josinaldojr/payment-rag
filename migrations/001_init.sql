-- Ativa extensão pgvector
CREATE EXTENSION IF NOT EXISTS vector;

-- Tabela principal dos pedaços de documentação
CREATE TABLE IF NOT EXISTS doc_chunk (
    id           BIGSERIAL PRIMARY KEY,
    provider     TEXT NOT NULL,        
    section_type TEXT,
    title        TEXT,
    content      TEXT NOT NULL,
    source_url   TEXT,
    api_version  TEXT,
    tags         TEXT[],
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tabela dos embeddings
-- VECTOR(768) alinhado ao modelo de embedding escolhido
CREATE TABLE IF NOT EXISTS doc_chunk_embedding (
    chunk_id   BIGINT PRIMARY KEY REFERENCES doc_chunk(id) ON DELETE CASCADE,
    embedding  VECTOR(768),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_doc_chunk_provider
    ON doc_chunk (provider);

CREATE INDEX IF NOT EXISTS idx_doc_chunk_tags_gin
    ON doc_chunk USING GIN (tags);

CREATE INDEX IF NOT EXISTS idx_doc_chunk_embedding_vector
    ON doc_chunk_embedding
    USING ivfflat (embedding vector_l2_ops)
    WITH (lists = 100);