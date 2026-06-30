package adapters

import (
	"context"
	"os"
	"testing"
	"time"

	"video-summary-mvp/internal/domain"
)

func TestPostgresJobStoreLifecycle(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx := context.Background()
	store, err := OpenPostgresJobStore(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPostgresJobStore: %v", err)
	}
	defer func() { _ = store.Close() }()
	resetSummaryJobs(t, store)

	now := time.Now().UTC()
	job, err := store.CreateJob(ctx, domain.SummaryJob{
		ID:        "job_integration_lifecycle",
		SourceURL: "https://v.douyin.com/demo/",
		Status:    domain.JobStatusQueued,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if job.Status != domain.JobStatusQueued {
		t.Fatalf("created status = %q", job.Status)
	}

	claimed, ok, err := store.ClaimNextJob(ctx, "worker-1")
	if err != nil {
		t.Fatalf("ClaimNextJob: %v", err)
	}
	if !ok || claimed.ID != job.ID || claimed.Status != domain.JobStatusRunning || claimed.Attempts != 1 {
		t.Fatalf("unexpected claimed job: ok=%v job=%#v", ok, claimed)
	}

	result := domain.SummaryResult{Video: domain.Video{Title: "Demo"}, Summary: domain.Summary{OneLine: "summary"}}
	completed, err := store.CompleteJob(ctx, claimed, result)
	if err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}
	if completed.Status != domain.JobStatusSucceeded || completed.Result.Summary.OneLine != "summary" {
		t.Fatalf("unexpected completed job: %#v", completed)
	}

	stats, err := store.JobStats(ctx)
	if err != nil {
		t.Fatalf("JobStats: %v", err)
	}
	if stats.Succeeded != 1 || stats.Total != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestPostgresJobStoreRequeuesStaleRunningJobs(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN is not set")
	}

	ctx := context.Background()
	store, err := OpenPostgresJobStore(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPostgresJobStore: %v", err)
	}
	defer func() { _ = store.Close() }()
	resetSummaryJobs(t, store)

	staleStarted := time.Now().UTC().Add(-2 * time.Hour)
	_, err = store.db.ExecContext(ctx, `
INSERT INTO summary_jobs (id, source_url, status, error, attempts, created_at, started_at, updated_at)
VALUES ($1, $2, $3, '', 1, NOW() - INTERVAL '2 hours', $4, NOW() - INTERVAL '2 hours')
`, "job_stale_running", "https://v.douyin.com/demo/", string(domain.JobStatusRunning), staleStarted)
	if err != nil {
		t.Fatalf("insert stale job: %v", err)
	}

	requeued, err := store.RequeueStaleRunningJobs(ctx, 30*time.Minute)
	if err != nil {
		t.Fatalf("RequeueStaleRunningJobs: %v", err)
	}
	if requeued != 1 {
		t.Fatalf("requeued = %d, want 1", requeued)
	}

	claimed, ok, err := store.ClaimNextJob(ctx, "worker-1")
	if err != nil {
		t.Fatalf("ClaimNextJob: %v", err)
	}
	if !ok || claimed.ID != "job_stale_running" || claimed.Status != domain.JobStatusRunning {
		t.Fatalf("unexpected claimed stale job: ok=%v job=%#v", ok, claimed)
	}
}

func resetSummaryJobs(t *testing.T, store *PostgresJobStore) {
	t.Helper()
	if _, err := store.db.ExecContext(context.Background(), `TRUNCATE summary_jobs`); err != nil {
		t.Fatalf("truncate summary_jobs: %v", err)
	}
}
