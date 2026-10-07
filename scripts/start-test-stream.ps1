param(
    [string]$Url = "rtsp://localhost:8554/test",
    [string]$VideoFile = ""
)
$ErrorActionPreference = "Stop"
if (-not (Get-Command ffmpeg -ErrorAction SilentlyContinue)) {
    throw "Install FFmpeg and add it to PATH."
}
if ($VideoFile) {
    if (-not (Test-Path -LiteralPath $VideoFile -PathType Leaf)) { throw "Video file not found." }
    $inputOptions = @("-re", "-stream_loop", "-1", "-i", $VideoFile)
} else {
    $inputOptions = @("-re", "-f", "lavfi", "-i", "testsrc2=size=960x540:rate=25")
}
$ffmpegArgs = @("-hide_banner", "-loglevel", "warning") + $inputOptions + @(
    "-an", "-c:v", "libx264", "-preset", "ultrafast", "-tune", "zerolatency", "-pix_fmt", "yuv420p",
    "-r", "25", "-g", "25", "-threads", "2", "-rtsp_transport", "tcp", "-f", "rtsp", $Url
)
Write-Host "Publishing test video. Keep this terminal open; Ctrl+C stops the publisher."
& ffmpeg @ffmpegArgs
exit $LASTEXITCODE
