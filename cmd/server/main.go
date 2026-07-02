package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"video-summary-mvp/internal/adapters"
	"video-summary-mvp/internal/app"
	"video-summary-mvp/internal/bootstrap"
	"video-summary-mvp/internal/config"
	"video-summary-mvp/internal/httpapi"
)

func main() {
	cfg := config.Load()
	configureLogger(cfg)

	service := bootstrap.BuildSummaryService(cfg)
	var jobs httpapi.JobService
	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		store, err := adapters.OpenPostgresJobStore(context.Background(), cfg.DatabaseURL)
		if err != nil {
			slog.Error("open job store failed", "error", err)
			os.Exit(1)
		}
		defer func() { _ = store.Close() }()
		jobs = httpapi.AppJobAdapter{Service: app.NewJobService(store)}
	}

	handler := httpapi.NewHandlerWithOptions(httpapi.HandlerOptions{
		Summary: httpapi.AppAdapter{Service: service},
		Jobs:    jobs,
		APIKey:  cfg.APIKey,
	})

	slog.Info("api listening", "addr", cfg.Addr, "mode", cfg.Mode, "video_resolver", cfg.VideoResolver, "media_preparer", cfg.MediaPreparer, "summary_mode", cfg.SummaryMode)
	if err := http.ListenAndServe(cfg.Addr, handler.Routes()); err != nil {
		slog.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func configureLogger(cfg config.Config) {
	if strings.EqualFold(cfg.LogFormat, "text") {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
		return
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
}
