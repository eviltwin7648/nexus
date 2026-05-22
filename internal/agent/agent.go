package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/eviltwin7648/nexus/internal/observe"
)

const (
	maxIterations    = 5
	openAIChatURL    = "https://api.openai.com/v1/chat/completions"
	defaultChatModel = "gpt-5.4-nano-2026-03-17"
)

type Agent struct {
	apiKey   string
	model    string
	executor *Executor
	client   *http.Client
	log      *slog.Logger
	obs      ObserveStore
}
type ObserveStore interface {
	SaveTrace(ctx context.Context, t *observe.Trace) error
}

func New(apiKey, model string, executor *Executor, log *slog.Logger, obs ObserveStore) *Agent {
	if model == "" {
		model = defaultChatModel
	}
	return &Agent{
		apiKey:   apiKey,
		model:    model,
		executor: executor,
		client:   &http.Client{Timeout: 60 * time.Second},
		log:      log,
		obs:      obs,
	}
}

type chatMessage struct {
	Role       string    `json:"role"`
	Content    any       `json:"content"`
	ToolCalls  []oaiTool `json:"tool_calls,omitempty"`
	ToolCallId string    `json:"tool_call_id,omitempty"`
	Name       string    `json:"name,omitempty"`
}

type oaiTool struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Function oaiFunction `json:"function"`
}

type oaiFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatRequest struct {
	Model      string           `json:"model"`
	Messages   []chatMessage    `json:"messages"`
	Tools      []map[string]any `json:"tools"`
	ToolChoice string           `json:"tool_choice"`
}
type chatResponse struct {
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	}
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}
type Step struct {
	Iteration int
	Tool      string
	Query     string
	ResultLen int
}

type Result struct {
	Answer string
	Steps  []Step
}

func (a *Agent) Run(ctx context.Context, question string) (*Result, error) {
	trace := observe.NewTrace(question)
	messages := []chatMessage{
		{Role: "system", Content: systemPrompt()},
		{Role: "user", Content: question},
	}
	var steps []Step
	for i := 0; i < maxIterations; i++ {
		a.log.Info("agent iteration", "iteration", i+1, "question", question)
		resp, err := a.callLLM(ctx, messages)
		if err != nil {
			trace.Status = observe.StatusFailed
			trace.Error = err.Error()
			trace.FinishedAt = time.Now()
			a.saveTrace(ctx, trace)
			return nil, fmt.Errorf("iteration %d llm call: %w", i+1, err)
		}
		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("no choices returned at iteration %d", i+1)
		}
		trace.InputTokens += resp.Usage.PromptTokens
		trace.OutputTokens += resp.Usage.CompletionTokens
		raw := resp.Choices[0].Message

		msg := chatMessage{
			Role:      raw.Role,
			ToolCalls: raw.ToolCalls,
		}

		if raw.Content != nil {
			if s, ok := raw.Content.(string); ok {
				msg.Content = s
			}
		}

		if msg.Content == "" && len(msg.ToolCalls) > 0 {
			msg.Content = ""
		}

		messages = append(messages, msg)

		if len(raw.ToolCalls) == 0 {
			content, _ := raw.Content.(string)
			trace.Answer = content
			trace.Status = observe.StatusSuccess
			trace.FinishedAt = time.Now()
			a.saveTrace(ctx, trace)
			return &Result{
				Answer: content,
				Steps:  steps,
			}, nil
		}
		finished := false
		var finalAnswer string
		for _, tc := range raw.ToolCalls {
			toolCall := ToolCall{
				Name:      tc.Function.Name,
				Arguments: json.RawMessage(tc.Function.Arguments),
			}

			if toolCall.Name == ToolFinish {
				var args FinishArgs
				if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
					return nil, fmt.Errorf("unmarshal finish args: %w", err)
				}
				finalAnswer = args.Answer
				finished = true
				messages = append(messages, chatMessage{
					Role:       "tool",
					ToolCallId: tc.ID,
					Content:    "done",
				})
				break
			}
			stepStart := time.Now()
			result := a.executor.Execute(ctx, toolCall)
			trace.EmbeddingTokens += result.EmbeddingTokens
			stepEnd := time.Now()
			if result.IsError {
				a.log.Error("tool execution failed",
					"tool", toolCall.Name,
					"error", result.Content,
				)
			}
			var inputMap map[string]any
			json.Unmarshal(toolCall.Arguments, &inputMap)
			trace.Steps = append(trace.Steps, observe.TraceStep{
				Iteration:  i + 1,
				Tool:       toolCall.Name,
				Input:      inputMap,
				OutputLen:  len(result.Content),
				TokensUsed: trace.InputTokens + trace.OutputTokens,
				StartedAt:  stepStart,
				FinishedAt: stepEnd,
			})

			steps = append(steps, Step{
				Iteration: i + 1,
				Tool:      toolCall.Name,
				ResultLen: len(result.Content),
			})

			a.log.Info("tool executed",
				"tool", toolCall.Name,
				"duration_ms", stepEnd.Sub(stepStart).Milliseconds(),
				"result_len", len(result.Content),
			)

			messages = append(messages, chatMessage{
				Role:       "tool",
				ToolCallId: tc.ID,
				Name:       tc.Function.Name,
				Content:    result.Content,
			})
		}
		if finished {
			trace.Answer = finalAnswer
			trace.Status = observe.StatusSuccess
			trace.FinishedAt = time.Now()
			a.saveTrace(ctx, trace)

			return &Result{
				Answer: finalAnswer, Steps: steps,
			}, nil
		}
	}
	a.log.Warn("max iterations reached, forcing final answer")
	finalAnswer, err := a.forceFinalAnswer(ctx, messages)
	if err != nil {
		trace.Status = observe.StatusFailed
		trace.Error = err.Error()
		trace.FinishedAt = time.Now()
		a.saveTrace(ctx, trace)

		return nil, fmt.Errorf("force final answer: %w", &err)
	}
	trace.Answer = finalAnswer
	trace.Status = observe.StatusSuccess
	trace.FinishedAt = time.Now()
	a.saveTrace(ctx, trace)

	return &Result{
		Answer: finalAnswer, Steps: steps,
	}, nil
}

func (a *Agent) forceFinalAnswer(ctx context.Context, messages []chatMessage) (string, error) {
	messages = append(messages, chatMessage{
		Role:    "user",
		Content: "You have reached the maximum number of steps. Based on everything you found so far, give your best answer now.",
	})
	body, err := json.Marshal(map[string]any{
		"model":    a.model,
		"messages": messages,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIChatURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var result chatResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("openai: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices")
	}
	content, _ := result.Choices[0].Message.Content.(string)
	return content, nil
}

func (a *Agent) callLLM(ctx context.Context, messages []chatMessage) (*chatResponse, error) {
	body, err := json.Marshal(chatRequest{
		Model:      a.model,
		Messages:   messages,
		Tools:      ToolDefinitions(),
		ToolChoice: "required",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIChatURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result chatResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("openai [%s]: %s", result.Error.Type, result.Error.Message)
	}

	return &result, nil
}

func (a *Agent) saveTrace(ctx context.Context, trace *observe.Trace) {
	if a.obs == nil {
		return
	}
	if err := a.obs.SaveTrace(ctx, trace); err != nil {
		a.log.Error("failed to save trace", "error", err)
	}
	a.log.Info(trace.Summary())
}
