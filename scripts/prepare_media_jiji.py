#!/usr/bin/env python3
import argparse
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")
if hasattr(sys.stderr, "reconfigure"):
    sys.stderr.reconfigure(encoding="utf-8")


def load_cookie_file(path: Path) -> dict[str, str]:
    text = path.read_text(encoding="utf-8-sig")
    stripped = text.lstrip()
    if stripped.startswith("{") or stripped.startswith("["):
        data = json.loads(text)
        if isinstance(data, dict) and isinstance(data.get("cookies"), list):
            rows = data["cookies"]
        elif isinstance(data, list):
            rows = data
        elif isinstance(data, dict):
            return {str(k): str(v) for k, v in data.items() if v is not None}
        else:
            rows = []
        return {
            str(row["name"]): str(row.get("value", ""))
            for row in rows
            if isinstance(row, dict) and row.get("name")
        }

    cookies: dict[str, str] = {}
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split("\t")
        if len(parts) >= 7:
            cookies[parts[5]] = parts[6]
    return cookies


def extract_first_url(value: str) -> str:
    match = re.search(r"https?://[^\s]+", value)
    if not match:
        return value.strip()
    return match.group(0).rstrip("，,。.!！?？）)]}\"'")


def douyin_video_id(value: str) -> str:
    parsed = urllib.parse.urlparse(value)
    match = re.search(r"/(?:share/)?video/(\d+)", parsed.path)
    if match:
        return match.group(1)
    return ""


def canonical_douyin_url(value: str) -> str:
    video_id = douyin_video_id(value)
    if video_id:
        return f"https://www.douyin.com/video/{video_id}"
    return value


def resolve_douyin_input(value: str) -> str:
    url = extract_first_url(value)
    parsed = urllib.parse.urlparse(url)
    if parsed.netloc.lower() != "v.douyin.com":
        return canonical_douyin_url(url)

    request = urllib.request.Request(
        url,
        headers={
            "User-Agent": (
                "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) "
                "AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 "
                "Mobile/15E148 Safari/604.1"
            )
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            return canonical_douyin_url(response.url)
    except urllib.error.HTTPError as error:
        location = error.headers.get("Location")
        if location:
            return canonical_douyin_url(urllib.parse.urljoin(url, location))
        raise


def find_newest_mp4(root: Path) -> Path:
    candidates = [p for p in root.rglob("*.mp4") if p.is_file()]
    if not candidates:
        raise FileNotFoundError(f"no mp4 files found under {root}")
    return max(candidates, key=lambda p: p.stat().st_mtime)


def extract_metadata(root: Path) -> dict[str, str | int]:
    candidates = [p for p in root.rglob("*_data.json") if p.is_file()]
    if not candidates:
        return {}
    data_path = max(candidates, key=lambda p: p.stat().st_mtime)
    try:
        data = json.loads(data_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return {}

    metadata: dict[str, str | int] = {}
    title = first_text(data, "item_title", "title", "desc", "caption")
    if title:
        metadata["title"] = title
    author = data.get("author")
    if isinstance(author, dict):
        author_name = first_text(author, "nickname", "unique_id", "short_id")
        if author_name:
            metadata["author"] = author_name
    duration = data.get("duration")
    if isinstance(duration, (int, float)) and duration > 0:
        metadata["duration"] = int(duration / 1000) if duration > 10000 else int(duration)
    return metadata


def first_text(data: dict, *keys: str) -> str:
    for key in keys:
        value = data.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
    return ""


def write_text(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


def run_jiji(repo: Path, python: str, config: Path, url: str, output_dir: Path) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    env["PYTHONIOENCODING"] = "utf-8"
    env["PYTHONUTF8"] = "1"
    env["TERM"] = env.get("TERM", "xterm")
    command = [
        python,
        str(repo / "run.py"),
        "-c",
        str(config),
        "-u",
        url,
        "-p",
        str(output_dir),
        "--show-warnings",
    ]
    result = subprocess.run(
        command,
        cwd=repo,
        env=env,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
    )
    if result.returncode != 0:
        sys.stderr.write(result.stdout)
        sys.stderr.write(result.stderr)
        raise SystemExit(result.returncode)
    return result


def ffmpeg_path(python: str) -> str:
    script = "import imageio_ffmpeg; print(imageio_ffmpeg.get_ffmpeg_exe())"
    result = subprocess.run(
        [python, "-c", script],
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
    )
    if result.returncode != 0:
        sys.stderr.write(result.stderr)
        raise SystemExit(result.returncode)
    return result.stdout.strip()


def extract_audio(python: str, video_path: Path, audio_path: Path) -> None:
    audio_path.parent.mkdir(parents=True, exist_ok=True)
    command = [
        ffmpeg_path(python),
        "-y",
        "-i",
        str(video_path),
        "-vn",
        "-acodec",
        "pcm_s16le",
        "-ar",
        "16000",
        "-ac",
        "1",
        str(audio_path),
        "-loglevel",
        "error",
    ]
    result = subprocess.run(command, capture_output=True, text=True, encoding="utf-8", errors="replace")
    if result.returncode != 0:
        sys.stderr.write(result.stderr)
        raise SystemExit(result.returncode)


def build_config(output_dir: Path) -> str:
    normalized = output_dir.as_posix().rstrip("/") + "/"
    return f"""link: []
path: "{normalized}"
mode:
  - post
number:
  post: 1
  like: 0
  allmix: 0
  mix: 0
  music: 0
  collect: 0
  collectmix: 0
thread: 1
retry_times: 1
rate_limit: 1
proxy: ""
database: false
music: false
cover: false
avatar: false
json: true
auto_cookie: true
browser_fallback:
  enabled: false
progress:
  quiet_logs: true
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Download Douyin media with jiji downloader.")
    parser.add_argument("url")
    args = parser.parse_args()

    repo_value = os.environ.get("JIJI_REPO")
    if not repo_value:
        raise SystemExit("JIJI_REPO is required")
    repo = Path(repo_value).expanduser().resolve()
    python = os.environ.get("JIJI_PYTHON") or sys.executable
    workdir = Path(os.environ.get("MEDIA_WORKDIR", "work/real")).expanduser().resolve()
    cookies_value = os.environ.get("DOUYIN_COOKIES") or os.environ.get("YTDLP_COOKIES")
    if not cookies_value:
        raise SystemExit("DOUYIN_COOKIES or YTDLP_COOKIES is required")

    output_dir = workdir / "Downloaded"
    config_dir = workdir / "jiji-config"
    cookies = load_cookie_file(Path(cookies_value).expanduser())
    if not cookies:
        raise SystemExit("cookie file did not contain any cookies")

    cookies_json = config_dir / ".cookies.json"
    config = config_dir / "config.yml"
    write_text(cookies_json, json.dumps(cookies, ensure_ascii=False, indent=2))
    write_text(config, build_config(output_dir))

    resolved_url = resolve_douyin_input(args.url)
    jiji_result = run_jiji(repo, python, config, resolved_url, output_dir)
    try:
        video_path = find_newest_mp4(output_dir)
    except FileNotFoundError:
        sys.stderr.write(jiji_result.stdout)
        sys.stderr.write(jiji_result.stderr)
        raise
    audio_path = workdir / "audio.wav"
    extract_audio(python, video_path, audio_path)
    payload = {"resolved_url": resolved_url, "video_path": str(video_path), "audio_path": str(audio_path)}
    payload.update(extract_metadata(output_dir))
    print(json.dumps(payload, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
