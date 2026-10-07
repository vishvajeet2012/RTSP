#!/usr/bin/env bash
set -eu
app_dir="${APP_DIR:-/app}"
children=()
cleanup() {
  trap - EXIT INT TERM
  for child in "${children[@]}"; do kill -TERM "$child" 2>/dev/null || true; done
  wait || true
}
trap cleanup EXIT INT TERM
mediamtx "$app_dir/mediamtx.yml" & children+=("$!")
publish() {
  local path="$1" size="$2"
  while true; do
    ffmpeg -hide_banner -loglevel error -re -f lavfi -i "testsrc2=size=${size}:rate=25" \
      -an -c:v libx264 -preset ultrafast -tune zerolatency -pix_fmt yuv420p \
      -r 25 -g 25 -threads 2 -rtsp_transport tcp -f rtsp "rtsp://127.0.0.1:8554/${path}" || true
    sleep 2
  done
}
publish test 960x540 & children+=("$!")
publish test2 640x360 & children+=("$!")
"$app_dir/server" & children+=("$!")
# If any long-running child exits, stop the others. tini forwards signals to the whole process group.
wait -n "${children[@]}"
