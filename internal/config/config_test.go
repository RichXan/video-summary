package config

import (
	"os"
	"testing"
)

func TestSplitArgsPreservesQuotedValues(t *testing.T) {
	t.Parallel()

	got := splitArgs(`-J --cookies "C:\Users\me\douyin cookies.txt" "{url}"`)
	want := []string{"-J", "--cookies", `C:\Users\me\douyin cookies.txt`, "{url}"}

	if len(got) != len(want) {
		t.Fatalf("len(splitArgs) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitArgs()[%d] = %q, want %q; full = %#v", i, got[i], want[i], got)
		}
	}
}

func TestSplitArgsSupportsSingleQuotes(t *testing.T) {
	t.Parallel()

	got := splitArgs(`--output 'work/real/%(id)s.%(ext)s' {resolved_url}`)
	want := []string{"--output", "work/real/%(id)s.%(ext)s", "{resolved_url}"}

	if len(got) != len(want) {
		t.Fatalf("len(splitArgs) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitArgs()[%d] = %q, want %q; full = %#v", i, got[i], want[i], got)
		}
	}
}

func TestLoadUsesAnthropicCompatibleSummaryEnv(t *testing.T) {
	t.Setenv("SUMMARY_MODE", "http")
	t.Setenv("ANTHROPIC_BASE_URL", "https://ai.example.com")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "secret-token")

	cfg := Load()

	if cfg.SummaryBaseURL != "https://ai.example.com" {
		t.Fatalf("SummaryBaseURL = %q", cfg.SummaryBaseURL)
	}
	if cfg.SummaryAuthToken != "secret-token" {
		t.Fatalf("SummaryAuthToken was not loaded from ANTHROPIC_AUTH_TOKEN")
	}
	if cfg.SummaryModel != "chatgpt5.5" {
		t.Fatalf("SummaryModel = %q, want chatgpt5.5", cfg.SummaryModel)
	}
}

func TestLoadPrefersExplicitSummaryEnvOverAnthropicEnv(t *testing.T) {
	t.Setenv("SUMMARY_BASE_URL", "http://localhost:11434")
	t.Setenv("SUMMARY_AUTH_TOKEN", "summary-token")
	t.Setenv("SUMMARY_MODEL", "qwen")
	t.Setenv("ANTHROPIC_BASE_URL", "https://ai.example.com")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "anthropic-token")
	t.Setenv("ANTHROPIC_MODEL", "chatgpt5.5")

	cfg := Load()

	if cfg.SummaryBaseURL != "http://localhost:11434" {
		t.Fatalf("SummaryBaseURL = %q", cfg.SummaryBaseURL)
	}
	if cfg.SummaryAuthToken != "summary-token" {
		t.Fatalf("SummaryAuthToken = %q", cfg.SummaryAuthToken)
	}
	if cfg.SummaryModel != "qwen" {
		t.Fatalf("SummaryModel = %q", cfg.SummaryModel)
	}
}

func TestLoadReadsSummaryAuthTokenFromFile(t *testing.T) {
	path := t.TempDir() + "/token.txt"
	if err := os.WriteFile(path, []byte("file-token\n"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	t.Setenv("SUMMARY_AUTH_TOKEN_FILE", path)

	cfg := Load()

	if cfg.SummaryAuthToken != "file-token" {
		t.Fatalf("SummaryAuthToken = %q", cfg.SummaryAuthToken)
	}
}
