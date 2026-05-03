CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE document_chunks(
    id TEXT PRIMARY KEY,
    doc_id TEXT NOT NULL REFERENCES raw_documents(id) ON DELETE CASCADE,
    source_id TEXT NOT NULL,
    source_type TEXT NOT NULL,
    chunk_index INT NOT NULL,
    content INT NOT NULL,
    embedding vector(1536),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chunks_embedding ON document_chunks
    USING hnsw(embedding vector_cosine_ops);

CREATE INDEX idx_chunk_doc_id ON document_chunks(id);
CREATE INDEX idx_chunk_source_id ON document_chunks(source_id);
