package adapters

import (
	"context"
	"fmt"
	"strings"

	"video-summary-mvp/internal/domain"
)

type StaticVideoResolver struct{}

func (StaticVideoResolver) Resolve(ctx context.Context, rawURL string) (domain.Video, error) {
	return domain.Video{
		SourceURL: rawURL,
		Title:     "Untitled short video",
		Author:    "unknown",
	}, nil
}

type StaticMediaPreparer struct{}

func (StaticMediaPreparer) Prepare(ctx context.Context, video domain.Video) (domain.MediaAsset, error) {
	return domain.MediaAsset{AudioPath: video.SourceURL}, nil
}

type StaticTranscriber struct{}

func (StaticTranscriber) Transcribe(ctx context.Context, media domain.MediaAsset) ([]domain.TranscriptSegment, error) {
	return []domain.TranscriptSegment{
		{Start: 0, End: 8, Text: "This is a demo transcript segment. In production it can come from whisper.cpp, FunASR, or faster-whisper."},
		{Start: 8, End: 16, Text: "The service turns transcript text into a summary, outline, quotes, and viewpoint analysis."},
	}, nil
}

type TemplateSummarizer struct{}

func (TemplateSummarizer) Summarize(ctx context.Context, video domain.Video, transcript []domain.TranscriptSegment) (domain.Summary, error) {
	text := joinTranscript(transcript)
	first := firstNonEmpty(transcript)
	if first == "" {
		first = "No transcript is available."
	}

	title := strings.TrimSpace(video.Title)
	if title == "" {
		title = "This video"
	}

	return domain.Summary{
		OneLine: fmt.Sprintf("%s: %s", title, trimRunes(first, 96)),
		Outline: extractivePoints(transcript, 5),
		Quotes:  extractivePoints(transcript, 3),
		Viewpoints: []string{
			"Key claims are extracted directly from the transcript.",
			"Use an LLM summarizer for deeper synthesis, but this result is grounded in the real ASR output.",
		},
		Analysis: fmt.Sprintf("Extractive summary built from %d transcript segments and about %d runes.", len(transcript), len([]rune(text))),
	}, nil
}

func joinTranscript(transcript []domain.TranscriptSegment) string {
	parts := make([]string, 0, len(transcript))
	for _, segment := range transcript {
		if text := strings.TrimSpace(segment.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func firstNonEmpty(transcript []domain.TranscriptSegment) string {
	for _, segment := range transcript {
		if text := strings.TrimSpace(segment.Text); text != "" {
			return text
		}
	}
	return ""
}

func trimRunes(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "..."
}

func extractivePoints(transcript []domain.TranscriptSegment, max int) []string {
	lines := make([]string, 0, len(transcript))
	for _, segment := range transcript {
		text := strings.TrimSpace(segment.Text)
		if text != "" {
			lines = append(lines, text)
		}
	}
	if len(lines) == 0 || max <= 0 {
		return []string{"No transcript content was available."}
	}
	if len(lines) <= max {
		points := make([]string, len(lines))
		for i, line := range lines {
			points[i] = trimRunes(line, 120)
		}
		return points
	}

	points := make([]string, 0, max)
	seen := map[int]bool{}
	for i := 0; i < max; i++ {
		idx := i * (len(lines) - 1) / (max - 1)
		if seen[idx] {
			continue
		}
		seen[idx] = true
		points = append(points, trimRunes(lines[idx], 120))
	}
	return points
}
