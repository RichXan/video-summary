#!/usr/bin/env sh
set -eu

if [ "$#" -lt 1 ]; then
  echo "usage: resolve-video.sh <url>" >&2
  exit 2
fi

YTDLP_BIN="${YTDLP_BIN:-yt-dlp}"
if ! command -v "$YTDLP_BIN" >/dev/null 2>&1; then
  echo "yt-dlp was not found. Install yt-dlp or set YTDLP_BIN." >&2
  exit 127
fi

set -- -J --no-playlist "$1"
if [ -n "${YTDLP_COOKIES:-}" ]; then
  set -- --cookies "$YTDLP_COOKIES" "$@"
fi
if [ -n "${YTDLP_COOKIES_FROM_BROWSER:-}" ]; then
  set -- --cookies-from-browser "$YTDLP_COOKIES_FROM_BROWSER" "$@"
fi

exec "$YTDLP_BIN" "$@"
