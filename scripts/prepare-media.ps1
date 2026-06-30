param(
    [Parameter(Mandatory = $true)]
    [string]$Url
)

$ErrorActionPreference = 'Stop'

$ytDlp = if ($env:YTDLP_BIN) { $env:YTDLP_BIN } else { 'yt-dlp' }
$ffmpeg = if ($env:FFMPEG_BIN) { $env:FFMPEG_BIN } else { 'ffmpeg' }
$ytCommand = Get-Command $ytDlp -ErrorAction SilentlyContinue
if (-not $ytCommand) {
    Write-Error "yt-dlp was not found. Install yt-dlp or set YTDLP_BIN."
    exit 127
}
$ffmpegCommand = Get-Command $ffmpeg -ErrorAction SilentlyContinue
if (-not $ffmpegCommand) {
    Write-Error "ffmpeg was not found. Install ffmpeg or set FFMPEG_BIN."
    exit 127
}

$workDir = if ($env:MEDIA_WORKDIR) { $env:MEDIA_WORKDIR } else { Join-Path (Get-Location) 'work\real' }
New-Item -ItemType Directory -Force -Path $workDir | Out-Null

$outputTemplate = Join-Path $workDir 'video.%(ext)s'
$ytArgs = @(
    '--no-playlist',
    '--force-overwrites',
    '--merge-output-format', 'mp4',
    '--print', 'after_move:filepath',
    '-o', $outputTemplate
)
if ($env:YTDLP_COOKIES) {
    $ytArgs += @('--cookies', $env:YTDLP_COOKIES)
}
if ($env:YTDLP_COOKIES_FROM_BROWSER) {
    $ytArgs += @('--cookies-from-browser', $env:YTDLP_COOKIES_FROM_BROWSER)
}
$ytArgs += $Url

$downloadOutput = & $ytCommand.Source @ytArgs
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

$videoPath = ($downloadOutput | Where-Object { $_ -and $_.Trim() } | Select-Object -Last 1)
if (-not $videoPath) {
    Write-Error "yt-dlp did not print a downloaded video path."
    exit 1
}
$videoPath = [System.IO.Path]::GetFullPath($videoPath.Trim())
$audioPath = [System.IO.Path]::GetFullPath((Join-Path $workDir 'audio.wav'))

& $ffmpegCommand.Source -y -i $videoPath -vn -ac 1 -ar 16000 $audioPath | Out-Null
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

[ordered]@{
    video_path = $videoPath
    audio_path = $audioPath
} | ConvertTo-Json -Compress
