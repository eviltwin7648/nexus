package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eviltwin7648/nexus/internal/embedder"
	"github.com/eviltwin7648/nexus/internal/store"
)

const (
	defaultTopK      = 8
	defaultMaxTokens = 1024
)

type Result struct {
	Answer  string
	Sources []Source
}

type Source struct {
	Path       string
	SourceId   string
	SourceType string
	ChunkIndex int
	Content    string
	Score      float64
}

type Engine struct {
	store    *store.Store
	embedder *embedder.Embedder
	apiKey   string
	model    string
	client   *http.Client
}

func New(st *store.Store, emb *embedder.Embedder, apiKey, model string) *Engine {
	return &Engine{
		store:    st,
		embedder: emb,
		apiKey:   apiKey,
		model:    model,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (e *Engine) Query(ctx context.Context, question string, topK int) (*Result, error) {
	if topK <= 0 {
		topK = defaultTopK
	}
	qvec, err := e.embedder.Embed(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("embed question :%w", err)
	}
	chunks, err := e.store.SearchChunks(ctx, qvec, topK)
	if len(chunks) == 0 {
		return &Result{
			Answer: "I couldn't find any relevant information for that question."}, nil
	}

	prompt := buildPrompt(question, chunks)
	answer, err := e.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("call llm: %w", err)
	}

	sources := make([]Source, len(chunks))
	for i, c := range chunks {
		snippet := c.Content
		if len(snippet) > 300 {
			snippet = snippet[:300] + "..."
		}
		sources[i] = Source{
			Path:       c.Path,
			SourceId:   c.SourceId,
			SourceType: c.SourceType,
			ChunkIndex: c.ChunkIndex,
			Content:    snippet,
			Score:      c.Score,
		}
	}
	return &Result{
		Answer:  answer,
		Sources: sources,
	}, nil
}

func buildPrompt(question string, chunks []store.ChunkResult) string {
	var sb strings.Builder
	sb.WriteString("You are a helpful assistant with access to a personal knowledge base.\n")
	sb.WriteString("Answer the question using ONLY the context provided below.\n")
	sb.WriteString("If the context doesn't contain enough information, say so honestly.\n")
	sb.WriteString("Always mention which file or source your answer comes from.\n\n")
	sb.WriteString("--- CONTEXT ---\n\n")

	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("[%d] Source: %s (type: %s)\n", i+1, chunk.Path, chunk.SourceType))
		sb.WriteString(chunk.Content)
		sb.WriteString("\n\n")
	}
	sb.WriteString("--- QUESTION ---\n\n")
	sb.WriteString(question)

	return sb.String()
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_completion_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (e *Engine) callLLM(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model: e.model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: defaultMaxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	var result ChatResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("openai error [%s]: %s", result.Error.Type, result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return result.Choices[0].Message.Content, nil

}
