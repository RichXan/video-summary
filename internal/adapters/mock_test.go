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
