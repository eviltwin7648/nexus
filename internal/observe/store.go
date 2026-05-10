package observe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TraceSummary struct {
	ID               string    `json:"id"`
	Question         string    `json:"question"`
	Status           string    `json:"status"`
	TotalMS          int       `json:"total_ms"`
	TotalTokens      int       `json:"total_tokens"`
	EstimatedCostUSD float64   `json:"estimated_cost_usd"`
	CreatedAt        time.Time `json:"created_at"`
}

type TraceDetail struct {
	TraceSummary
	Answer string       `json:"answer"`
	Error  string       `json:"error,omitempty"`
	Steps  []StepDetail `json:"steps"`
}

type StepDetail struct {
	Iteration  int            `json:"iteration"`
	Tool       string         `json:"tool"`
	Input      map[string]any `json:"input"`
	OutputLen  int            `json:"output_len"`
	DurationMS int            `json:"duration_ms"`
	TokensUsed int            `json:"tokens_used"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Stats struct {
	TotalQueries    int     `json:"total_queries"`
	Successful      int     `json:"successful"`
	Failed          int     `json:"failed"`
	AvgDurationMS   float64 `json:"avg_duration_ms"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	EmbeddingTokens int     `json:"embedding_tokens"`
	TotalCostUSD    float64 `json:"total_cost_usd"`
}
type ObserveStore struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *ObserveStore {
	return &ObserveStore{pool: pool}
}

func (s *ObserveStore) SaveTrace(ctx context.Context, t *Trace) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// insert trace
	_, err = tx.Exec(ctx, `
		INSERT INTO traces (
				id, question, answer, status, error,
				total_ms, total_tokens, input_tokens, output_tokens,
				embedding_tokens, estimated_cost_usd, created_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT (id) DO UPDATE SET
				answer           = EXCLUDED.answer,
				status           = EXCLUDED.status,
				error            = EXCLUDED.error,
				total_ms         = EXCLUDED.total_ms,
				total_tokens     = EXCLUDED.total_tokens,
				input_tokens     = EXCLUDED.input_tokens,
				output_tokens    = EXCLUDED.output_tokens,
				embedding_tokens = EXCLUDED.embedding_tokens,
				estimated_cost_usd = EXCLUDED.estimated_cost_usd
	`,
		t.ID, t.Question, t.Answer, t.Status, t.Error,
		t.TotalMS(), t.TotalTokens(),
		t.InputTokens, t.OutputTokens, t.EmbeddingTokens,
		t.TotalCostUSD(), t.StartedAt,
	)
	if err != nil {
		return fmt.Errorf("insert trace: %w", err)
	}

	// insert steps
	for _, step := range t.Steps {
		inputJSON, err := json.Marshal(step.Input)
		if err != nil {
			inputJSON = []byte("{}")
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO trace_steps (
				trace_id, iteration, tool, input,
				output_len, duration_ms, tokens_used, created_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		`,
			t.ID, step.Iteration, step.Tool, inputJSON,
			step.OutputLen, step.DurationMS(), step.TokensUsed,
			step.StartedAt,
		)
		if err != nil {
			return fmt.Errorf("insert step: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (s *ObserveStore) ListTraces(ctx context.Context, limit int) ([]TraceSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := s.pool.Query(ctx, `
		SELECT
			id, question, status, total_ms,
			total_tokens, estimated_cost_usd, created_at
		FROM traces
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []TraceSummary
	for rows.Next() {
		var ts TraceSummary
		if err := rows.Scan(
			&ts.ID, &ts.Question, &ts.Status,
			&ts.TotalMS, &ts.TotalTokens, &ts.EstimatedCostUSD,
			&ts.CreatedAt,
		); err != nil {
			return nil, err
		}
		summaries = append(summaries, ts)
	}

	return summaries, rows.Err()
}
func (s *ObserveStore) GetTrace(ctx context.Context, id string) (*TraceDetail, error) {
	var td TraceDetail

	err := s.pool.QueryRow(ctx, `
		SELECT
			id, question, answer, status, error,
			total_ms, total_tokens, estimated_cost_usd, created_at
		FROM traces WHERE id = $1
	`, id).Scan(
		&td.ID, &td.Question, &td.Answer, &td.Status, &td.Error,
		&td.TotalMS, &td.TotalTokens, &td.EstimatedCostUSD,
		&td.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get trace: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT iteration, tool, input, output_len, duration_ms, tokens_used, created_at
		FROM trace_steps
		WHERE trace_id = $1
		ORDER BY id ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get steps: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var step StepDetail
		var inputJSON []byte
		if err := rows.Scan(
			&step.Iteration, &step.Tool, &inputJSON,
			&step.OutputLen, &step.DurationMS, &step.TokensUsed,
			&step.CreatedAt,
		); err != nil {
			return nil, err
		}
		json.Unmarshal(inputJSON, &step.Input)
		td.Steps = append(td.Steps, step)
	}

	return &td, rows.Err()
}
func (s *ObserveStore) GetStats(ctx context.Context) (*Stats, error) {
	var st Stats
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)                                     AS total_queries,
			COUNT(*) FILTER (WHERE status='success')     AS successful,
			COUNT(*) FILTER (WHERE status='failed')      AS failed,
			COALESCE(AVG(total_ms), 0)                   AS avg_duration_ms,
			COALESCE(SUM(input_tokens), 0)               AS input_tokens,
			COALESCE(SUM(output_tokens), 0)              AS output_tokens,
			COALESCE(SUM(embedding_tokens), 0)           AS embedding_tokens,
			COALESCE(SUM(estimated_cost_usd), 0)         AS total_cost_usd
		FROM traces
`).Scan(
		&st.TotalQueries, &st.Successful, &st.Failed,
		&st.AvgDurationMS,
		&st.InputTokens, &st.OutputTokens, &st.EmbeddingTokens,
		&st.TotalCostUSD,
	)
	return &st, err
}
