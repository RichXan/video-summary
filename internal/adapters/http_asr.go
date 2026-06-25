package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"video-summary-mvp/internal/domain"
)

type HTTPASRTranscriber struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

type httpASRResponse struct {
	Text     string                     `json:"text"`
	Segments []domain.TranscriptSegment `json:"segments"`
}

func (t HTTPASRTranscriber) Transcribe(ctx context.Context, media domain.MediaAsset) ([]domain.TranscriptSegment, error) {
	if strings.TrimSpace(t.BaseURL) == "" {
		return nil, errors.New("asr base url is not configured")
	}
	if strings.TrimSpace(media.AudioPath) == "" {
		return nil, errors.New("audio path is required")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writeASRMultipart(writer, media.AudioPath, firstNonEmptyString(t.Model, "whisper-1")); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(t.BaseURL, "/") + "/v1/audio/transcriptions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errors.New(strings.TrimSpace(string(resBody)))
	}

	var parsed httpASRResponse
	if err := json.Unmarshal(resBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Segments) > 0 {
		return parsed.Segments, nil
	}
	if strings.TrimSpace(parsed.Text) == "" {
		return nil, errors.New("asr response did not include transcript text")
	}
	return []domain.TranscriptSegment{{Text: strings.TrimSpace(parsed.Text)}}, nil
}

func writeASRMultipart(writer *multipart.Writer, audioPath string, model string) error {
	if err := writer.WriteField("model", model); err != nil {
		return err
	}
	if err := writer.WriteField("response_format", "verbose_json"); err != nil {
		return err
	}

	file, err := os.Open(audioPath)
	if err != nil {
		return err
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	return err
}
