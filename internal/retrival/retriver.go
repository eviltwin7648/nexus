package retriver

import (
	"context"
	"fmt"
	"sort"

	"github.com/eviltwin7648/nexus/internal/embedder"
	"github.com/eviltwin7648/nexus/internal/store"
	"golang.org/x/sync/errgroup"
)

type Retriver struct {
	store    *store.Store
	embedder *embedder.Embedder
	reranker Reranker
}

func NewRetriver(store *store.Store, emb *embedder.Embedder, reranker Reranker) *Retriver {
	return &Retriver{
		store:    store,
		embedder: emb,
		reranker: reranker,
	}
}

func (s *Retriver) Store() *store.Store {
	return s.store
}

func (s *Retriver) HybridSearch(ctx context.Context, query string, sourceType *string) ([]RankedCandidate, error) {
	g, groupCtx := errgroup.WithContext(ctx)
	var lex []store.ChunkResult
	var vec []store.ChunkResult

	g.Go(func() error {
		var err error
		lex, err = s.LexicalSearch(groupCtx, query, sourceType)
		return err
	})

	g.Go(func() error {
		var err error
		vec, err = s.VectorSearch(groupCtx, query, sourceType)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	rrfResults, err := mergeResults(lex, vec)
	if err != nil {
		return nil, err
	}

	candidates := make([]Candidate, len(rrfResults))
	for i, r := range rrfResults {
		candidates[i] = Candidate{
			ID:    r.ChunkResult.Id,
			Text:  r.ChunkResult.Content,
			Score: r.RRFScore,
			Rank:  i + 1,
			Meta: map[string]string{
				"path":        r.ChunkResult.Path,
				"doc_id":      r.ChunkResult.DocId,
				"source_id":   r.ChunkResult.SourceId,
				"source_type": r.ChunkResult.SourceType,
			},
		}
	}

	ranked, err := s.reranker.Rerank(ctx, query, candidates)
	if err != nil {
		return nil, fmt.Errorf("rerank error: %w", err)
	}

	return ranked, nil
}

func (s *Retriver) LexicalSearch(ctx context.Context, query string, sourceType *string) ([]store.ChunkResult, error) {
	chunks, err := s.store.LexicalSearch(ctx, query, 50, sourceType)
	if err != nil {
		return nil, err
	}
	return chunks, nil
}

func (s *Retriver) VectorSearch(ctx context.Context, query string, sourceType *string) ([]store.ChunkResult, error) {
	queryVec, _, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(queryVec) == 0 {
		return nil, fmt.Errorf("empty query vector")
	}
	if sourceType == nil {
		chunks, err := s.store.SearchChunks(ctx, queryVec, 50)
		if err != nil {
			return nil, err
		}
		return chunks, nil
	}
	chunks, err := s.store.SearchChunksByType(ctx, queryVec, *sourceType, 50)
	if err != nil {
		return nil, err
	}
	return chunks, nil
}

type MergedResult struct {
	ChunkResult store.ChunkResult
	RRFScore    float64 `json:"rrf_score"`
}

func mergeResults(lex, vec []store.ChunkResult) ([]MergedResult, error) {
	const k = 60.0 //smoothing constant
	rrfScores := make(map[string]float64)
	chunkMap := make(map[string]store.ChunkResult)

	for rank, item := range lex {
		rrfScores[item.Id] += 1.0 / (float64(rank+1) + k)
		chunkMap[item.Id] = item
	}

	for rank, item := range vec {
		rrfScores[item.Id] += 1.0 / (float64(rank+1) + k)
		chunkMap[item.Id] = item
	}

	var results []MergedResult
	for id, score := range rrfScores {
		results = append(results, MergedResult{
			ChunkResult: chunkMap[id],
			RRFScore:    score,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].RRFScore > results[j].RRFScore
	})

	if len(results) > 20 {
		results = results[:20]
	}
	return results, nil
}
