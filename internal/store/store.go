package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eviltwin7648/nexus/internal/chunker"
	"github.com/eviltwin7648/nexus/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgx config: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}
func (s *Store) RegisterSource(ctx context.Context, id, sourceType string, config map[string]any) error {
	configJson, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal err %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO source_registry (id, type, config)
		VALUES($1,$2,$3)
		ON CONFLICT (id) DO UPDATE
			SET config = EXCLUDED.config
		`, id, sourceType, configJson)
	return err
}

func (s *Store) UpdateSourceSyncTime(ctx context.Context, sourceId string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE source_registry
		SET last_synced_at = NOW()
		WHERE id = $1
		`, sourceId)
	return err
}

func (s *Store) GetSourceLastSynced(ctx context.Context, sourceId string) (time.Time, error) {
	var t *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT last_synced_at FROM source_registry WHERE id = $1
		`, sourceId).Scan(&t)
	if err == pgx.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	if t == nil {
		return time.Time{}, nil
	}
	return *t, err
}

func (s *Store) ChecksumExists(ctx context.Context, docId, checksum string) (bool, error) {
	var existing string
	err := s.pool.QueryRow(ctx, `
		SELECT checksum FROM raw_documents WHERE id = $1
		`, docId).Scan(&existing)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return existing == checksum, nil
}
func (s *Store) UpsertDocument(ctx context.Context, doc domain.RawDocument) error {
	metaJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadaata: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
	INSERT INTO raw_documents(
	id, source_id, source_type, path, title, content, metadata, url, checksum, status, ingested_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending_enrichment',$10, $11)
	ON CONFLICT (id) DO UPDATE SET
		title= EXCLUDED.title,
		content    = EXCLUDED.content,
		metadata   = EXCLUDED.metadata,
		checksum   = EXCLUDED.checksum,
		status     = 'pending_enrichment',
		updated_at = EXCLUDED.updated_at
	`, doc.ID, doc.SourceId, string(doc.SourceType), doc.Path,
		doc.Title, doc.Content, metaJSON, doc.URL, doc.Checksum, time.Now(), doc.UpdatedAt)
	return err

}

func (s *Store) RecordJob(ctx context.Context, sourceId string, docId *string, status, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
	INSERT INTO ingest_jobs (source_id, doc_id, status, error, finished_at)
	VALUES ($1, $2, $3, NULLIF($4, ''),NOW())
	`, sourceId, docId, status, errMsg)
	return err
}

// func (s *Store) UpsertChunk(ctx context.Context, chunk chunker.Chunk, embedding []float32) error {
// 	metaJson, err := json.Marshal(chunk.Metadata)
// 	if err != nil {
// 		return fmt.Errorf("marshall metadata: %w", err)
// 	}
// 	// convert []float32 to pgvector string format: "[0.1,0.2,...]"
// 	vec := float32SliceToPgVector(embedding)
// 	_, err = s.pool.Exec(ctx, `
// 		INSERT INTO document_chunks(
// 		id, doc_id, source_id, source_type, chunk_index, content, embedding, metadata
// 		) VALUES ($1,$2, $3, $4, $5, $6, $7::vector, $8)
// 		ON CONFLICT (id) DO UPDATE SET
// 			content = EXCLUDED.content,
// 			embedding = EXCLUDED.embedding, b
// 			metadata = EXCLUDED.metadata
// 		`, chunk.ID, chunk.DocId, chunk.SourceId, string(chunk.SourceType), chunk.Index, chunk.Content, vec, metaJson)
// 	return err
// }

func (s *Store) UpsertChunks(ctx context.Context, chunks []chunker.Chunk, embedding [][]float32) error {
	if len(chunks) == 0 {
		return nil
	}

	ids := make([]string, len(chunks))
	docIds := make([]string, len(chunks))
	sourceIds := make([]string, len(chunks))
	sourceTypes := make([]string, len(chunks))
	indexes := make([]int32, len(chunks))
	contents := make([]string, len(chunks))
	vectors := make([]string, len(chunks)) // pgvector as string
	metas := make([]string, len(chunks))
	for i, c := range chunks {
		metaJSON, err := json.Marshal(c.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata chunk %d: %w", i, err)
		}

		ids[i] = c.ID
		docIds[i] = c.DocId
		sourceIds[i] = c.SourceId
		sourceTypes[i] = string(c.SourceType)
		indexes[i] = int32(c.Index)
		contents[i] = c.Content
		vectors[i] = float32SliceToPgVector(embedding[i])
		metas[i] = string(metaJSON)
	}
	_, err := s.pool.Exec(ctx, `
			INSERT INTO document_chunks (
				id, doc_id, source_id, source_type,
				chunk_index, content, embedding, metadata
			)
			SELECT * FROM unnest(
				$1::text[],
				$2::text[],
				$3::text[],
				$4::text[],
				$5::int[],
				$6::text[],
				$7::vector[],
				$8::jsonb[]
			)
			ON CONFLICT (id) DO UPDATE SET
				content   = EXCLUDED.content,
				embedding = EXCLUDED.embedding,
				metadata  = EXCLUDED.metadata
		`, ids, docIds, sourceIds, sourceTypes, indexes, contents, vectors, metas)

	return err

}

func (s *Store) MarkDocumentEnriched(ctx context.Context, docId string) error {
	_, err := s.pool.Exec(ctx, `
			UPDATE raw_documents SET status = 'enriched' WHERE id = $1
		`, docId)
	return err
}

func (s *Store) MarkDocumentFailed(ctx context.Context, docID string, reason string) error {
	_, err := s.pool.Exec(ctx, `
        UPDATE raw_documents
        SET status = 'enrichment_failed', metadata = metadata || jsonb_build_object('enrich_error', $2)
        WHERE id = $1
    `, docID, reason)
	return err
}

func (s *Store) GetPendingDocuments(ctx context.Context, limit int) ([]domain.RawDocument, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, source_id, source_type, path, title, content, metadata, url, checksum, updated_at
		FROM raw_documents
		WHERE status = 'pending_enrichment'
		ORDER BY updated_at ASC
		LIMIT $1
		`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []domain.RawDocument
	for rows.Next() {
		var doc domain.RawDocument
		var metaJSON []byte
		var sourceType string
		err := rows.Scan(
			&doc.ID, &doc.SourceId, &sourceType, &doc.Path, &doc.Title,
			&doc.Content, &metaJSON, &doc.URL, &doc.Checksum, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		doc.SourceType = domain.SourceType(sourceType)
		if err := json.Unmarshal(metaJSON, &doc.Metadata); err != nil {
			return nil, err
		}

		docs = append(docs, doc)
	}

	return docs, rows.Err()

}

func float32SliceToPgVector(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteString("[")
	for i, f := range v {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(strconv.FormatFloat(float64(f), 'f', 8, 32))
	}
	b.WriteString("]")
	return b.String()
}

type ChunkResult struct {
	Id         string
	DocId      string
	SourceId   string
	SourceType string
	Path       string
	ChunkIndex int
	Content    string
	Score      float64
}

func (s *Store) SearchChunks(ctx context.Context, queryVec []float32, topK int) ([]ChunkResult, error) {
	vec := float32SliceToPgVector(queryVec)
	rows, err := s.pool.Query(ctx, `
		SELECT
			c.id,
			c.doc_id,
			c.source_id,
			c.source_type,
			COALESCE(c.metadata->>'doc_path', c.id) AS path,
			c.chunk_index,
			c.content,
			1 - (c.embedding <=> $1::vector) AS score
		FROM document_chunks c
		ORDER BY c.embedding <=> $1::vector
		LIMIT $2
		`, vec, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ChunkResult
	for rows.Next() {
		var r ChunkResult
		if err := rows.Scan(&r.Id, &r.DocId, &r.SourceId, &r.SourceType,
			&r.Path, &r.ChunkIndex, &r.Content, &r.Score,
		); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) SearchChunksByType(ctx context.Context, queryVec []float32, sourceType string, topK int) ([]ChunkResult, error) {
	vec := float32SliceToPgVector(queryVec)
	rows, err := s.pool.Query(ctx, `
	SELECT 	c.id,
	c.doc_id,
	c.source_id,
	c.source_type,
	COALESCE(c.metadata->>'doc_path', c.id) AS path,
	c.chunk_index,
	c.content,
	1 - (c.embedding <=> $1::vector) AS score
	FROM document_chunks c
	WHERE source_type = $2
	ORDER BY c.embedding <=> $1::vector
	LIMIT $3
	`, vec, sourceType, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []ChunkResult
	for rows.Next() {
		var r ChunkResult
		if err := rows.Scan(&r.Id, &r.DocId, &r.SourceId, &r.SourceType,
			&r.Path, &r.ChunkIndex, &r.Content, &r.Score,
		); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *Store) GetDocumentByPath(ctx context.Context, path string) (*domain.RawDocument, error) {
	var doc domain.RawDocument
	var metaJSON []byte
	var sourceType string

	err := s.pool.QueryRow(ctx, `
		SELECT id, source_id, source_type, path, title,
		       content, metadata, url, checksum, updated_at
		FROM raw_documents
		WHERE path = $1
		LIMIT 1
	`, path).Scan(
		&doc.ID, &doc.SourceId, &sourceType, &doc.Path, &doc.Title,
		&doc.Content, &metaJSON, &doc.URL, &doc.Checksum, &doc.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("document not found: %s", path)
	}
	if err != nil {
		return nil, err
	}

	doc.SourceType = domain.SourceType(sourceType)
	if err := json.Unmarshal(metaJSON, &doc.Metadata); err != nil {
		return nil, err
	}

	return &doc, nil
}

func (s *Store) LexicalSearch(ctx context.Context, query string, topK int, sourceType *string) ([]ChunkResult, error) {
	rows, err := s.pool.Query(ctx, `
	SELECT id,doc_id,source_id,source_type,metadata->>'doc_path' as path, chunk_index, content, ts_rank(search_vector, plainto_tsquery('simple', $1)) as rank
	FROM document_chunks
	WHERE search_vector @@ plainto_tsquery('simple', $1)
	ORDER BY rank DESC
	LIMIT $2
	`, query, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []ChunkResult
	for rows.Next() {
		var r ChunkResult
		if err := rows.Scan(&r.Id, &r.DocId, &r.SourceId, &r.SourceType,
			&r.Path, &r.ChunkIndex, &r.Content, &r.Score); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
