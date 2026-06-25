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

type fakeSummaryService struct {
	result domain.SummaryResult
}

func (s fakeSummaryService) Summarize(ctx context.Context, input SummarizeRequest) (domain.SummaryResult, error) {
	return s.result, nil
}
