#!/usr/bin/env sh
set -eu

if [ "$#" -lt 1 ]; then
  echo "usage: prepare-media.sh <url>" >&2
  exit 2
fi

YTDLP_BIN="${YTDLP_BIN:-yt-dlp}"
FFMPEG_BIN="${FFMPEG_BIN:-ffmpeg}"
MEDIA_WORKDIR="${MEDIA_WORKDIR:-work/real}"

if ! command -v "$YTDLP_BIN" >/dev/null 2>&1; then
  echo "yt-dlp was not found. Install yt-dlp or set YTDLP_BIN." >&2
  exit 127
fi
if ! command -v "$FFMPEG_BIN" >/dev/null 2>&1; then
  echo "ffmpeg was not found. Install ffmpeg or set FFMPEG_BIN." >&2
  exit 127
fi

mkdir -p "$MEDIA_WORKDIR"
output_template="$MEDIA_WORKDIR/video.%(ext)s"

set -- \
  --no-playlist \
  --force-overwrites \
  --merge-output-format mp4 \
  --print after_move:filepath \
  -o "$output_template" \
  "$1"
if [ -n "${YTDLP_COOKIES:-}" ]; then
  set -- --cookies "$YTDLP_COOKIES" "$@"
fi
if [ -n "${YTDLP_COOKIES_FROM_BROWSER:-}" ]; then
  set -- --cookies-from-browser "$YTDLP_COOKIES_FROM_BROWSER" "$@"
fi

download_output=$("$YTDLP_BIN" "$@")
video_path=$(printf '%s\n' "$download_output" | awk 'NF { line=$0 } END { print line }')
if [ -z "$video_path" ]; then
  echo "yt-dlp did not print a downloaded video path." >&2
  exit 1
fi

audio_path="$MEDIA_WORKDIR/audio.wav"
"$FFMPEG_BIN" -y -i "$video_path" -vn -ac 1 -ar 16000 "$audio_path" >/dev/null

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

printf '{"video_path":"%s","audio_path":"%s"}\n' "$(json_escape "$video_path")" "$(json_escape "$audio_path")"
