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

type CommandVideoResolver struct {
	Name string
	Args []string
}

type commandVideoOutput struct {
	Title      string `json:"title"`
	Uploader   string `json:"uploader"`
	Author     string `json:"author"`
	Duration   int    `json:"duration"`
	WebpageURL string `json:"webpage_url"`
	OriginalURL string `json:"original_url"`
}

func (r CommandVideoResolver) Resolve(ctx context.Context, rawURL string) (domain.Video, error) {
	if strings.TrimSpace(r.Name) == "" {
		return domain.Video{}, errors.New("video command is not configured")
	}

	args := replaceArgs(r.Args, map[string]string{"{url}": rawURL})
	cmd := exec.CommandContext(ctx, r.Name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return domain.Video{}, errors.New(strings.TrimSpace(stderr.String()))
		}
		return domain.Video{}, err
	}

	var parsed commandVideoOutput
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		return domain.Video{}, err
	}

	title := firstNonEmptyString(parsed.Title, "Untitled short video")
	author := firstNonEmptyString(parsed.Uploader, parsed.Author, "unknown")
	resolvedURL := firstNonEmptyString(parsed.WebpageURL, parsed.OriginalURL)

	return domain.Video{
		SourceURL:  rawURL,
		ResolvedURL: resolvedURL,
		Title:      title,
		Author:     author,
		Duration:   parsed.Duration,
	}, nil
}
