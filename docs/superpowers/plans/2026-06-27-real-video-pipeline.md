# Real Video Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the MVP configurable for a real short-video flow: resolve a Douyin link, download video, extract WAV audio, transcribe with a real ASR adapter, and summarize the transcript.

**Architecture:** Keep the Go service adapters unchanged where possible and plug real tools in through the existing command/http boundaries. Add thin cross-platform scripts around `yt-dlp` and `ffmpeg` so cookies, browser cookies, output paths, and JSON output are stable.

**Tech Stack:** Go 1.25, PowerShell, POSIX shell, `yt-dlp`, `ffmpeg`, OpenAI-compatible ASR and LLM HTTP services.

---

### Task 1: Robust Command Argument Configuration

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] Write a failing test showing quoted args are preserved.
- [ ] Implement quote-aware env argument splitting.
- [ ] Run `go test ./internal/config`.

### Task 2: Real Video Resolver Scripts

**Files:**
- Create: `scripts/resolve-video.ps1`
- Create: `scripts/resolve-video.sh`

- [ ] Add scripts that call `yt-dlp -J` and support `YTDLP_COOKIES` / `YTDLP_COOKIES_FROM_BROWSER`.
- [ ] Ensure stdout remains the raw JSON expected by `CommandVideoResolver`.
- [ ] Run PowerShell parser validation and shell syntax validation.

### Task 3: Real Media Preparer Scripts

**Files:**
- Create: `scripts/prepare-media.ps1`
- Create: `scripts/prepare-media.sh`

- [ ] Add scripts that download a video with `yt-dlp`, convert to `16kHz mono wav` with `ffmpeg`, and print `{"video_path": "...", "audio_path": "..."}`.
- [ ] Support cookies and configurable work directory.
- [ ] Run parser/syntax validation.

### Task 4: Documentation and Smoke Commands

**Files:**
- Modify: `README.md`

- [ ] Document required external dependencies.
- [ ] Document Douyin cookie requirement and exact env vars.
- [ ] Document end-to-end commands for Windows and Linux/macOS shells.
- [ ] Run full test suite and verify the real downloader failure mode with the provided link.
