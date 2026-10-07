#!/usr/bin/env bash
set -euo pipefail
target_url="${1:-rtsp://localhost:8554/test}"
video_file="${2:-}"
command -v ffmpeg >/dev/null || { echo 'Install FFmpeg and add it to PATH.' >&2; exit 1; }
if [[ -n "$video_file" ]]; then
  [[ -f "$video_file" ]] || { echo 'Video file not found.' >&2; exit 1; }
  input_options=(-re -stream_loop -1 -i "$video_file")
else
  input_options=(-re -f lavfi -i 'testsrc2=size=960x540:rate=25')
fi
echo 'Publishing test video. Keep this terminal open; Ctrl+C stops the publisher.'
exec ffmpeg -hide_banner -loglevel warning "${input_options[@]}" \
  -an -c:v libx264 -preset ultrafast -tune zerolatency -pix_fmt yuv420p \
  -r 25 -g 25 -threads 2 -rtsp_transport tcp -f rtsp "$target_url"
