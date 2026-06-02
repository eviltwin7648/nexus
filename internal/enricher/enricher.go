package enricher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/eviltwin7648/nexus/internal/chunker"
	"github.com/eviltwin7648/nexus/internal/domain"
	"github.com/eviltwin7648/nexus/internal/embedder"
	"github.com/eviltwin7648/nexus/internal/parser"
	"github.com/eviltwin7648/nexus/internal/store"
)

const (
	batchSize     = 50 //chunks per openAI embedding req
	pollBatchSize = 20 //docs to pull from db per enrichment cycle
)

// take pending docs, chunk them, embed them and store back in db
type Enricher struct {
	store    *store.Store
	chunker  *chunker.Chunker
	embedder *embedder.Embedder
	log      *slog.Logger
}

func New(st *store.Store, emb *embedder.Embedder, log *slog.Logger, registry *parser.Registry) *Enricher {
	return &Enricher{
		store:    st,
		chunker:  chunker.New(registry),
		embedder: emb,
		log:      log,
	}
}

func (e *Enricher) Run(ctx context.Context) {
	e.log.Info("enricher started")
	for {
		select {
		case <-ctx.Done():
			e.log.Info("enricher stopped")
			return
		default:
			proccessed, err := e.ProcessBatch(ctx)
			if err != nil {
				e.log.Error("enrichment batch failed", "error", err)
				sleep(ctx, 10*time.Second) // back-off on err
			}
			if proccessed == 0 {
				sleep(ctx, 15*time.Second) // sleep if nothing pending
			}
		}
	}
}

func (e *Enricher) ProcessBatch(ctx context.Context) (int, error) {
	docs, err := e.store.GetPendingDocuments(ctx, pollBatchSize)
	if err != nil {
		return 0, fmt.Errorf("get pending documents: %w", err)
	}
	if len(docs) == 0 {
		return 0, nil
	}
	e.log.Info("enriching batch", "count", len(docs))

	for _, doc := range docs {
		if err := e.enrichOne(ctx, doc); err != nil {
			e.log.Error("failed to enrich document",
				"id", doc.ID,
				"path", doc.Path,
				"error", err,
			)
			continue
		}
	}
	return len(docs), nil
}

func (e *Enricher) enrichOne(ctx context.Context, doc domain.RawDocument) error {
	start := time.Now()
	if strings.TrimSpace(doc.Content) == "" {
		e.log.Warn("skipping empty document", "path", doc.Path)
		return e.store.MarkDocumentEnriched(ctx, doc.ID)
	}
	//chunk
	chunks, err := e.chunker.Chunk(doc)
	if err != nil {
		_ = e.store.MarkDocumentFailed(ctx, doc.ID, err.Error())
		return fmt.Errorf("chunk document: %w", err)
	}
	if len(chunks) == 0 {
		e.log.Warn("document produced no chunks", "id", doc.ID, "path", doc.Path)
		return e.store.MarkDocumentEnriched(ctx, doc.ID)
	}

	//embed
	allEmbeddings, err := e.embedChunks(ctx, chunks)
	if err != nil {
		_ = e.store.MarkDocumentFailed(ctx, doc.ID, err.Error())
		return fmt.Errorf("embed chunks: %w", err)
	}

	//store chunk with vector
	// filter chunks that got no embedding (were empty)
	validChunks := chunks[:0]
	validEmbeddings := allEmbeddings[:0]
	for i, emb := range allEmbeddings {
		if emb != nil {
			validChunks = append(validChunks, chunks[i])
			validEmbeddings = append(validEmbeddings, emb)
		}
	}

	if err := e.store.UpsertChunks(ctx, validChunks, validEmbeddings); err != nil {
		return fmt.Errorf("upsert chunks: %w", err)
	}

	//mark doc as done
	if err := e.store.MarkDocumentEnriched(ctx, doc.ID); err != nil {
		return fmt.Errorf("mark enriched: %w", err)
	}

	e.log.Info("enriched document",
		"id", doc.ID,
		"path", doc.Path,
		"chunks", len(chunks),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

func (e *Enricher) embedChunks(ctx context.Context, chunks []chunker.Chunk) ([][]float32, error) {
	allEmbeddings := make([][]float32, len(chunks))
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[i:end]
		texts := make([]string, len(batch))
		for j, c := range batch {
			texts[j] = c.Content
		}

		vectors, err := e.embedder.EmbedBatch(ctx, texts)
		if err != nil {
			return nil, fmt.Errorf("embed batch %d-%d: %w", i, end, err)
		}
		for j, vec := range vectors.Vectors {
			if vec != nil {
				allEmbeddings[i+j] = vec
			}
		}
	}
	return allEmbeddings, nil
}
func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
