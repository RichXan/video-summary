package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"video-summary-mvp/internal/domain"
)

func TestHTTPSummarizerCallsOpenAICompatibleChatEndpoint(t *testing.T) {
	t.Parallel()

	var sawTranscript bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		payload, _ := json.Marshal(body)
		sawTranscript = strings.Contains(string(payload), "important transcript")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": `{"one_line":"short","outline":["a"],"quotes":["q"],"viewpoints":["v"],"analysis":"analysis"}`,
					},
				},
			},
		})
	}))
	defer server.Close()

	summarizer := HTTPSummarizer{BaseURL: server.URL, Model: "qwen"}
	summary, err := summarizer.Summarize(context.Background(), domain.Video{Title: "Demo"}, []domain.TranscriptSegment{{Text: "important transcript"}})
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	if !sawTranscript {
		t.Fatal("request did not include transcript")
	}
	if summary.OneLine != "short" || summary.Analysis != "analysis" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestHTTPSummarizerFallsBackToPlainContent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "plain summary"}},
			},
		})
	}))
	defer server.Close()

	summarizer := HTTPSummarizer{BaseURL: server.URL}
	summary, err := summarizer.Summarize(context.Background(), domain.Video{}, []domain.TranscriptSegment{{Text: "x"}})
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	if summary.OneLine != "plain summary" {
		t.Fatalf("one line = %q", summary.OneLine)
	}
}
