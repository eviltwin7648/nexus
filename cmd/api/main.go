package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eviltwin7648/nexus/config"
	"github.com/eviltwin7648/nexus/internal/agent"
	"github.com/eviltwin7648/nexus/internal/embedder"
	"github.com/eviltwin7648/nexus/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM,
	)
	defer cancel()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	emb := embedder.New(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingModel)
	executor := agent.NewExecutor(st, emb)
	ag := agent.New(cfg.OpenAIAPIKey, cfg.OpenAIChatModel, executor, log)

	h := &handlers{ag: ag, log: log}

	// register routes
	mux := http.NewServeMux()
	mux.HandleFunc("/query", h.query)
	mux.HandleFunc("/health", h.health)

	port := cfg.APIPort
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// start server in goroutine so we can listen for shutdown signal
	go func() {
		log.Info("api server started", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// wait for shutdown signal
	<-ctx.Done()
	log.Info("shutting down api server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "error", err)
	}
}
