# Video Summary MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go backend MVP that accepts a short-video URL and returns structured transcript-based summary output.

**Architecture:** The service uses stdlib HTTP handlers and a small application service that orchestrates video metadata extraction, media preparation, ASR transcription, and summarization. External tools such as downloaders, ffmpeg, whisper.cpp, or FunASR are hidden behind interfaces so the MVP can run in mock mode before those binaries are installed.

**Tech Stack:** Go stdlib HTTP, JSON file-like in-memory repository for MVP, command adapters for external binaries, table-driven Go tests.

---

### Task 1: Domain and Orchestrator

**Files:**
- Create: `go.mod`
- Create: `internal/domain/video.go`
- Create: `internal/app/service.go`
- Test: `internal/app/service_test.go`

- [ ] Write failing tests for successful orchestration and dependency errors.
- [ ] Implement domain types and `Service.Summarize`.
- [ ] Run `go test ./internal/app`.

### Task 2: HTTP API

**Files:**
- Create: `internal/httpapi/handler.go`
- Test: `internal/httpapi/handler_test.go`

- [ ] Write failing tests for `POST /api/v1/videos/summarize`.
- [ ] Implement request parsing, status codes, and JSON response.
- [ ] Run `go test ./internal/httpapi`.

### Task 3: Adapters and Server

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/adapters/mock.go`
- Create: `internal/adapters/command.go`
- Create: `cmd/server/main.go`
- Create: `README.md`
- Create: `.gitignore`

- [ ] Write focused tests for command rendering where useful.
- [ ] Implement mock adapters and command adapters.
- [ ] Wire the server with environment configuration.
- [ ] Run `go test ./...` and `go build ./cmd/server`.
