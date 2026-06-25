package adapters

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCommandVideoResolverParsesJSONOutput(t *testing.T) {
	t.Parallel()

	command := writeTestCommand(t, `{"title":"Resolved title","uploader":"Resolved author","duration":123,"webpage_url":"https://example.com/video"}`)
	resolver := CommandVideoResolver{Name: command, Args: []string{}}

	video, err := resolver.Resolve(context.Background(), "https://v.douyin.com/demo/")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if video.Title != "Resolved title" {
		t.Fatalf("title = %q", video.Title)
	}
	if video.Author != "Resolved author" {
		t.Fatalf("author = %q", video.Author)
	}
	if video.Duration != 123 {
		t.Fatalf("duration = %d", video.Duration)
	}
	if video.ResolvedURL != "https://example.com/video" {
		t.Fatalf("resolved url = %q", video.ResolvedURL)
	}
}

func TestCommandVideoResolverReplacesURLPlaceholder(t *testing.T) {
	t.Parallel()

	command := writeEchoArgsCommand(t)
	resolver := CommandVideoResolver{Name: command, Args: []string{"--dump-json", "{url}"}}

	video, err := resolver.Resolve(context.Background(), "https://v.douyin.com/demo/")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if video.Title != "--dump-json https://v.douyin.com/demo/" {
		t.Fatalf("title = %q", video.Title)
	}
}

func writeTestCommand(t *testing.T, output string) string {
	t.Helper()

	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, "cmd.bat")
		if err := os.WriteFile(path, []byte("@echo "+output+"\r\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	path := filepath.Join(dir, "cmd.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s\\n' '"+output+"'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeEchoArgsCommand(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, "cmd.bat")
		if err := os.WriteFile(path, []byte("@echo {\"title\":\"%*\"}\r\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	path := filepath.Join(dir, "cmd.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '{\"title\":\"%s\"}\\n' \"$*\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
