package app

import (
	"context"
	"errors"
	"testing"

	"video-summary-mvp/internal/domain"
)

func TestJobServiceCreatesQueuedJob(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{}
	service := NewJobService(store)

	job, err := service.CreateJob(context.Background(), CreateJobInput{URL: "https://v.douyin.com/demo/"})
	if err != nil {
		t.Fatalf("CreateJob returned error: %v", err)
	}

	if job.ID == "" || job.Status != domain.JobStatusQueued {
		t.Fatalf("unexpected job: %#v", job)
	}
	if store.created.SourceURL != "https://v.douyin.com/demo/" {
		t.Fatalf("stored url = %q", store.created.SourceURL)
	}
}

func TestWorkerRunOnceCompletesClaimedJob(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		claim: domain.SummaryJob{ID: "job_1", SourceURL: "https://v.douyin.com/demo/", Status: domain.JobStatusRunning},
	}
	service := NewService(ServiceDeps{
		Video:      fakeVideoResolver{},
		Media:      fakeMediaPreparer{},
		ASR:        fakeTranscriber{},
		Summarizer: fakeSummarizer{result: domain.Summary{OneLine: "summary"}},
	})
	worker := NewWorker(WorkerConfig{Store: store, Service: service})

	ran, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}
	if !ran {
		t.Fatal("RunOnce did not claim a job")
	}
	if store.completed.ID != "job_1" || store.completedResult.Summary.OneLine != "summary" {
		t.Fatalf("job not completed: %#v %#v", store.completed, store.completedResult)
	}
}

func TestWorkerRunOnceMarksJobFailed(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		claim: domain.SummaryJob{ID: "job_1", SourceURL: "https://v.douyin.com/demo/", Status: domain.JobStatusRunning},
	}
	service := NewService(ServiceDeps{
		Video:      failingVideoResolver{err: errors.New("resolver failed")},
		Media:      fakeMediaPreparer{},
		ASR:        fakeTranscriber{},
		Summarizer: fakeSummarizer{},
	})
	worker := NewWorker(WorkerConfig{Store: store, Service: service})

	ran, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}
	if !ran {
		t.Fatal("RunOnce did not claim a job")
	}
	if store.failed.ID != "job_1" || store.failedError == "" {
		t.Fatalf("job not failed: %#v error=%q", store.failed, store.failedError)
	}
}

type fakeJobStore struct {
	created         domain.SummaryJob
	claim           domain.SummaryJob
	completed       domain.SummaryJob
	completedResult domain.SummaryResult
	failed          domain.SummaryJob
	failedError     string
}

func (s *fakeJobStore) CreateJob(ctx context.Context, job domain.SummaryJob) (domain.SummaryJob, error) {
	s.created = job
	return job, nil
}

func (s *fakeJobStore) GetJob(ctx context.Context, id string) (domain.SummaryJob, error) {
	return s.created, nil
}

func (s *fakeJobStore) ClaimNextJob(ctx context.Context, workerID string) (domain.SummaryJob, bool, error) {
	if s.claim.ID == "" {
		return domain.SummaryJob{}, false, nil
	}
	return s.claim, true, nil
}

func (s *fakeJobStore) CompleteJob(ctx context.Context, job domain.SummaryJob, result domain.SummaryResult) (domain.SummaryJob, error) {
	s.completed = job
	s.completedResult = result
	job.Status = domain.JobStatusSucceeded
	job.Result = &result
	return job, nil
}

func (s *fakeJobStore) FailJob(ctx context.Context, job domain.SummaryJob, message string) (domain.SummaryJob, error) {
	s.failed = job
	s.failedError = message
	job.Status = domain.JobStatusFailed
	job.Error = message
	return job, nil
}
