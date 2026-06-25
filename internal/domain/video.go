package domain

type Video struct {
	SourceURL  string `json:"source_url"`
	ResolvedURL string `json:"resolved_url,omitempty"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	Duration   int    `json:"duration"`
}

type MediaAsset struct {
	VideoPath string `json:"video_path,omitempty"`
	AudioPath string `json:"audio_path"`
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
