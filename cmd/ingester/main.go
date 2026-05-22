package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eviltwin7648/nexus/config"
	"github.com/eviltwin7648/nexus/internal/connector/github"
	"github.com/eviltwin7648/nexus/internal/domain"
	"github.com/eviltwin7648/nexus/internal/embedder"
	"github.com/eviltwin7648/nexus/internal/enricher"
	"github.com/eviltwin7648/nexus/internal/store"
	"github.com/eviltwin7648/nexus/internal/ingestor"
)

func main() {
	//logger
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	//config
	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	log.Info("config loaded", "repos", cfg.Repos, "poll_interval", cfg.PollInterval)

	//context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// store
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer st.Close()
	log.Info("connected to postgres")

	//ingester
	ingester := worker.NewIngester(st, log)

	//connector
	// one per repo
	var connectors []connector

	for _, repo := range cfg.Repos {
		c, err := github.New(cfg.GitHubToken, repo)
		if err != nil {
			log.Error("invalid repo", "repo", repo, "error", err)
			os.Exit(1)
		}

		if err := st.RegisterSource(ctx, c.ID(), "github", map[string]any{
			"repo": repo,
		}); err != nil {
			log.Error("failed to register source", "source", c.ID(), "error", err)
			os.Exit(1)
		}
		connectors = append(connectors, c)
		log.Info("register connector", "soruce", c.ID())
	}

	//initial sync
	for _, c := range connectors {
		if err := syncConnector(ctx, c, st, ingester, log); err != nil {
			log.Error("initial sync failed", "source", c.ID(), "error", err)
		}
	}

	//enricher (runs concurrentely with the ingestion loop)

	emb := embedder.New(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingModel)
	enr := enricher.New(st, emb, log)

	go enr.Run(ctx)
	//polling ticker

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()
	log.Info("started polling loop", "interval", cfg.PollInterval)
	for {
		select {
		case <-ctx.Done():
			log.Info("shutting down")
			return
		case <-ticker.C:
			for _, c := range connectors {
				if err := syncConnector(ctx, c, st, ingester, log); err != nil {
					log.Error("sync failed", "source", c.ID(), "error", err)
				}
			}
		}
	}

}

// just a local interface (dont want main to import from documents(esp. connector))
type connector interface {
	ID() string
	Fetch(ctx context.Context) ([]domain.RawDocument, error)
	Diff(ctx context.Context, since time.Time) ([]domain.RawDocument, error)
}

func syncConnector(
	ctx context.Context,
	c connector,
	st *store.Store,
	ingester *worker.Ingester,
	log *slog.Logger,
) error {
	lastsynced, err := st.GetSourceLastSynced(ctx, c.ID())
	if err != nil {
		return fmt.Errorf("get last synced: %w", err)
	}
	var docs []domain.RawDocument
	if lastsynced.IsZero() {
		log.Info("full fetch", "source", c.ID())
		docs, err = c.Fetch(ctx)
	} else {
		log.Info("diff fetch", "source", c.ID(), "since", lastsynced)
		docs, err = c.Diff(ctx, lastsynced)
	}
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	result := ingester.Insgestbatch(ctx, docs)
	log.Info("sync complete", "source", c.ID(), "result", result.String())
	return st.UpdateSourceSyncTime(ctx, c.ID())
}
