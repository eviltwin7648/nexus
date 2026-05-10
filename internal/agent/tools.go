package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eviltwin7648/nexus/internal/store"
)

const (
	ToolSearch      = "search"
	ToolFilter      = "filter"
	ToolGetDocument = "get_document"
	ToolFinish      = "finish"
)

type ToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	Tool            string `json:"name"`
	Content         string `json:"content"`
	IsError         bool   `json:"is_error"`
	EmbeddingTokens int    `json:"embedding_tokens"`
}

type SearchArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type FilterArgs struct {
	Query      string `json:"query"`
	SourceType string `json:"source_type"`
	Limit      int    `json:"limit"`
}

type GetDocumentArgs struct {
	Path string `json:"path"`
}

type FinishArgs struct {
	Answer string `json:"answer"`
}

func ToolDefinitions() []map[string]any {
	return []map[string]any{
		{
			"type": "function",
			"function": map[string]any{
				"name":        ToolSearch,
				"description": "Semantically search across all ingested documents — code files, issues, PRs, and notes. Use this first for any question.",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "Natural language search query",
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "Number of results to return (1-10, default 5)",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        ToolFilter,
				"description": "Search within a specific document type. Use when you want only code files, only issues, or only notes.",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "Natural language search query",
						},
						"source_type": map[string]any{
							"type":        "string",
							"enum":        []string{"github_file", "github_issue", "github_pr"},
							"description": "Document type to search within",
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "Number of results to return (1-10, default 5)",
						},
					},
					"required": []string{"query", "source_type"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        ToolGetDocument,
				"description": "Fetch the full content of a specific document by its file path. Use when search gives you a promising file but you need the complete content.",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "File path of the document e.g. 'Devfleet-backend/src/queue/processor.ts'",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        ToolFinish,
				"description": "Return the final answer to the user. Call this when you have enough information to answer confidently.",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"answer": map[string]any{
							"type":        "string",
							"description": "The complete, well-structured answer to the user's question",
						},
					},
					"required": []string{"answer"},
				},
			},
		},
	}
}

type Executor struct {
	store *store.Store
	emb   Embedder
}

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, int, error)
}

func NewExecutor(st *store.Store, emb Embedder) *Executor {
	return &Executor{store: st, emb: emb}
}

func (e *Executor) Execute(ctx context.Context, call ToolCall) ToolResult {
	switch call.Name {
	case ToolSearch:
		return e.execSearch(ctx, call.Arguments)
	case ToolFilter:
		return e.execFilter(ctx, call.Arguments)
	case ToolGetDocument:
		return e.execGetDocument(ctx, call.Arguments)

	default:
		return ToolResult{
			Tool:    call.Name,
			Content: fmt.Sprintf("unknown tool: %s", call.Name),
			IsError: true,
		}
	}
}

func (e *Executor) execSearch(ctx context.Context, raw json.RawMessage) ToolResult {
	var args SearchArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{Tool: ToolSearch, Content: "invalid arguments", IsError: true}
	}
	if args.Limit <= 0 || args.Limit > 0 {
		args.Limit = 5
	}
	vec, tokens, err := e.emb.Embed(ctx, args.Query)
	if err != nil {
		return ToolResult{Tool: ToolSearch, Content: fmt.Sprintf("embed error: %v", err), IsError: true}
	}
	chunks, err := e.store.SearchChunks(ctx, vec, args.Limit)
	if err != nil {
		return ToolResult{
			Tool:    ToolSearch,
			Content: fmt.Sprintf("search error: %w", err),
			IsError: true,
		}
	}
	return ToolResult{Tool: ToolSearch, Content: formatChunks(chunks), EmbeddingTokens: tokens}
}

func (e *Executor) execFilter(ctx context.Context, raw json.RawMessage) ToolResult {
	var args FilterArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{Tool: ToolFilter, Content: "invalid arguments", IsError: true}
	}
	if args.Limit <= 0 || args.Limit >= 10 {
		args.Limit = 5
	}
	vec, tokens, err := e.emb.Embed(ctx, args.Query)
	if err != nil {
		return ToolResult{Tool: ToolFilter, Content: fmt.Sprintf("embed error: %v", err), IsError: true}
	}
	chunks, err := e.store.SearchChunksByType(ctx, vec, args.SourceType, args.Limit)
	if err != nil {
		return ToolResult{Tool: ToolFilter, Content: fmt.Sprintf("search error: %v", err), IsError: true}
	}
	return ToolResult{Tool: ToolFilter, Content: formatChunks(chunks), EmbeddingTokens: tokens}
}

func (e *Executor) execGetDocument(ctx context.Context, raw json.RawMessage) ToolResult {
	var args GetDocumentArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{Tool: ToolGetDocument, Content: "invalid arguments", IsError: true}
	}

	doc, err := e.store.GetDocumentByPath(ctx, args.Path)
	if err != nil {
		return ToolResult{Tool: ToolGetDocument, Content: fmt.Sprintf("not found: %v", err), IsError: true}
	}

	// truncate very large documents
	content := doc.Content
	if len(content) > 8000 {
		content = content[:8000] + "\n... [truncated]"
	}

	return ToolResult{
		Tool:    ToolGetDocument,
		Content: fmt.Sprintf("Path: %s\n\n%s", doc.Path, content),
	}
}

func formatChunks(chunks []store.ChunkResult) string {
	if len(chunks) == 0 {
		return "No results found."
	}

	var sb strings.Builder
	for i, c := range chunks {
		sb.WriteString(fmt.Sprintf("[%d] %s (score: %.2f)\n%s\n\n",
			i+1, c.Path, c.Score, c.Content))
	}
	return sb.String()
}
