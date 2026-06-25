package adapters

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"video-summary-mvp/internal/domain"
)

func TestCommandMediaPreparerParsesJSONOutput(t *testing.T) {
	t.Parallel()

	command := writeMediaTestCommand(t, `{"video_path":"work/video.mp4","audio_path":"work/audio.wav"}`)
	preparer := CommandMediaPreparer{Name: command, Args: []string{}}

	asset, err := preparer.Prepare(context.Background(), domain.Video{SourceURL: "https://example.com/source"})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	if asset.VideoPath != "work/video.mp4" {
		t.Fatalf("video path = %q", asset.VideoPath)
	}
	if asset.AudioPath != "work/audio.wav" {
		t.Fatalf("audio path = %q", asset.AudioPath)
	}
}

func TestCommandMediaPreparerTreatsPlainOutputAsAudioPath(t *testing.T) {
	t.Parallel()

	command := writeMediaTestCommand(t, `work/audio.wav`)
	preparer := CommandMediaPreparer{Name: command, Args: []string{}}

	asset, err := preparer.Prepare(context.Background(), domain.Video{SourceURL: "https://example.com/source"})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	if asset.AudioPath != "work/audio.wav" {
		t.Fatalf("audio path = %q", asset.AudioPath)
	}
}

func TestCommandMediaPreparerReplacesVideoPlaceholders(t *testing.T) {
	t.Parallel()

	command := writeEchoPlainArgsCommand(t)
	preparer := CommandMediaPreparer{Name: command, Args: []string{"{url}", "{resolved_url}", "{title}"}}

	asset, err := preparer.Prepare(context.Background(), domain.Video{
		SourceURL:   "https://short.example/video",
		ResolvedURL: "https://resolved.example/video",
		Title:       "Demo-title",
	})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	want := "https://short.example/video https://resolved.example/video Demo-title"
	if asset.AudioPath != want {
		t.Fatalf("audio path = %q, want %q", asset.AudioPath, want)
	}
}

func writeMediaTestCommand(t *testing.T, output string) string {
	t.Helper()

	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, "media.bat")
		if err := os.WriteFile(path, []byte("@echo "+output+"\r\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	path := filepath.Join(dir, "media.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s\\n' '"+output+"'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeEchoPlainArgsCommand(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, "echoargs.bat")
		if err := os.WriteFile(path, []byte("@echo %*\r\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	path := filepath.Join(dir, "echoargs.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
