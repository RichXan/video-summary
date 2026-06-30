package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"video-summary-mvp/internal/domain"
)

var errTestSummaryFailed = errors.New("summary failed")

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

func TestHTTPSummarizerSendsBearerTokenAndModel(t *testing.T) {
	t.Parallel()

	var auth string
	var model string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		var body chatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		model = body.Model
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"one_line":"ok","outline":[],"quotes":[],"viewpoints":[],"analysis":"ok"}`}},
			},
		})
	}))
	defer server.Close()

	summarizer := HTTPSummarizer{BaseURL: server.URL, Model: "chatgpt5.5", AuthToken: "secret-token"}
	if _, err := summarizer.Summarize(context.Background(), domain.Video{}, []domain.TranscriptSegment{{Text: "x"}}); err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	if auth != "Bearer secret-token" {
		t.Fatalf("Authorization = %q", auth)
	}
	if model != "chatgpt5.5" {
		t.Fatalf("model = %q", model)
	}
}

func TestHTTPSummarizerParsesFencedJSONContent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": "```json\n{\"one_line\":\"one\",\"outline\":[\"point\"],\"quotes\":[\"quote\"],\"viewpoints\":[\"view\"],\"analysis\":\"analysis\"}\n```",
					},
				},
			},
		})
	}))
	defer server.Close()

	summarizer := HTTPSummarizer{BaseURL: server.URL}
	summary, err := summarizer.Summarize(context.Background(), domain.Video{}, []domain.TranscriptSegment{{Text: "x"}})
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	if summary.OneLine != "one" || len(summary.Outline) != 1 || summary.Analysis != "analysis" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestBuildSummaryPromptCapsVeryLongTranscript(t *testing.T) {
	t.Parallel()

	prompt := buildSummaryPrompt(domain.Video{Title: "Long"}, []domain.TranscriptSegment{
		{Text: strings.Repeat("long transcript ", 12000)},
	})

	if len([]rune(prompt)) > maxSummaryPromptRunes+500 {
		t.Fatalf("prompt runes = %d, want capped near %d", len([]rune(prompt)), maxSummaryPromptRunes)
	}
	if !strings.Contains(prompt, "Transcript truncated") {
		t.Fatalf("prompt did not mention truncation")
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

func TestFallbackSummarizerUsesFallbackWhenPrimaryFails(t *testing.T) {
	t.Parallel()

	primary := failingSummarizer{}
	fallback := TemplateSummarizer{}
	summarizer := FallbackSummarizer{Primary: primary, Fallback: fallback}

	summary, err := summarizer.Summarize(context.Background(), domain.Video{Title: "Demo"}, []domain.TranscriptSegment{{Text: "real transcript"}})
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}
	if !strings.Contains(summary.OneLine, "real transcript") {
		t.Fatalf("fallback summary did not use transcript: %#v", summary)
	}
}

type failingSummarizer struct{}

func (failingSummarizer) Summarize(context.Context, domain.Video, []domain.TranscriptSegment) (domain.Summary, error) {
	return domain.Summary{}, errTestSummaryFailed
}
