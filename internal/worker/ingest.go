package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/eviltwin7648/nexus/internal/domain"
	"github.com/eviltwin7648/nexus/internal/store"
)

type Ingester struct {
	store *store.Store
	log   *slog.Logger
}

func NewIngester(store *store.Store, log *slog.Logger) *Ingester {
	return &Ingester{store: store, log: log}
}

func (w *Ingester) Ingest(ctx context.Context, doc domain.RawDocument) error {
	//for logs and permormance typeshit
	start := time.Now()
	same, err := w.store.ChecksumExists(ctx, doc.ID, doc.Checksum)
	if err != nil {
		w.recordFailure(ctx, doc, fmt.Errorf("checksum checl: %w", err))
		return err
	}
	if same {
		w.log.Debug("skipped unchanged document",
			"id", doc.ID,
			"path", doc.Path,
			"source", doc.SourceId,
		)
		return w.store.RecordJob(ctx, doc.SourceId, &doc.ID, "skipped", "")
	}
	if err := w.store.UpsertDocument(ctx, doc); err != nil {
		w.recordFailure(ctx, doc, fmt.Errorf("upsert: %w", err))
		return err
	}
	w.log.Info("ingested document",
		"id", doc.ID,
		"path", doc.Path,
		"source", doc.SourceId,
		"source_type", doc.SourceType,
		"duration_ms", time.Since(start).Milliseconds())
	return w.store.RecordJob(ctx, doc.SourceId, &doc.ID, "success", "")
}

type BatchResult struct {
	Total     int
	Succeeded int
	Failed    int
}

func (w *Ingester) Insgestbatch(ctx context.Context, docs []domain.RawDocument) BatchResult {
	result := BatchResult{Total: len(docs)}
	for _, doc := range docs {
		if err := w.Ingest(ctx, doc); err != nil {
			result.Failed++
			w.log.Error("failed to ingest document",
				"id", doc.ID,
				"path", doc.Path,
				"error", err,
			)
			continue
		}
		result.Succeeded++
	}
	return result
}

func (r BatchResult) String() string {
	skipped := r.Total - r.Succeeded - r.Failed
	return fmt.Sprintf("total=%d succeeded=%d skipped=%d failed=%d",
		r.Total, r.Succeeded, skipped, r.Failed,
	)
}

func (w *Ingester) recordFailure(ctx context.Context, doc domain.RawDocument, err error) {
	if jobErr := w.store.RecordJob(ctx, doc.SourceId, &doc.ID, "failed", err.Error()); jobErr != nil {
		w.log.Error("failed to record job failure", "error", jobErr)
	}
}
