ALTER TABLE document_chunks add column search_vector tsvector GENERATED ALWAYS AS (
to_tsvector('simple', content)
) STORED;

CREATE index idx_document_search ON document_chunks USING GIN(search_vector);
