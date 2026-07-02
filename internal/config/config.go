package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr               string
	APIKey             string
	Role               string
	DatabaseURL        string
	Mode               string
	VideoResolver      string
	VideoCommand       string
	VideoArgs          []string
	MediaPreparer      string
	MediaCommand       string
	MediaArgs          []string
	ASRBaseURL         string
	ASRCommand         string
	ASRArgs            []string
	ASRModel           string
	SummaryMode        string
	SummaryBaseURL     string
	SummaryAuthToken   string
	SummaryModel       string
	WorkerID           string
	WorkerConcurrency  int
	WorkerPollInterval time.Duration
	JobTimeout         time.Duration
	RunningJobTTL      time.Duration
	LogFormat          string
}

func Load() Config {
	summaryBaseURL := firstEnv("SUMMARY_BASE_URL", "ANTHROPIC_BASE_URL")
	return Config{
		Addr:               env("ADDR", ":13000"),
		APIKey:             firstSecret("API_KEY_FILE", "API_KEY"),
		Role:               env("APP_ROLE", "api"),
		DatabaseURL:        firstEnv("DATABASE_URL", "POSTGRES_DSN"),
		Mode:               env("MODE", "mock"),
		VideoResolver:      env("VIDEO_RESOLVER", "mock"),
		VideoCommand:       os.Getenv("VIDEO_COMMAND"),
		VideoArgs:          splitArgs(os.Getenv("VIDEO_ARGS")),
		MediaPreparer:      env("MEDIA_PREPARER", "mock"),
		MediaCommand:       os.Getenv("MEDIA_COMMAND"),
		MediaArgs:          splitArgs(os.Getenv("MEDIA_ARGS")),
		ASRBaseURL:         os.Getenv("ASR_BASE_URL"),
		ASRCommand:         os.Getenv("ASR_COMMAND"),
		ASRArgs:            splitArgs(os.Getenv("ASR_ARGS")),
		ASRModel:           env("ASR_MODEL", "whisper-1"),
		SummaryMode:        env("SUMMARY_MODE", "template"),
		SummaryBaseURL:     summaryBaseURL,
		SummaryAuthToken:   firstSecret("SUMMARY_AUTH_TOKEN_FILE", "ANTHROPIC_AUTH_TOKEN_FILE", "SUMMARY_AUTH_TOKEN", "OPENAI_API_KEY", "ANTHROPIC_AUTH_TOKEN"),
		SummaryModel:       summaryModel(summaryBaseURL),
		WorkerID:           os.Getenv("WORKER_ID"),
		WorkerConcurrency:  EnvInt("WORKER_CONCURRENCY", 1),
		WorkerPollInterval: EnvDuration("WORKER_POLL_INTERVAL", 5*time.Second),
		JobTimeout:         EnvDuration("JOB_TIMEOUT", 30*time.Minute),
		RunningJobTTL:      EnvDuration("RUNNING_JOB_TTL", EnvDuration("JOB_TIMEOUT", 30*time.Minute)+5*time.Minute),
		LogFormat:          env("LOG_FORMAT", "json"),
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

func firstSecret(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		if strings.HasSuffix(key, "_FILE") {
			data, err := os.ReadFile(value)
			if err != nil {
				continue
			}
			return strings.TrimSpace(string(data))
		}
		return value
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

func EnvDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}
