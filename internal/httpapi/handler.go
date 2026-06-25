package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"video-summary-mvp/internal/domain"
)

type SummarizeRequest struct {
	URL string `json:"url"`
}

type SummaryService interface {
	Summarize(ctx context.Context, input SummarizeRequest) (domain.SummaryResult, error)
}

type Handler struct {
	service SummaryService
	mux     *http.ServeMux
}

func NewHandler(service SummaryService) *Handler {
	h := &Handler{service: service, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /api/v1/videos/summarize", h.summarize)
	h.mux.HandleFunc("GET /healthz", h.healthz)
	return h
}

func (h *Handler) Routes() http.Handler {
	return h.mux
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
