package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr             string
	Mode             string
	VideoResolver    string
	VideoCommand     string
	VideoArgs        []string
	MediaPreparer    string
	MediaCommand     string
	MediaArgs        []string
	ASRBaseURL       string
	ASRCommand       string
	ASRArgs          []string
	ASRModel         string
	SummaryMode      string
	SummaryBaseURL   string
	SummaryAuthToken string
	SummaryModel     string
}

func Load() Config {
	summaryBaseURL := firstEnv("SUMMARY_BASE_URL", "ANTHROPIC_BASE_URL")
	return Config{
		Addr:             env("ADDR", ":8080"),
		Mode:             env("MODE", "mock"),
		VideoResolver:    env("VIDEO_RESOLVER", "mock"),
		VideoCommand:     os.Getenv("VIDEO_COMMAND"),
		VideoArgs:        splitArgs(os.Getenv("VIDEO_ARGS")),
		MediaPreparer:    env("MEDIA_PREPARER", "mock"),
		MediaCommand:     os.Getenv("MEDIA_COMMAND"),
		MediaArgs:        splitArgs(os.Getenv("MEDIA_ARGS")),
		ASRBaseURL:       os.Getenv("ASR_BASE_URL"),
		ASRCommand:       os.Getenv("ASR_COMMAND"),
		ASRArgs:          splitArgs(os.Getenv("ASR_ARGS")),
		ASRModel:         env("ASR_MODEL", "whisper-1"),
		SummaryMode:      env("SUMMARY_MODE", "template"),
		SummaryBaseURL:   summaryBaseURL,
		SummaryAuthToken: firstEnv("SUMMARY_AUTH_TOKEN", "OPENAI_API_KEY", "ANTHROPIC_AUTH_TOKEN"),
		SummaryModel:     summaryModel(summaryBaseURL),
	}
}

func env(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func summaryModel(summaryBaseURL string) string {
	if value := firstEnv("SUMMARY_MODEL", "ANTHROPIC_MODEL"); value != "" {
		return value
	}
	if strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL")) != "" && strings.TrimSpace(summaryBaseURL) != "" {
		return "chatgpt5.5"
	}
	return "qwen"
}

func splitArgs(value string) []string {
	var args []string
	var current strings.Builder
	var quote rune
	for _, r := range value {
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			continue
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

func EnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}
	return value
}
