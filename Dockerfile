FROM golang:1.25-bookworm AS go-builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/video-summary-api ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/video-summary-worker ./cmd/worker

FROM python:3.12-slim AS runtime

ARG JIJI_REPO_URL=https://github.com/jiji262/douyin-downloader.git

ENV PYTHONUNBUFFERED=1 \
    PYTHONUTF8=1 \
    PIP_NO_CACHE_DIR=1

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates git ffmpeg && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
RUN git clone --depth 1 "$JIJI_REPO_URL" /opt/douyin-downloader
RUN python -m pip install --upgrade pip && \
    python -m pip install -r /opt/douyin-downloader/requirements.txt fastapi uvicorn playwright httpx imageio-ffmpeg faster-whisper

COPY --from=go-builder /out/video-summary-api /usr/local/bin/video-summary-api
COPY --from=go-builder /out/video-summary-worker /usr/local/bin/video-summary-worker
COPY scripts ./scripts

RUN mkdir -p /data/work

ENV JIJI_REPO=/opt/douyin-downloader \
    JIJI_PYTHON=python \
    MEDIA_WORKDIR=/data/work \
    LOG_FORMAT=json

EXPOSE 8080

CMD ["video-summary-api"]
