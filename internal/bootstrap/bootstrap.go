package bootstrap

import (
	"strings"

	"video-summary-mvp/internal/adapters"
	"video-summary-mvp/internal/app"
	"video-summary-mvp/internal/config"
)

func BuildSummaryService(cfg config.Config) *app.Service {
	transcriber := app.Transcriber(adapters.StaticTranscriber{})
	if strings.EqualFold(cfg.Mode, "command") {
		transcriber = adapters.CommandTranscriber{Name: cfg.ASRCommand, Args: cfg.ASRArgs}
	} else if strings.EqualFold(cfg.Mode, "http") {
		transcriber = adapters.HTTPASRTranscriber{BaseURL: cfg.ASRBaseURL, Model: cfg.ASRModel}
	}

	videoResolver := app.VideoResolver(adapters.StaticVideoResolver{})
	if strings.EqualFold(cfg.VideoResolver, "web") {
		videoResolver = adapters.WebVideoResolver{}
	} else if strings.EqualFold(cfg.VideoResolver, "command") {
		videoResolver = adapters.CommandVideoResolver{Name: cfg.VideoCommand, Args: cfg.VideoArgs}
	}

	mediaPreparer := app.MediaPreparer(adapters.StaticMediaPreparer{})
	if strings.EqualFold(cfg.MediaPreparer, "command") {
		mediaPreparer = adapters.CommandMediaPreparer{Name: cfg.MediaCommand, Args: cfg.MediaArgs}
	}

	summarizer := app.Summarizer(adapters.TemplateSummarizer{})
	if strings.EqualFold(cfg.SummaryMode, "http") {
		summarizer = adapters.FallbackSummarizer{
			Primary:  adapters.HTTPSummarizer{BaseURL: cfg.SummaryBaseURL, Model: cfg.SummaryModel, AuthToken: cfg.SummaryAuthToken},
			Fallback: adapters.TemplateSummarizer{},
		}
	}

	return app.NewService(app.ServiceDeps{
		Video:      videoResolver,
		Media:      mediaPreparer,
		ASR:        transcriber,
		Summarizer: summarizer,
	})
}
