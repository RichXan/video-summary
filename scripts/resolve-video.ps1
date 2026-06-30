param(
    [Parameter(Mandatory = $true)]
    [string]$Url
)

$ErrorActionPreference = 'Stop'

$ytDlp = if ($env:YTDLP_BIN) { $env:YTDLP_BIN } else { 'yt-dlp' }
$command = Get-Command $ytDlp -ErrorAction SilentlyContinue
if (-not $command) {
    Write-Error "yt-dlp was not found. Install yt-dlp or set YTDLP_BIN."
    exit 127
}

$ytArgs = @('-J', '--no-playlist')
if ($env:YTDLP_COOKIES) {
    $ytArgs += @('--cookies', $env:YTDLP_COOKIES)
}
if ($env:YTDLP_COOKIES_FROM_BROWSER) {
    $ytArgs += @('--cookies-from-browser', $env:YTDLP_COOKIES_FROM_BROWSER)
}
$ytArgs += $Url

& $command.Source @ytArgs
exit $LASTEXITCODE
