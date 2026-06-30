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

type AppJobAdapter struct {
	Service *app.JobService
}

func (a AppJobAdapter) CreateJob(ctx context.Context, input CreateJobRequest) (domain.SummaryJob, error) {
	return a.Service.CreateJob(ctx, app.CreateJobInput{URL: input.URL})
}

func (a AppJobAdapter) GetJob(ctx context.Context, id string) (domain.SummaryJob, error) {
	return a.Service.GetJob(ctx, id)
}

func (a AppJobAdapter) JobStats(ctx context.Context) (domain.JobStats, error) {
	return a.Service.JobStats(ctx)
}
