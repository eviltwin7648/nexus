CREATE TABLE source_registry(
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    config JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW());

CREATE TABLE raw_documents(
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES source_registry(id),
    source_type TEXT NOT NULL,
    path TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    url TEXT NOT NULL,
    checksum TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending_enrichment',
    ingested_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE ingest_jobs (
    id BIGSERIAL PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES source_registry(id),
    doc_id TEXT,
    status TEXT NOT NULL,
    error TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX idx_raw_documents_source_id ON raw_documents(source_id);
CREATE INDEX idx_raw_documents_status ON raw_documents(status);
CREATE INDEX idx_raw_documents_checksum ON raw_documents(checksum);
CREATE INDEX idx_ingest_jobs_source_id ON ingest_jobs(source_id);
