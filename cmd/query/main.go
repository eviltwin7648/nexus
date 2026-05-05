package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/eviltwin7648/nexus/config"
	"github.com/eviltwin7648/nexus/internal/agent"
	"github.com/eviltwin7648/nexus/internal/embedder"
	"github.com/eviltwin7648/nexus/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db error: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	emb := embedder.New(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingModel)
	executor := agent.NewExecutor(st, emb)
	ag := agent.New(cfg.OpenAIAPIKey, cfg.OpenAIChatModel, executor, log)
	// engine := query.New(st, emb, cfg.OpenAIAPIKey, cfg.OpenAIChatModel)

	_ = log
	fmt.Println("Nexus — ask anything about your repos and notes.")
	fmt.Println("Type 'exit' to quit.\n")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		question := strings.TrimSpace(scanner.Text())
		if question == "" {
			continue
		}
		if question == "exit" {
			break
		}
		result, err := ag.Run(ctx, question)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		fmt.Printf("\nNexus: %s\n", result.Answer)
		if len(result.Steps) > 0 {
			fmt.Println("\nReasoning steps:")
			for _, s := range result.Steps {
				fmt.Printf("  [%d] %s → %d chars retrieved\n",
					s.Iteration, s.Tool, s.ResultLen)
			}
		}

		fmt.Println()

	}
}
