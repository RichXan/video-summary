package adapters

import (
	"context"
	"strings"
	"testing"

	"video-summary-mvp/internal/domain"
)

func TestTemplateSummarizerBuildsStructuredSummary(t *testing.T) {
	t.Parallel()

	summarizer := TemplateSummarizer{}
	transcript := []domain.TranscriptSegment{
		{Text: "第一句讲经济模型。"},
		{Text: "第二句讲通货膨胀。"},
		{Text: "第三句讲普通人的策略。"},
	}

	result, err := summarizer.Summarize(context.Background(), domain.Video{Title: "Demo"}, transcript)
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	if !strings.Contains(result.OneLine, "Demo") {
		t.Fatalf("one line summary = %q, want title included", result.OneLine)
	}
	if len(result.Outline) == 0 {
		t.Fatal("outline should not be empty")
	}
	if len(result.Quotes) == 0 {
		t.Fatal("quotes should not be empty")
	}
}

func TestTemplateSummarizerUsesTranscriptContentInOutline(t *testing.T) {
	t.Parallel()

	summarizer := TemplateSummarizer{}
	transcript := []domain.TranscriptSegment{
		{Text: "Dollar dominance depends on oil pricing."},
		{Text: "SWIFT and treasury markets form the payment pipe."},
		{Text: "Central banks are buying gold as a hedge."},
	}

	result, err := summarizer.Summarize(context.Background(), domain.Video{Title: "Dollar system"}, transcript)
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	joined := strings.Join(append(result.Outline, result.Analysis), "\n")
	if !strings.Contains(joined, "oil pricing") {
		t.Fatalf("summary should include transcript content, got: %#v", result)
	}
	if strings.Contains(strings.ToLower(joined), "smoke") {
		t.Fatalf("summary should not describe itself as a smoke test: %#v", result)
	}
}
