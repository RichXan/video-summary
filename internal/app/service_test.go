package app

import (
	"context"
	"errors"
	"testing"

	"video-summary-mvp/internal/domain"
)

func TestServiceSummarizeRunsPipeline(t *testing.T) {
	t.Parallel()

	service := NewService(ServiceDeps{
		Video: fakeVideoResolver{},
		Media: fakeMediaPreparer{},
		ASR:   fakeTranscriber{},
		Summarizer: fakeSummarizer{
			result: domain.Summary{
				OneLine: "A concise summary.",
				Outline: []string{"Point one", "Point two"},
				Quotes:  []string{"Important line"},
			},
		},
	})

	result, err := service.Summarize(context.Background(), SummarizeInput{URL: "https://v.douyin.com/demo/"})
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	if result.Video.Title != "Demo video" {
		t.Fatalf("title = %q, want Demo video", result.Video.Title)
	}
	if len(result.Transcript) != 1 || result.Transcript[0].Text != "hello world" {
		t.Fatalf("unexpected transcript: %#v", result.Transcript)
	}
	if result.Summary.OneLine != "A concise summary." {
		t.Fatalf("summary = %q, want concise summary", result.Summary.OneLine)
	}
}

func TestServiceSummarizeAppliesMediaMetadata(t *testing.T) {
	t.Parallel()

	service := NewService(ServiceDeps{
		Video: fakeVideoResolver{},
		Media: fakeMediaPreparerWithMetadata{},
		ASR:   fakeTranscriber{},
		Summarizer: fakeSummarizer{
			result: domain.Summary{OneLine: "summary"},
		},
	})

	result, err := service.Summarize(context.Background(), SummarizeInput{URL: "https://v.douyin.com/demo/"})
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	if result.Video.Title != "Downloaded title" {
		t.Fatalf("title = %q, want media metadata title", result.Video.Title)
	}
	if result.Video.Author != "Downloaded author" {
		t.Fatalf("author = %q, want media metadata author", result.Video.Author)
	}
	if result.Video.Duration != 503 {
		t.Fatalf("duration = %d, want media metadata duration", result.Video.Duration)
	}
}

func TestServiceSummarizeReturnsDependencyError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("resolver failed")
	service := NewService(ServiceDeps{
		Video:      failingVideoResolver{err: wantErr},
		Media:      fakeMediaPreparer{},
		ASR:        fakeTranscriber{},
		Summarizer: fakeSummarizer{},
	})

	_, err := service.Summarize(context.Background(), SummarizeInput{URL: "https://v.douyin.com/demo/"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

type fakeVideoResolver struct{}

func (fakeVideoResolver) Resolve(ctx context.Context, rawURL string) (domain.Video, error) {
	return domain.Video{
		SourceURL: rawURL,
		Title:     "Demo video",
		Author:    "Demo author",
		Duration:  42,
	}, nil
}

type failingVideoResolver struct {
	err error
}

func (r failingVideoResolver) Resolve(ctx context.Context, rawURL string) (domain.Video, error) {
	return domain.Video{}, r.err
}

type fakeMediaPreparer struct{}

func (fakeMediaPreparer) Prepare(ctx context.Context, video domain.Video) (domain.MediaAsset, error) {
	return domain.MediaAsset{AudioPath: "demo.wav"}, nil
}

type fakeMediaPreparerWithMetadata struct{}

func (fakeMediaPreparerWithMetadata) Prepare(ctx context.Context, video domain.Video) (domain.MediaAsset, error) {
	return domain.MediaAsset{
		AudioPath: "demo.wav",
		Title:     "Downloaded title",
		Author:    "Downloaded author",
		Duration:  503,
	}, nil
}

type fakeTranscriber struct{}

func (fakeTranscriber) Transcribe(ctx context.Context, media domain.MediaAsset) ([]domain.TranscriptSegment, error) {
	return []domain.TranscriptSegment{{Start: 0, End: 1.2, Text: "hello world"}}, nil
}

type fakeSummarizer struct {
	result domain.Summary
}

func (s fakeSummarizer) Summarize(ctx context.Context, video domain.Video, transcript []domain.TranscriptSegment) (domain.Summary, error) {
	return s.result, nil
}
