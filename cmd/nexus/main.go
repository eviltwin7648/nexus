package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/eviltwin7648/nexus/config"
	"github.com/eviltwin7648/nexus/internal/agent"
	"github.com/eviltwin7648/nexus/internal/connector/github"
	"github.com/eviltwin7648/nexus/internal/domain"
	"github.com/eviltwin7648/nexus/internal/embedder"
	"github.com/eviltwin7648/nexus/internal/enricher"
	worker "github.com/eviltwin7648/nexus/internal/ingestor"
	"github.com/eviltwin7648/nexus/internal/observe"
	"github.com/eviltwin7648/nexus/internal/parser"
	"github.com/eviltwin7648/nexus/internal/parser/languages"
	retriver "github.com/eviltwin7648/nexus/internal/retrival"
	"github.com/eviltwin7648/nexus/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "index":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: missing repository link/name. Usage: nexus index <github_repo_link>")
			os.Exit(1)
		}
		runIndex(os.Args[2])
	case "ask":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: missing question. Usage: nexus ask <question>")
			os.Exit(1)
		}
		question := strings.Join(os.Args[2:], " ")
		runAsk(question)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Nexus — A CLI personal knowledge base assistant")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  nexus <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  index <github_repo_link>   Fetch, ingest, and index (embed) a GitHub repository synchronously.")
	fmt.Println("  ask <question>             Ask a question about the indexed repositories.")
	fmt.Println("  help                       Show this help information.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  nexus index https://github.com/google/googletest")
	fmt.Println("  nexus ask \"What is the main test macro in googletest?\"")
}

func runIndex(repoArg string) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	repo, err := parseGitHubRepo(repoArg)
	if err != nil {
		log.Error("invalid repository argument", "error", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Connect to postgres
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	// Initialize GitHub connector
	c, err := github.New(cfg.GitHubToken, repo)
	if err != nil {
		log.Error("failed to initialize github connector", "repo", repo, "error", err)
		os.Exit(1)
	}

	// Register source
	if err := st.RegisterSource(ctx, c.ID(), "github", map[string]any{
		"repo": repo,
	}); err != nil {
		log.Error("failed to register source", "source", c.ID(), "error", err)
		os.Exit(1)
	}
	log.Info("registered repository source", "source", c.ID())

	// Initialize ingester
	ingester := worker.NewIngester(st, log)

	log.Info("synchronizing repository content from GitHub...", "repo", repo)
	lastSynced, err := st.GetSourceLastSynced(ctx, c.ID())
	if err != nil {
		log.Error("failed to get last sync time", "error", err)
		os.Exit(1)
	}

	var docs []domain.RawDocument
	if lastSynced.IsZero() {
		log.Info("performing full repository fetch")
		docs, err = c.Fetch(ctx)
	} else {
		log.Info("performing delta fetch since last sync", "since", lastSynced)
		docs, err = c.Diff(ctx, lastSynced)
	}
	if err != nil {
		log.Error("failed to fetch from GitHub", "error", err)
		os.Exit(1)
	}

	log.Info("ingesting raw documents to database...")
	result := ingester.Insgestbatch(ctx, docs)
	log.Info("ingestion completed", "result", result.String())

	if err := st.UpdateSourceSyncTime(ctx, c.ID()); err != nil {
		log.Error("failed to update source sync time", "error", err)
		os.Exit(1)
	}

	// Synchronously run enrichment loop until all documents are processed
	log.Info("processing, chunking, and embedding documents...")
	parserRegistry := parser.NewRegistry(
		languages.NewTypeScriptParser(),
		languages.NewJavaScriptParser(),
		languages.NewGoParser(),
		languages.NewPythonParser(),
		languages.NewJavaParser(),
	)
	emb := embedder.New(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingModel)
	enr := enricher.New(st, emb, log, parserRegistry)

	totalEnriched := 0
	for {
		processed, err := enr.ProcessBatch(ctx)
		if err != nil {
			log.Error("enrichment batch failed", "error", err)
			os.Exit(1)
		}
		if processed == 0 {
			break
		}
		totalEnriched += processed
		log.Info("processed enrichment batch", "documents", processed)
	}

	log.Info("indexing finished successfully!", "repo", repo, "documents_enriched", totalEnriched)
}

func runAsk(question string) {
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
	cohereKey := os.Getenv("COHERE_API_KEY")
	reranker := retriver.NewCohereReranker(cohereKey, os.Getenv("COHERE_MODEL"))
	r := retriver.NewRetriver(st, emb, reranker)
	executor := agent.NewExecutor(r)
	obsStore := observe.NewStore(st.Pool())
	ag := agent.New(cfg.OpenAIAPIKey, cfg.OpenAIChatModel, executor, log, obsStore)

	fmt.Printf("Question: %s\n", question)
	fmt.Println("Thinking...")
	fmt.Println()

	result, err := ag.Run(ctx, question)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Answer: %s\n", result.Answer)
	if len(result.Steps) > 0 {
		fmt.Println()
		fmt.Println("Reasoning steps:")
		for _, s := range result.Steps {
			fmt.Printf("  [%d] %s → %d chars retrieved\n",
				s.Iteration, s.Tool, s.ResultLen)
		}
	}
}

func parseGitHubRepo(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty repository argument")
	}

	if strings.HasPrefix(input, "https://") {
		input = strings.TrimPrefix(input, "https://")
	} else if strings.HasPrefix(input, "http://") {
		input = strings.TrimPrefix(input, "http://")
	}

	if strings.HasPrefix(input, "github.com/") {
		input = strings.TrimPrefix(input, "github.com/")
	}

	input = strings.Trim(input, "/")

	parts := strings.Split(input, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid repository format %q; expected 'owner/repo' or GitHub URL", input)
	}
	return parts[0] + "/" + parts[1], nil
}
