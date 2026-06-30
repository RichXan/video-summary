package domain

import "time"

type Video struct {
	SourceURL   string `json:"source_url"`
	ResolvedURL string `json:"resolved_url,omitempty"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Duration    int    `json:"duration"`
}

type MediaAsset struct {
	VideoPath   string `json:"video_path,omitempty"`
	AudioPath   string `json:"audio_path"`
	ResolvedURL string `json:"resolved_url,omitempty"`
	Title       string `json:"title,omitempty"`
	Author      string `json:"author,omitempty"`
	Duration    int    `json:"duration,omitempty"`
}

type TranscriptSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type Summary struct {
	OneLine    string   `json:"one_line"`
	Outline    []string `json:"outline"`
	Quotes     []string `json:"quotes"`
	Viewpoints []string `json:"viewpoints"`
	Analysis   string   `json:"analysis"`
}

type SummaryResult struct {
	Video      Video               `json:"video"`
	Transcript []TranscriptSegment `json:"transcript"`
	Summary    Summary             `json:"summary"`
}

type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCanceled  JobStatus = "canceled"
)

type SummaryJob struct {
	ID         string         `json:"id"`
	SourceURL  string         `json:"source_url"`
	Status     JobStatus      `json:"status"`
	Result     *SummaryResult `json:"result,omitempty"`
	Error      string         `json:"error,omitempty"`
	Attempts   int            `json:"attempts"`
	CreatedAt  time.Time      `json:"created_at"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type JobStats struct {
	Queued    int `json:"queued"`
	Running   int `json:"running"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Canceled  int `json:"canceled"`
	Total     int `json:"total"`
}
