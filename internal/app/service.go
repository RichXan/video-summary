package app

import (
	"context"
	"errors"
	"strings"

	"video-summary-mvp/internal/domain"
)

type VideoResolver interface {
	Resolve(ctx context.Context, rawURL string) (domain.Video, error)
}

type MediaPreparer interface {
	Prepare(ctx context.Context, video domain.Video) (domain.MediaAsset, error)
}

type Transcriber interface {
	Transcribe(ctx context.Context, media domain.MediaAsset) ([]domain.TranscriptSegment, error)
}

type Summarizer interface {
	Summarize(ctx context.Context, video domain.Video, transcript []domain.TranscriptSegment) (domain.Summary, error)
}

type ServiceDeps struct {
	Video      VideoResolver
	Media      MediaPreparer
	ASR        Transcriber
	Summarizer Summarizer
}

type Service struct {
	video      VideoResolver
	media      MediaPreparer
	asr        Transcriber
	summarizer Summarizer
}

type SummarizeInput struct {
	URL string
}

func NewService(deps ServiceDeps) *Service {
	return &Service{
		video:      deps.Video,
		media:      deps.Media,
		asr:        deps.ASR,
		summarizer: deps.Summarizer,
	}
}

func (s *Service) Summarize(ctx context.Context, input SummarizeInput) (domain.SummaryResult, error) {
	if strings.TrimSpace(input.URL) == "" {
		return domain.SummaryResult{}, errors.New("url is required")
	}
	if s.video == nil || s.media == nil || s.asr == nil || s.summarizer == nil {
		return domain.SummaryResult{}, errors.New("service dependencies are not configured")
	}

	video, err := s.video.Resolve(ctx, input.URL)
	if err != nil {
		return domain.SummaryResult{}, err
	}

	media, err := s.media.Prepare(ctx, video)
	if err != nil {
		return domain.SummaryResult{}, err
	}

	transcript, err := s.asr.Transcribe(ctx, media)
	if err != nil {
		return domain.SummaryResult{}, err
	}

	summary, err := s.summarizer.Summarize(ctx, video, transcript)
	if err != nil {
		return domain.SummaryResult{}, err
	}

	return domain.SummaryResult{
		Video:      video,
		Transcript: transcript,
		Summary:    summary,
	}, nil
}
