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

type CommandMediaPreparer struct {
	Name string
	Args []string
}

func (p CommandMediaPreparer) Prepare(ctx context.Context, video domain.Video) (domain.MediaAsset, error) {
	if strings.TrimSpace(p.Name) == "" {
		return domain.MediaAsset{}, errors.New("media command is not configured")
	}

	args := replaceArgs(p.Args, map[string]string{
		"{url}":          video.SourceURL,
		"{source_url}":   video.SourceURL,
		"{resolved_url}": video.ResolvedURL,
		"{title}":        video.Title,
	})
	cmd := exec.CommandContext(ctx, p.Name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return domain.MediaAsset{}, errors.New(strings.TrimSpace(stderr.String()))
		}
		return domain.MediaAsset{}, err
	}

	var asset domain.MediaAsset
	if err := json.Unmarshal(stdout.Bytes(), &asset); err == nil && strings.TrimSpace(asset.AudioPath) != "" {
		return asset, nil
	}

	audioPath := strings.TrimSpace(stdout.String())
	if audioPath == "" {
		return domain.MediaAsset{}, errors.New("media command produced empty output")
	}
	return domain.MediaAsset{AudioPath: audioPath}, nil
}
