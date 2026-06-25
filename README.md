# Video Summary MVP

Go backend MVP for turning a short-video URL into a structured transcript-based summary.

## Current MVP

- `POST /api/v1/videos/summarize`
- Mock mode for local smoke tests.
- Optional web metadata resolver that follows short links and extracts title/author from video page HTML.
- Mock media preparer, ASR transcriber, and template summarizer.
- Command ASR adapter for tools such as `whisper.cpp`, `FunASR`, or `faster-whisper` wrappers.
- No external Go dependencies.

## Run

```powershell
$env:GOCACHE="$PWD\\.gocache"
go run ./cmd/server
```

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/videos/summarize `
  -ContentType 'application/json' `
  -Body '{"url":"https://v.douyin.com/demo/"}'
```

## Web Metadata Mode

Use this mode to resolve a real video page and extract metadata before the mock ASR step:

```powershell
$env:VIDEO_RESOLVER="web"
go run ./cmd/server
```

Then call the API with a real short link:

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/videos/summarize `
  -ContentType 'application/json; charset=utf-8' `
  -Body '{"url":"https://v.douyin.com/fZYSzDGCgcY/"}'
```

This fetches page metadata only. Downloading media and running real ASR still require replacing `StaticMediaPreparer` and configuring a real transcriber.

## Command Video Resolver Mode

Use this mode when a dedicated downloader/parser is available. The command must print JSON with fields similar to `yt-dlp -J`:

```json
{
  "title": "Video title",
  "uploader": "Author name",
  "duration": 702,
  "webpage_url": "https://www.douyin.com/video/..."
}
```

Example:

```powershell
$env:VIDEO_RESOLVER="command"
$env:VIDEO_COMMAND="yt-dlp"
$env:VIDEO_ARGS="-J {url}"
go run ./cmd/server
```

For Douyin, a dedicated parser such as a wrapper around `Douyin_TikTok_Download_API` can expose the same JSON shape and plug in without changing Go application code.

## Command Media Preparer Mode

Use this mode to call an external downloader/transcoder. The command can print either:

- plain text: the final audio path
- JSON:

```json
{
  "video_path": "work/video.mp4",
  "audio_path": "work/audio.wav"
}
```

Configuration:

```powershell
$env:MEDIA_PREPARER="command"
$env:MEDIA_COMMAND="path/to/prepare-media-script"
$env:MEDIA_ARGS="{resolved_url} work/audio.wav"
go run ./cmd/server
```

Supported placeholders:

- `{url}` / `{source_url}`
- `{resolved_url}`
- `{title}`

A practical script can run:

```bash
yt-dlp -o work/input.%(ext)s "$1"
ffmpeg -y -i work/input.mp4 -vn -ac 1 -ar 16000 "$2"
printf '%s\n' "$2"
```

## Command ASR Mode

The command adapter expects stdout to be either plain transcript text or JSON:

```json
[
  {"start":0,"end":3.2,"text":"hello"}
]
```

Example configuration:

```powershell
$env:MODE="command"
$env:ASR_COMMAND="whisper-cli"
$env:ASR_ARGS="-m models/ggml-base.bin -f {audio}"
go run ./cmd/server
```

## OpenAI-Compatible ASR HTTP Mode

Use this mode with a free/open-source ASR server that exposes an OpenAI-compatible endpoint:

```text
POST /v1/audio/transcriptions
multipart fields:
- file
- model
- response_format=verbose_json
```

Configuration:

```powershell
$env:MODE="http"
$env:ASR_BASE_URL="http://localhost:18081"
$env:ASR_MODEL="whisper-1"
go run ./cmd/server
```

Local protocol smoke test:

```powershell
go run ./tools/mock-asr-server
```

Then start the main service with `MODE=http`. Replace the mock server with `whisper.cpp` server, `faster-whisper-server`, or a FunASR OpenAI-compatible wrapper for real transcription.

## OpenAI-Compatible LLM Summary Mode

Use this mode with Ollama, LM Studio, vLLM, or any OpenAI-compatible local LLM service:

```powershell
$env:SUMMARY_MODE="http"
$env:SUMMARY_BASE_URL="http://localhost:11434"
$env:SUMMARY_MODEL="qwen"
go run ./cmd/server
```

The model should return JSON in the assistant message:

```json
{
  "one_line": "...",
  "outline": ["..."],
  "quotes": ["..."],
  "viewpoints": ["..."],
  "analysis": "..."
}
```

Local full-pipeline smoke servers:

```powershell
go run ./tools/mock-asr-server
go run ./tools/mock-llm-server
```

Then start the main service:

```powershell
$env:MEDIA_PREPARER="command"
$env:MEDIA_COMMAND="$PWD\work\mock-media.bat"
$env:MODE="http"
$env:ASR_BASE_URL="http://localhost:18081"
$env:SUMMARY_MODE="http"
$env:SUMMARY_BASE_URL="http://localhost:18082"
go run ./cmd/server
```
