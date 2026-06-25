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
		OneLine: fmt.Sprintf("%s is summarized from transcript text. Core signal: %s", title, trimRunes(first, 80)),
		Outline: []string{
			"Video metadata and transcript input are available to the pipeline.",
			"The transcript is condensed into main points and information hierarchy.",
			"The template summarizer can later be replaced by OpenAI, Ollama, Qwen, or DeepSeek.",
		},
		Quotes: []string{
			trimRunes(first, 80),
		},
		Viewpoints: []string{
			"The MVP validates the orchestration path before replacing ASR and LLM components.",
			"Structured summaries should be grounded in full transcripts, not only title or captions.",
		},
		Analysis: fmt.Sprintf("Transcript length is about %d runes. The template summarizer is for smoke tests; production should use an LLM adapter.", len([]rune(text))),
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
