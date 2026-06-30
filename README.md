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

## Async Production-Style Mode

The synchronous endpoint is useful for local smoke tests, but production-style runs should use the job API with a worker and PostgreSQL-backed queue.

Create a job:

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/jobs `
  -ContentType 'application/json; charset=utf-8' `
  -Body '{"url":"https://v.douyin.com/demo/"}'
```

Check status:

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/jobs/<job_id>
```

Operational endpoints:

- `GET /healthz`: process is alive.
- `GET /readyz`: API is ready.
- `GET /metrics`: Prometheus-style service and job counters.

### Docker Compose

Prepare local secrets:

```powershell
Copy-Item .env.example .env
New-Item -ItemType Directory -Force secrets
Copy-Item D:\Downloads\douyin-firefox-cookies.txt secrets\douyin-cookies.txt
Set-Content -Path secrets\llm-token.txt -Value "<your relay token>"
```

Edit `.env` and set the LLM relay URL/model values, for example:

```text
ANTHROPIC_BASE_URL=https://your-relay.example.com
SUMMARY_MODEL=chatgpt5.5
```

Start the production-style stack:

```powershell
docker compose --env-file .env up --build
```

This starts:

- `postgres`: persistent job/result database.
- `api`: receives jobs and exposes health/readiness/metrics.
- `worker`: claims queued jobs, downloads media, runs ASR, and calls the summarizer.

Resource controls are configured in `docker-compose.yml` with per-service `cpus`, `mem_limit`, `WORKER_CONCURRENCY`, `WORKER_POLL_INTERVAL`, and `JOB_TIMEOUT`.

Secrets and cookies are mounted through Docker secrets. Real files under `secrets/` are ignored by git; do not commit cookie exports or API tokens. The application also supports `SUMMARY_AUTH_TOKEN_FILE` and `ANTHROPIC_AUTH_TOKEN_FILE` for secret-file based deployments.

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

## Real Douyin Pipeline

The real pipeline needs external tools and, for many Douyin links, fresh cookies. The
recommended downloader path is `jiji262/douyin-downloader`, because it succeeds on
links that currently fail with `yt-dlp`'s Douyin extractor.

Required local pieces:

- `jiji262/douyin-downloader` for resolving and downloading Douyin videos.
- `imageio-ffmpeg` from the downloader environment for converting video to `16kHz` mono WAV.
- `faster-whisper` for local ASR, or another command/HTTP ASR adapter.
- Optional: an OpenAI-compatible LLM service, such as Ollama, LM Studio, vLLM, Qwen, or DeepSeek.

Douyin often rejects bare URL downloads. The most reliable cookie path tested for this MVP is:

1. Install Firefox.
2. Log in to `https://www.douyin.com/`.
3. Confirm the target video plays in Firefox.
4. Close Firefox.
5. Export cookies with `yt-dlp --cookies-from-browser firefox --cookies D:\Downloads\douyin-firefox-cookies.txt ...`, or export a Netscape `cookies.txt` file with a trusted browser extension.

### Recommended Windows Setup

```powershell
git clone https://github.com/jiji262/douyin-downloader.git D:\Tools\douyin-downloader

py -m venv D:\Tools\douyin-downloader\.venv
& D:\Tools\douyin-downloader\.venv\Scripts\python.exe -m pip install `
  -r D:\Tools\douyin-downloader\requirements.txt `
  fastapi uvicorn playwright httpx

py -m venv D:\Tools\video-summary-asr
& D:\Tools\video-summary-asr\Scripts\python.exe -m pip install faster-whisper
```

Then start this service:

```powershell
$env:VIDEO_RESOLVER="mock"

$env:MEDIA_PREPARER="command"
$env:MEDIA_COMMAND="py"
$env:MEDIA_ARGS="scripts\prepare_media_jiji.py {url}"
$env:JIJI_REPO="D:\Tools\douyin-downloader"
$env:JIJI_PYTHON="D:\Tools\douyin-downloader\.venv\Scripts\python.exe"
$env:DOUYIN_COOKIES="D:\Downloads\douyin-firefox-cookies.txt"
$env:MEDIA_WORKDIR="work\real"

$env:MODE="command"
$env:ASR_COMMAND="D:\Tools\video-summary-asr\Scripts\python.exe"
$env:ASR_ARGS="scripts\transcribe_faster_whisper.py {audio} --model small --language zh"

$env:SUMMARY_MODE="template"

go run ./cmd/server
```

Call the API:

```powershell
Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/videos/summarize `
  -ContentType 'application/json; charset=utf-8' `
  -Body '{"url":"https://www.douyin.com/video/7655554393391537802"}'
```

The default template summarizer is extractive and grounded in the real transcript. For deeper synthesis, run an OpenAI-compatible LLM service and set `SUMMARY_MODE=http`.

To use an OpenAI-compatible relay that exposes Chat Completions with Anthropic-style environment names:

```powershell
$env:SUMMARY_MODE="http"
$env:ANTHROPIC_BASE_URL="https://your-relay.example.com"
$env:ANTHROPIC_AUTH_TOKEN="<your relay token>"
$env:SUMMARY_MODEL="chatgpt5.5"
```

`SUMMARY_BASE_URL`, `SUMMARY_AUTH_TOKEN`, and `SUMMARY_MODEL` take precedence when set. If only `ANTHROPIC_BASE_URL` is set, the default summary model is `chatgpt5.5`.

### yt-dlp Fallback

`yt-dlp` is still supported through the generic command scripts, but current Douyin links may fail with `Fresh cookies are needed` even when cookies are valid.

```powershell
$env:YTDLP_COOKIES="C:\path\to\douyin-cookies.txt"

$env:VIDEO_RESOLVER="command"
$env:VIDEO_COMMAND="powershell"
$env:VIDEO_ARGS="-ExecutionPolicy Bypass -File scripts\resolve-video.ps1 {url}"

$env:MEDIA_PREPARER="command"
$env:MEDIA_COMMAND="powershell"
$env:MEDIA_ARGS="-ExecutionPolicy Bypass -File scripts\prepare-media.ps1 {resolved_url}"
$env:MEDIA_WORKDIR="work\real"

$env:MODE="http"
$env:ASR_BASE_URL="http://localhost:18081"
$env:ASR_MODEL="whisper-1"

$env:SUMMARY_MODE="http"
$env:SUMMARY_BASE_URL="http://localhost:18082"
$env:SUMMARY_MODEL="qwen"

go run ./cmd/server
```

### Linux, macOS, or WSL

```bash
export VIDEO_RESOLVER=command
export VIDEO_COMMAND=./scripts/resolve-video.sh
export VIDEO_ARGS='{url}'

export MEDIA_PREPARER=command
export MEDIA_COMMAND=./scripts/prepare-media.sh
export MEDIA_ARGS='{resolved_url}'
export MEDIA_WORKDIR=work/real

export MODE=http
export ASR_BASE_URL=http://localhost:18081
export ASR_MODEL=whisper-1

export SUMMARY_MODE=http
export SUMMARY_BASE_URL=http://localhost:18082
export SUMMARY_MODEL=qwen

go run ./cmd/server
```

The helper scripts also accept:

- `YTDLP_BIN` to use a non-`PATH` `yt-dlp` binary.
- `FFMPEG_BIN` to use a non-`PATH` `ffmpeg` binary.
- `YTDLP_COOKIES` for a Netscape cookie file.
- `YTDLP_COOKIES_FROM_BROWSER` for `yt-dlp --cookies-from-browser`.

If `yt-dlp` returns `Fresh cookies are needed`, the service is configured correctly but Douyin is blocking anonymous access. Provide a fresh cookie file and retry.

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

For hosted or relay services that need bearer authentication:

```powershell
$env:SUMMARY_MODE="http"
$env:SUMMARY_BASE_URL="https://your-openai-compatible-endpoint"
$env:SUMMARY_AUTH_TOKEN="<your token>"
$env:SUMMARY_MODEL="chatgpt5.5"
go run ./cmd/server
```

The service also accepts `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, and `ANTHROPIC_MODEL` as aliases for relay deployments that use those names. Tokens are read only from environment variables and should not be committed.

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
