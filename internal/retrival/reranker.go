package retriver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Candidate struct {
	ID    string            `json:"id"`
	Text  string            `json:"text"`
	Score float64           `json:"score"`
	Rank  int               `json:"rank"`
	Meta  map[string]string `json:"meta,omitempty"`
}

const Topk = 8

type Reranker interface {
	Rerank(ctx context.Context, query string, candidates []Candidate) ([]RankedCandidate, error)
}

type CohereReranker struct {
	cohereApiKey string
	model        string
	client       *http.Client
}

func NewCohereReranker(apiKey, model string) Reranker {
	return &CohereReranker{
		cohereApiKey: apiKey,
		model:        model,
		client:       &http.Client{},
	}
}

type cohereRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

type cohereRerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type cohereRerankResponse struct {
	Id      string               `json:"id"`
	Results []cohereRerankResult `json:"results"`
}

type RankedCandidate struct {
	Candidate Candidate `json:"candidate"`
	Score     float64   `json:"score"`
	Rank      int       `json:"rank"`
}

func (r *CohereReranker) Rerank(ctx context.Context, query string, candidates []Candidate) ([]RankedCandidate, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	if r.cohereApiKey == "" {
		var results []RankedCandidate
		limit := len(candidates)
		if limit > Topk {
			limit = Topk
		}
		for i := 0; i < limit; i++ {
			results = append(results, RankedCandidate{
				Candidate: candidates[i],
				Score:     candidates[i].Score,
				Rank:      i + 1,
			})
		}
		return results, nil
	}

	documents := make([]string, len(candidates))
	for i, c := range candidates {
		documents[i] = c.Text
	}

	model := r.model
	if model == "" {
		model = "rerank-v4.0-pro"
	}

	request := cohereRerankRequest{
		Model:     model,
		Query:     query,
		Documents: documents,
		TopN:      Topk,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.cohere.com/v2/rerank", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer "+r.cohereApiKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%s): %s", resp.Status, string(bodyBytes))
	}

	var rerankResp cohereRerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rerankResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var results []RankedCandidate
	for i, res := range rerankResp.Results {
		if res.Index < 0 || res.Index >= len(candidates) {
			return nil, fmt.Errorf("API returned invalid document index: %d", res.Index)
		}
		results = append(results, RankedCandidate{
			Candidate: candidates[res.Index],
			Score:     res.RelevanceScore,
			Rank:      i + 1,
		})
	}

	return results, nil
}
