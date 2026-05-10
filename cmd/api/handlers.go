package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/eviltwin7648/nexus/internal/agent"
	"github.com/eviltwin7648/nexus/internal/observe"
)

type queryRequest struct {
	Question string `json:"question"`
	TopK     int    `json:"tok_k"`
}

type queryResponse struct {
	Answer   string     `json:"answer"`
	Steps    []stepJSON `json:"steps"`
	Duration string     `json:"duration"`
}

type stepJSON struct {
	Iteration int    `josn:"iteration"`
	Tool      string `json:"tool"`
	Retrieved int    `json:"chars_retrieved"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type handlers struct {
	ag  *agent.Agent
	log *slog.Logger
	obs *observe.ObserveStore
}

// POST /query
func (h *handlers) query(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Question == "" {
		writeError(w, "question is required", http.StatusBadRequest)
		return
	}

	h.log.Info("query received", "question", req.Question)
	start := time.Now()

	result, err := h.ag.Run(r.Context(), req.Question)
	if err != nil {
		h.log.Error("agent failed", "error", err)
		writeError(w, "agent error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	steps := make([]stepJSON, len(result.Steps))
	for i, s := range result.Steps {
		steps[i] = stepJSON{
			Iteration: s.Iteration,
			Tool:      s.Tool,
			Retrieved: s.ResultLen,
		}
	}

	writeJSON(w, queryResponse{
		Answer:   result.Answer,
		Steps:    steps,
		Duration: time.Since(start).Round(time.Millisecond).String(),
	}, http.StatusOK)
}

// GET /health
func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}, http.StatusOK)
}

// GET /traces?limit=20
func (h *handlers) listTraces(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	traces, err := h.obs.ListTraces(r.Context(), limit)
	if err != nil {
		writeError(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, traces, http.StatusOK)
}

// GET /traces/{id}
func (h *handlers) getTrace(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/traces/")
	if id == "" {
		writeError(w, "trace id required", http.StatusBadRequest)
		return
	}

	trace, err := h.obs.GetTrace(r.Context(), id)
	if err != nil {
		writeError(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, trace, http.StatusOK)
}

// GET /stats
func (h *handlers) stats(w http.ResponseWriter, r *http.Request) {
	st, err := h.obs.GetStats(r.Context())
	if err != nil {
		writeError(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, st, http.StatusOK)
}

// --- helpers -----------------------------------------------------------------

func writeJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, msg string, status int) {
	writeJSON(w, errorResponse{Error: msg}, status)
}
