package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"video-summary-mvp/internal/domain"
)

type HTTPSummarizer struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (s HTTPSummarizer) Summarize(ctx context.Context, video domain.Video, transcript []domain.TranscriptSegment) (domain.Summary, error) {
	if strings.TrimSpace(s.BaseURL) == "" {
		return domain.Summary{}, errors.New("summary base url is not configured")
	}

	payload := chatRequest{
		Model:       firstNonEmptyString(s.Model, "qwen"),
		Temperature: 0.2,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: "You summarize transcripts. Return compact JSON with keys: one_line, outline, quotes, viewpoints, analysis.",
			},
			{
				Role:    "user",
				Content: buildSummaryPrompt(video, transcript),
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.Summary{}, err
	}

	endpoint := strings.TrimRight(s.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return domain.Summary{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	res, err := client.Do(req)
	if err != nil {
		return domain.Summary{}, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return domain.Summary{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return domain.Summary{}, errors.New(strings.TrimSpace(string(resBody)))
	}

	var parsed chatResponse
	if err := json.Unmarshal(resBody, &parsed); err != nil {
		return domain.Summary{}, err
	}
	if len(parsed.Choices) == 0 {
		return domain.Summary{}, errors.New("summary response did not include choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	var summary domain.Summary
	if err := json.Unmarshal([]byte(content), &summary); err == nil && strings.TrimSpace(summary.OneLine) != "" {
		return summary, nil
	}
	if content == "" {
		return domain.Summary{}, errors.New("summary response content is empty")
	}
	return domain.Summary{OneLine: content, Analysis: content}, nil
}

func buildSummaryPrompt(video domain.Video, transcript []domain.TranscriptSegment) string {
	var b strings.Builder
	b.WriteString("Video title: ")
	b.WriteString(video.Title)
	b.WriteString("\nAuthor: ")
	b.WriteString(video.Author)
	b.WriteString("\nTranscript:\n")
	for _, segment := range transcript {
		text := strings.TrimSpace(segment.Text)
		if text != "" {
			b.WriteString(text)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
