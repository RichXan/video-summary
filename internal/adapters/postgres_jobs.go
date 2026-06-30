package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"video-summary-mvp/internal/domain"
)

type PostgresJobStore struct {
	db *sql.DB
}

func OpenPostgresJobStore(ctx context.Context, dsn string) (*PostgresJobStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &PostgresJobStore{db: db}
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresJobStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *PostgresJobStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS summary_jobs (
	id TEXT PRIMARY KEY,
	source_url TEXT NOT NULL,
	status TEXT NOT NULL,
	result JSONB,
	error TEXT NOT NULL DEFAULT '',
	attempts INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL,
	started_at TIMESTAMPTZ,
	finished_at TIMESTAMPTZ,
	updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS summary_jobs_status_created_idx ON summary_jobs (status, created_at);
`)
	return err
}

func (s *PostgresJobStore) CreateJob(ctx context.Context, job domain.SummaryJob) (domain.SummaryJob, error) {
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = job.CreatedAt
	}
	if job.Status == "" {
		job.Status = domain.JobStatusQueued
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO summary_jobs (id, source_url, status, error, attempts, created_at, updated_at)
VALUES ($1, $2, $3, '', $4, $5, $6)
`, job.ID, job.SourceURL, string(job.Status), job.Attempts, job.CreatedAt, job.UpdatedAt)
	if err != nil {
		return domain.SummaryJob{}, err
	}
	return job, nil
}

func (s *PostgresJobStore) GetJob(ctx context.Context, id string) (domain.SummaryJob, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, source_url, status, result, error, attempts, created_at, started_at, finished_at, updated_at
FROM summary_jobs
WHERE id = $1
`, id)
	return scanJob(row)
}

func (s *PostgresJobStore) ClaimNextJob(ctx context.Context, workerID string) (domain.SummaryJob, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.SummaryJob{}, false, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `
UPDATE summary_jobs
SET status = $1,
	attempts = attempts + 1,
	started_at = COALESCE(started_at, NOW()),
	updated_at = NOW()
WHERE id = (
	SELECT id
	FROM summary_jobs
	WHERE status = $2
	ORDER BY created_at
	FOR UPDATE SKIP LOCKED
	LIMIT 1
)
RETURNING id, source_url, status, result, error, attempts, created_at, started_at, finished_at, updated_at
`, string(domain.JobStatusRunning), string(domain.JobStatusQueued))
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SummaryJob{}, false, nil
	}
	if err != nil {
		return domain.SummaryJob{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.SummaryJob{}, false, err
	}
	return job, true, nil
}

func (s *PostgresJobStore) RequeueStaleRunningJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	if olderThan <= 0 {
		return 0, nil
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE summary_jobs
SET status = $1,
	started_at = NULL,
	updated_at = NOW(),
	error = ''
WHERE status = $2
	AND updated_at < NOW() - ($3 * INTERVAL '1 second')
`, string(domain.JobStatusQueued), string(domain.JobStatusRunning), int(olderThan.Seconds()))
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rows), nil
}

func (s *PostgresJobStore) CompleteJob(ctx context.Context, job domain.SummaryJob, result domain.SummaryResult) (domain.SummaryJob, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return domain.SummaryJob{}, err
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE summary_jobs
SET status = $1,
	result = $2,
	error = '',
	finished_at = NOW(),
	updated_at = NOW()
WHERE id = $3
RETURNING id, source_url, status, result, error, attempts, created_at, started_at, finished_at, updated_at
`, string(domain.JobStatusSucceeded), payload, job.ID)
	return scanJob(row)
}

func (s *PostgresJobStore) FailJob(ctx context.Context, job domain.SummaryJob, message string) (domain.SummaryJob, error) {
	row := s.db.QueryRowContext(ctx, `
UPDATE summary_jobs
SET status = $1,
	error = $2,
	finished_at = NOW(),
	updated_at = NOW()
WHERE id = $3
RETURNING id, source_url, status, result, error, attempts, created_at, started_at, finished_at, updated_at
`, string(domain.JobStatusFailed), message, job.ID)
	return scanJob(row)
}

func (s *PostgresJobStore) JobStats(ctx context.Context) (domain.JobStats, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM summary_jobs GROUP BY status`)
	if err != nil {
		return domain.JobStats{}, err
	}
	defer rows.Close()

	var stats domain.JobStats
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return domain.JobStats{}, err
		}
		stats.Total += count
		switch domain.JobStatus(status) {
		case domain.JobStatusQueued:
			stats.Queued = count
		case domain.JobStatusRunning:
			stats.Running = count
		case domain.JobStatusSucceeded:
			stats.Succeeded = count
		case domain.JobStatusFailed:
			stats.Failed = count
		case domain.JobStatusCanceled:
			stats.Canceled = count
		}
	}
	return stats, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(row rowScanner) (domain.SummaryJob, error) {
	var job domain.SummaryJob
	var status string
	var resultText sql.NullString
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	if err := row.Scan(
		&job.ID,
		&job.SourceURL,
		&status,
		&resultText,
		&job.Error,
		&job.Attempts,
		&job.CreatedAt,
		&startedAt,
		&finishedAt,
		&job.UpdatedAt,
	); err != nil {
		return domain.SummaryJob{}, err
	}
	job.Status = domain.JobStatus(status)
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		job.FinishedAt = &finishedAt.Time
	}
	if resultText.Valid && resultText.String != "" {
		var result domain.SummaryResult
		if err := json.Unmarshal([]byte(resultText.String), &result); err != nil {
			return domain.SummaryJob{}, err
		}
		job.Result = &result
	}
	return job, nil
}
