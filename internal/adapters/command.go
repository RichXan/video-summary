package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"

	"video-summary-mvp/internal/domain"
)

type CommandTranscriber struct {
	Name string
	Args []string
}

func (t CommandTranscriber) Transcribe(ctx context.Context, media domain.MediaAsset) ([]domain.TranscriptSegment, error) {
	if strings.TrimSpace(t.Name) == "" {
		return nil, errors.New("asr command is not configured")
	}

	args := replaceArgs(t.Args, map[string]string{
		"{audio}": media.AudioPath,
	})
	cmd := exec.CommandContext(ctx, t.Name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, errors.New(strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	var transcript []domain.TranscriptSegment
	if err := json.Unmarshal(stdout.Bytes(), &transcript); err == nil && len(transcript) > 0 {
		return transcript, nil
	}

	text := strings.TrimSpace(stdout.String())
	if text == "" {
		return nil, errors.New("asr command produced empty output")
	}
	return []domain.TranscriptSegment{{Text: text}}, nil
}

func replaceArgs(args []string, replacements map[string]string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		value := arg
		for key, replacement := range replacements {
			value = strings.ReplaceAll(value, key, replacement)
		}
		out[i] = value
	}
	return out
}
