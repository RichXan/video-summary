package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"video-summary-mvp/internal/domain"
)

type SummarizeRequest struct {
	URL string `json:"url"`
}

type CreateJobRequest struct {
	URL string `json:"url"`
}

type JobResponse struct {
	Job       domain.SummaryJob `json:"job"`
	StatusURL string            `json:"status_url"`
}

type SummaryService interface {
	Summarize(ctx context.Context, input SummarizeRequest) (domain.SummaryResult, error)
}

type JobService interface {
	CreateJob(ctx context.Context, input CreateJobRequest) (domain.SummaryJob, error)
	GetJob(ctx context.Context, id string) (domain.SummaryJob, error)
}

type JobStatsService interface {
	JobStats(ctx context.Context) (domain.JobStats, error)
}

type Handler struct {
	service SummaryService
	jobs    JobService
	mux     *http.ServeMux
}

func NewHandler(service SummaryService) *Handler {
	return NewHandlerWithJobs(service, nil)
}

func NewHandlerWithJobs(service SummaryService, jobs JobService) *Handler {
	h := &Handler{service: service, jobs: jobs, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /api/v1/videos/summarize", h.summarize)
	h.mux.HandleFunc("POST /api/v1/jobs", h.createJob)
	h.mux.HandleFunc("GET /api/v1/jobs/{id}", h.getJob)
	h.mux.HandleFunc("GET /healthz", h.healthz)
	h.mux.HandleFunc("GET /readyz", h.readyz)
	h.mux.HandleFunc("GET /metrics", h.metrics)
	return h
}

func (h *Handler) Routes() http.Handler {
	return h.mux
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) readyz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	var b strings.Builder
	b.WriteString("# HELP video_summary_up Service health.\n# TYPE video_summary_up gauge\nvideo_summary_up 1\n")
	if statsService, ok := h.jobs.(JobStatsService); ok {
		stats, err := statsService.JobStats(r.Context())
		if err == nil {
			b.WriteString("# HELP video_summary_jobs_total Jobs by status.\n# TYPE video_summary_jobs_total gauge\n")
			writeJobMetric(&b, "queued", stats.Queued)
			writeJobMetric(&b, "running", stats.Running)
			writeJobMetric(&b, "succeeded", stats.Succeeded)
			writeJobMetric(&b, "failed", stats.Failed)
			writeJobMetric(&b, "canceled", stats.Canceled)
			writeJobMetric(&b, "all", stats.Total)
		}
	}
	_, _ = w.Write([]byte(b.String()))
}

func writeJobMetric(b *strings.Builder, status string, value int) {
	_, _ = fmt.Fprintf(b, "video_summary_jobs_total{status=%q} %d\n", status, value)
}

func (h *Handler) summarize(w http.ResponseWriter, r *http.Request) {
	var req SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		writeError(w, http.StatusBadRequest, "missing_url", "url is required")
		return
	}
	if h.service == nil {
		writeError(w, http.StatusInternalServerError, "service_unavailable", "summary service is not configured")
		return
	}

	result, err := h.service.Summarize(r.Context(), req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, context.Canceled) {
			status = http.StatusRequestTimeout
		}
		writeError(w, status, "summary_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) createJob(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		writeError(w, http.StatusBadRequest, "missing_url", "url is required")
		return
	}
	if h.jobs == nil {
		writeError(w, http.StatusServiceUnavailable, "job_service_unavailable", "job service is not configured")
		return
	}
	job, err := h.jobs.CreateJob(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "job_create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, JobResponse{
		Job:       job,
		StatusURL: "/api/v1/jobs/" + job.ID,
	})
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	if h.jobs == nil {
		writeError(w, http.StatusServiceUnavailable, "job_service_unavailable", "job service is not configured")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_job_id", "job id is required")
		return
	}
	job, err := h.jobs.GetJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "job_not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}
