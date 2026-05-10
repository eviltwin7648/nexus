package observe

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// to be checked with the openai docs for latest pricing details.
const (
	costPer1kInputTokens  = 0.000150
	costPer1kOutputTokens = 0.000600
	embeddingCostPer1k    = 0.000020
	charsPerToken         = 4
)

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

type Trace struct {
	ID              string
	Question        string
	Answer          string
	Status          string
	Error           string
	Steps           []TraceStep
	InputTokens     int
	OutputTokens    int
	EmbeddingTokens int
	StartedAt       time.Time
	FinishedAt      time.Time
}

type TraceStep struct {
	Iteration  int
	Tool       string
	Input      map[string]any
	OutputLen  int
	TokensUsed int
	StartedAt  time.Time
	FinishedAt time.Time
}

func (t *Trace) TotalMS() int {
	return int(t.FinishedAt.Sub(t.StartedAt).Milliseconds())
}

func (t *Trace) TotalTokens() int {
	return t.InputTokens + t.OutputTokens + t.EmbeddingTokens
}

// func (t *Trace) EstimatedCostUSD() float64 {
// 	tokens := float64(t.TotalTokens())

//		inputCost := (tokens * 0.7 / 1000) * costPer1kInputTokens
//		outputCost := (tokens * 0.3 / 1000) * costPer1kOutputTokens
//		return inputCost + outputCost
//	}
func (t *Trace) TotalCostUSD() float64 {
	llmInput := float64(t.InputTokens) / 1000 * costPer1kInputTokens
	llmOutput := float64(t.OutputTokens) / 1000 * costPer1kOutputTokens
	embedding := float64(t.EmbeddingTokens) / 1000 * embeddingCostPer1k
	return llmInput + llmOutput + embedding
}

func (s *TraceStep) DurationMS() int {
	return int(s.FinishedAt.Sub(s.StartedAt).Milliseconds())
}

func NewTrace(question string) *Trace {
	return &Trace{
		ID:        uuid.New().String(),
		Question:  question,
		Status:    StatusFailed, //default - will be changed later
		StartedAt: time.Now(),
	}
}

// func EstimateTokens(text string) int {
// 	if len(text) == 0 {
// 		return 0
// 	}
// 	t := len(text) / charsPerToken
// 	if t == 0 {
// 		return 1
// 	}
// 	return t
// }

func (t *Trace) Summary() string {
	return fmt.Sprintf(
		"trace=%s status=%s steps=%d input_tokens=%d output_tokens=%d emb_tokens=%d cost=$%.6f duration=%dms",
		t.ID[:8], t.Status, len(t.Steps),
		t.InputTokens, t.OutputTokens, t.EmbeddingTokens,
		t.TotalCostUSD(), t.TotalMS(),
	)
}
