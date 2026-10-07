# EC2 production host

Templates used by the live deployment:

- Frontend: https://rtsp.vishvajeetshukla.in
- API: https://rtsp-api.vishvajeetshukla.in

Install `websocket-map.conf` into `/etc/nginx/conf.d/` (http context) and `nginx.conf` as `/etc/nginx/sites-available/rtsp-stream-viewer`. Copy `.env.example` to `/home/ubuntu/rtsp-stream-viewer/.env` and set `API_TOKEN` plus camera hosts. Copy the unit file to `/etc/systemd/system/rtsp-stream-viewer.service`.

Build the frontend with:

```sh
VITE_API_URL=https://rtsp-api.vishvajeetshukla.in \
VITE_WS_URL=wss://rtsp-api.vishvajeetshukla.in \
npm run build
```

Place `frontend/dist` at `/var/www/rtsp-viewer` and the Go binary at `/home/ubuntu/rtsp-stream-viewer/bin/server`.
