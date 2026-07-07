ARG GO_IMAGE=golang:1.25-bookworm
ARG PYTHON_IMAGE=python:3.12-slim

FROM ${GO_IMAGE} AS go-builder

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/video-summary-api ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/video-summary-worker ./cmd/worker

FROM ${PYTHON_IMAGE} AS runtime

ARG JIJI_REPO_URL=https://github.com/jiji262/douyin-downloader.git
ARG APT_DEBIAN_MIRROR=http://mirrors.cloud.tencent.com/debian
ARG APT_SECURITY_MIRROR=http://mirrors.cloud.tencent.com/debian-security
ARG PIP_INDEX_URL=https://mirrors.cloud.tencent.com/pypi/simple

ENV PYTHONUNBUFFERED=1 \
    PYTHONUTF8=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_INDEX_URL=${PIP_INDEX_URL}

RUN sed -i \
        -e "s|http://deb.debian.org/debian|${APT_DEBIAN_MIRROR}|g" \
        -e "s|http://deb.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        /etc/apt/sources.list.d/debian.sources && \
    printf 'Acquire::Retries "5";\nAcquire::http::Timeout "60";\nAcquire::https::Timeout "60";\n' > /etc/apt/apt.conf.d/80-retries && \
    for attempt in 1 2 3; do \
        apt-get update && \
        apt-get install -y --no-install-recommends ca-certificates git ffmpeg && \
        break; \
        if [ "$attempt" = "3" ]; then exit 1; fi; \
        rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/partial/*; \
        sleep 5; \
    done && \
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
