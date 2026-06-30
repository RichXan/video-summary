package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"video-summary-mvp/internal/domain"
)

type JobStore interface {
	CreateJob(ctx context.Context, job domain.SummaryJob) (domain.SummaryJob, error)
	GetJob(ctx context.Context, id string) (domain.SummaryJob, error)
	ClaimNextJob(ctx context.Context, workerID string) (domain.SummaryJob, bool, error)
	CompleteJob(ctx context.Context, job domain.SummaryJob, result domain.SummaryResult) (domain.SummaryJob, error)
	FailJob(ctx context.Context, job domain.SummaryJob, message string) (domain.SummaryJob, error)
}

type JobStatsStore interface {
	JobStats(ctx context.Context) (domain.JobStats, error)
}

type CreateJobInput struct {
	URL string
}

type JobService struct {
	store JobStore
}

func NewJobService(store JobStore) *JobService {
	return &JobService{store: store}
}

func (s *JobService) CreateJob(ctx context.Context, input CreateJobInput) (domain.SummaryJob, error) {
	if s.store == nil {
		return domain.SummaryJob{}, errors.New("job store is not configured")
	}
	url := strings.TrimSpace(input.URL)
	if url == "" {
		return domain.SummaryJob{}, errors.New("url is required")
	}
	now := time.Now().UTC()
	job := domain.SummaryJob{
		ID:        newJobID(),
		SourceURL: url,
		Status:    domain.JobStatusQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.store.CreateJob(ctx, job)
}

func (s *JobService) GetJob(ctx context.Context, id string) (domain.SummaryJob, error) {
	if s.store == nil {
		return domain.SummaryJob{}, errors.New("job store is not configured")
	}
	return s.store.GetJob(ctx, strings.TrimSpace(id))
}

func (s *JobService) JobStats(ctx context.Context) (domain.JobStats, error) {
	statsStore, ok := s.store.(JobStatsStore)
	if !ok {
		return domain.JobStats{}, nil
	}
	return statsStore.JobStats(ctx)
}

type WorkerConfig struct {
	Store        JobStore
	Service      *Service
	WorkerID     string
	PollInterval time.Duration
	JobTimeout   time.Duration
	Logger       *slog.Logger
}

type Worker struct {
	store        JobStore
	service      *Service
	workerID     string
	pollInterval time.Duration
	jobTimeout   time.Duration
	logger       *slog.Logger
}

func NewWorker(cfg WorkerConfig) *Worker {
	workerID := strings.TrimSpace(cfg.WorkerID)
	if workerID == "" {
		workerID = newJobID()
	}
	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	jobTimeout := cfg.JobTimeout
	if jobTimeout <= 0 {
		jobTimeout = 30 * time.Minute
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		store:        cfg.Store,
		service:      cfg.Service,
		workerID:     workerID,
		pollInterval: pollInterval,
		jobTimeout:   jobTimeout,
		logger:       logger,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if w.store == nil || w.service == nil {
		return errors.New("worker dependencies are not configured")
	}
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		_, err := w.RunOnce(ctx)
		if err != nil {
			w.logger.Error("worker iteration failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) (bool, error) {
	if w.store == nil || w.service == nil {
		return false, errors.New("worker dependencies are not configured")
	}
	job, ok, err := w.store.ClaimNextJob(ctx, w.workerID)
	if err != nil || !ok {
		return ok, err
	}

	jobCtx, cancel := context.WithTimeout(ctx, w.jobTimeout)
	defer cancel()

	w.logger.Info("job started", "job_id", job.ID, "worker_id", w.workerID)
	result, summarizeErr := w.service.Summarize(jobCtx, SummarizeInput{URL: job.SourceURL})
	if summarizeErr != nil {
		_, failErr := w.store.FailJob(ctx, job, summarizeErr.Error())
		if failErr != nil {
			return true, failErr
		}
		w.logger.Error("job failed", "job_id", job.ID, "error", summarizeErr)
		return true, nil
	}
	if _, err := w.store.CompleteJob(ctx, job, result); err != nil {
		return true, err
	}
	w.logger.Info("job completed", "job_id", job.ID)
	return true, nil
}

func newJobID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "job_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	}
	return "job_" + hex.EncodeToString(raw[:])
}
