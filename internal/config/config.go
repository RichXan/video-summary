package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr           string
	Mode           string
	VideoResolver  string
	VideoCommand   string
	VideoArgs      []string
	MediaPreparer  string
	MediaCommand   string
	MediaArgs      []string
	ASRBaseURL     string
	ASRCommand     string
	ASRArgs        []string
	ASRModel       string
	SummaryMode    string
	SummaryBaseURL string
	SummaryModel   string
}

func Load() Config {
	return Config{
		Addr:           env("ADDR", ":8080"),
		Mode:           env("MODE", "mock"),
		VideoResolver:  env("VIDEO_RESOLVER", "mock"),
		VideoCommand:   os.Getenv("VIDEO_COMMAND"),
		VideoArgs:      splitArgs(os.Getenv("VIDEO_ARGS")),
		MediaPreparer:  env("MEDIA_PREPARER", "mock"),
		MediaCommand:   os.Getenv("MEDIA_COMMAND"),
		MediaArgs:      splitArgs(os.Getenv("MEDIA_ARGS")),
		ASRBaseURL:     os.Getenv("ASR_BASE_URL"),
		ASRCommand:     os.Getenv("ASR_COMMAND"),
		ASRArgs:        splitArgs(os.Getenv("ASR_ARGS")),
		ASRModel:       env("ASR_MODEL", "whisper-1"),
		SummaryMode:    env("SUMMARY_MODE", "template"),
		SummaryBaseURL: os.Getenv("SUMMARY_BASE_URL"),
		SummaryModel:   env("SUMMARY_MODEL", "qwen"),
	}
}

func env(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitArgs(value string) []string {
	fields := strings.Fields(value)
	if fields == nil {
		return []string{}
	}
	return fields
}

func EnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}
	return value
}
