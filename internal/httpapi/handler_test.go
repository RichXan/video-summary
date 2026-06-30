package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"video-summary-mvp/internal/domain"
)

func TestHandlerSummarizeReturnsResult(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeSummaryService{
		result: domain.SummaryResult{
			Video: domain.Video{SourceURL: "https://v.douyin.com/demo/", Title: "Demo"},
			Transcript: []domain.TranscriptSegment{
				{Start: 0, End: 1, Text: "hello"},
			},
			Summary: domain.Summary{OneLine: "summary"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/videos/summarize", bytes.NewBufferString(`{"url":"https://v.douyin.com/demo/"}`))
	res := httptest.NewRecorder()

	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body domain.SummaryResult
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Video.Title != "Demo" || body.Summary.OneLine != "summary" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestHandlerSummarizeRejectsMissingURL(t *testing.T) {
	t.Parallel()

	handler := NewHandler(fakeSummaryService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/videos/summarize", bytes.NewBufferString(`{"url":""}`))
	res := httptest.NewRecorder()

	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestHandlerCreateJobReturnsAcceptedJob(t *testing.T) {
	t.Parallel()

	handler := NewHandlerWithJobs(fakeSummaryService{}, fakeJobService{
		job: domain.SummaryJob{ID: "job_123", SourceURL: "https://v.douyin.com/demo/", Status: domain.JobStatusQueued},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(`{"url":"https://v.douyin.com/demo/"}`))
	res := httptest.NewRecorder()

	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusAccepted)
	}
	var body JobResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Job.ID != "job_123" || body.StatusURL != "/api/v1/jobs/job_123" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestHandlerGetJobReturnsStoredJob(t *testing.T) {
	t.Parallel()

	handler := NewHandlerWithJobs(fakeSummaryService{}, fakeJobService{
		job: domain.SummaryJob{
			ID:        "job_123",
			SourceURL: "https://v.douyin.com/demo/",
			Status:    domain.JobStatusSucceeded,
			Result: &domain.SummaryResult{
				Video:   domain.Video{Title: "Demo"},
				Summary: domain.Summary{OneLine: "summary"},
			},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job_123", nil)
	res := httptest.NewRecorder()

	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body domain.SummaryJob
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != domain.JobStatusSucceeded || body.Result.Summary.OneLine != "summary" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestHandlerMetricsIncludesJobStats(t *testing.T) {
	t.Parallel()

	handler := NewHandlerWithJobs(fakeSummaryService{}, fakeJobService{
		stats: domain.JobStats{Queued: 2, Running: 1, Succeeded: 3, Failed: 4, Total: 10},
	})
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	res := httptest.NewRecorder()

	handler.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	body := res.Body.String()
	if !bytes.Contains([]byte(body), []byte(`video_summary_jobs_total{status="queued"} 2`)) {
		t.Fatalf("metrics did not include queued count: %s", body)
	}
}

type fakeSummaryService struct {
	result domain.SummaryResult
}

func (s fakeSummaryService) Summarize(ctx context.Context, input SummarizeRequest) (domain.SummaryResult, error) {
	return s.result, nil
}

type fakeJobService struct {
	job   domain.SummaryJob
	stats domain.JobStats
}

func (s fakeJobService) CreateJob(ctx context.Context, input CreateJobRequest) (domain.SummaryJob, error) {
	return s.job, nil
}

func (s fakeJobService) GetJob(ctx context.Context, id string) (domain.SummaryJob, error) {
	return s.job, nil
}

func (s fakeJobService) JobStats(ctx context.Context) (domain.JobStats, error) {
	return s.stats, nil
}
