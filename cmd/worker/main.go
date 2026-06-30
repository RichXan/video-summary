package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"video-summary-mvp/internal/adapters"
	"video-summary-mvp/internal/app"
	"video-summary-mvp/internal/bootstrap"
	"video-summary-mvp/internal/config"
)

func main() {
	cfg := config.Load()
	configureLogger(cfg)
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		slog.Error("DATABASE_URL is required for worker")
		os.Exit(1)
	}

	store, err := adapters.OpenPostgresJobStore(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("open job store failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = store.Close() }()

	service := bootstrap.BuildSummaryService(cfg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	concurrency := cfg.WorkerConcurrency
	if concurrency < 1 {
		concurrency = 1
	}
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		workerID := cfg.WorkerID
		if workerID == "" {
			workerID = "worker"
		}
		workerID = fmt.Sprintf("%s-%d", workerID, i+1)
		worker := app.NewWorker(app.WorkerConfig{
			Store:        store,
			Service:      service,
			WorkerID:     workerID,
			PollInterval: cfg.WorkerPollInterval,
			JobTimeout:   cfg.JobTimeout,
			RunningTTL:   cfg.RunningJobTTL,
			Logger:       slog.Default(),
		})
		go func() {
			defer wg.Done()
			if err := worker.Run(ctx); err != nil && ctx.Err() == nil {
				slog.Error("worker stopped", "error", err)
			}
		}()
	}
	slog.Info("worker started", "concurrency", concurrency)
	<-ctx.Done()
	slog.Info("worker shutting down")
	wg.Wait()
}

func configureLogger(cfg config.Config) {
	if strings.EqualFold(cfg.LogFormat, "text") {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
		return
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
}
