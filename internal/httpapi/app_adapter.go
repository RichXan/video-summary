package httpapi

import (
	"context"

	"video-summary-mvp/internal/app"
	"video-summary-mvp/internal/domain"
)

type AppAdapter struct {
	Service *app.Service
}

func (a AppAdapter) Summarize(ctx context.Context, input SummarizeRequest) (domain.SummaryResult, error) {
	return a.Service.Summarize(ctx, app.SummarizeInput{URL: input.URL})
}
