package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const openAIEmbeddingURL = "https://api.openai.com/v1/embeddings"

type Embedder struct {
	apiKey string
	model  string
	client *http.Client
}

func New(apiKey, model string) *Embedder {
	return &Embedder{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embeddingResponse struct { //ref openaidocs
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (e *Embedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	// gpt error : openai error [invalid_request_error]: Invalid 'input[0]': input cannot be an empty string."
	// to solve we add a map for original non-empty strings with its index, later we restore the position
	indexMap := make([]int, 0, len(texts))
	filtered := make([]string, 0, len(texts))
	for i, t := range texts {
		if strings.TrimSpace(t) != "" {
			filtered = append(filtered, t)
			indexMap = append(indexMap, i)
		}
	}
	if len(filtered) == 0 {
		return make([][]float32, len(texts)), nil // all empty — return nil vectors
	}
	body, err := json.Marshal(embeddingRequest{
		Input: filtered,
		Model: e.model,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEmbeddingURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request :%w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	var result embeddingResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("unmarshall response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("openai error [%s]: %s", result.Error.Type, result.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned %d: %s", resp.StatusCode, raw)
	}
	vectors := make([][]float32, len(texts))
	for filteredIdx, d := range result.Data {
		originalIdx := indexMap[filteredIdx]
		vectors[originalIdx] = d.Embedding
	}

	return vectors, nil
}

func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vectors, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 || vectors[0] == nil {
		return nil, fmt.Errorf("no embedding returned")
	}
	return vectors[0], nil
}
