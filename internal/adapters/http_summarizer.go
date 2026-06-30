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
	BaseURL   string
	Model     string
	AuthToken string
	Client    *http.Client
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

const maxSummaryPromptRunes = 24000

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
				Content: "你是中文视频内容分析助手。只返回 JSON，不要 Markdown。字段必须包含 one_line, outline, quotes, viewpoints, analysis。outline/quotes/viewpoints 使用字符串数组。",
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
	if token := strings.TrimSpace(s.AuthToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

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
	if err := json.Unmarshal([]byte(extractJSONContent(content)), &summary); err == nil && strings.TrimSpace(summary.OneLine) != "" {
		return summary, nil
	}
	if content == "" {
		return domain.Summary{}, errors.New("summary response content is empty")
	}
	return domain.Summary{OneLine: content, Analysis: content}, nil
}

func buildSummaryPrompt(video domain.Video, transcript []domain.TranscriptSegment) string {
	var b strings.Builder
	b.WriteString("请根据下面的视频信息和逐字稿，输出紧凑但有信息密度的中文 JSON 摘要。\n")
	b.WriteString("要求：one_line 为一句话总结；outline 给 4-6 个关键要点；quotes 给 2-4 句代表性原话；viewpoints 给主要观点/立场；analysis 给简短分析。\n\n")
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
	return capSummaryPrompt(b.String())
}

func capSummaryPrompt(prompt string) string {
	runes := []rune(prompt)
	if len(runes) <= maxSummaryPromptRunes {
		return prompt
	}
	headSize := maxSummaryPromptRunes * 2 / 3
	tailSize := maxSummaryPromptRunes - headSize
	head := string(runes[:headSize])
	tail := string(runes[len(runes)-tailSize:])
	return head + "\n\n[Transcript truncated: middle content omitted to fit the LLM context window.]\n\n" + tail
}

func extractJSONContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 3 {
			lines = lines[1:]
			if strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
				lines = lines[:len(lines)-1]
			}
			return strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return strings.TrimSpace(trimmed[start : end+1])
	}
	return trimmed
}

type FallbackSummarizer struct {
	Primary interface {
		Summarize(context.Context, domain.Video, []domain.TranscriptSegment) (domain.Summary, error)
	}
	Fallback interface {
		Summarize(context.Context, domain.Video, []domain.TranscriptSegment) (domain.Summary, error)
	}
}

func (s FallbackSummarizer) Summarize(ctx context.Context, video domain.Video, transcript []domain.TranscriptSegment) (domain.Summary, error) {
	if s.Primary == nil {
		if s.Fallback == nil {
			return domain.Summary{}, errors.New("summary fallback is not configured")
		}
		return s.Fallback.Summarize(ctx, video, transcript)
	}
	summary, err := s.Primary.Summarize(ctx, video, transcript)
	if err == nil {
		return summary, nil
	}
	if s.Fallback == nil {
		return domain.Summary{}, err
	}
	return s.Fallback.Summarize(ctx, video, transcript)
}
