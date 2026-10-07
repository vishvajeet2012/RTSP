# Go streaming backend

Run from this folder with Go 1.25+ and FFmpeg on PATH:

```sh
cp .env.example .env
go mod download
go run ./cmd/server
```

Windows PowerShell: `Copy-Item .env.example .env` and `go run ./cmd/server`.

The server listens on `PORT` (default 8080), starts one FFmpeg process per unique RTSP URL, and broadcasts MPEG-TS binary messages to bounded WebSocket subscribers. Streaming state is in memory. All shutdown, restart, and remove operations cancel and reap their FFmpeg process before allowing a replacement.

`GET /api/health` is public. Other API endpoints require `Authorization: Bearer <API_TOKEN>` when configured. Browser clients exchange this for a single-use, 60-second WebSocket ticket using `POST /api/streams/{id}/ticket`. Request logging excludes query strings and body content.

```sh
go fmt ./...
go vet ./...
go test -race ./...
go build -o bin/server ./cmd/server
```

Real integration test with a running MediaMTX publisher:

```sh
TEST_RTSP_URL=rtsp://localhost:8554/test go test -race ./...
```

Build `Dockerfile` from this backend directory for real cameras, or build `backend/Dockerfile.demo` from the repository root for the complete React/Go app with two generated test feeds. Both contain FFmpeg; the demo also contains MediaMTX, the built frontend, and a signal-aware supervisor. Set `APP_ENV=production`, an API token of at least 24 characters, allowed frontend origins, and an exact camera hostname allowlist when hosting publicly. On Render, omitted origins use its exact `RENDER_EXTERNAL_URL`.

RTSPS requires a GnuTLS-enabled FFmpeg build whose RTSP demuxer exposes `tls_verify` and `verifyhost`. The supplied Ubuntu 26.04 images provide modern FFmpeg. Peer and hostname checks are mandatory; older builds that silently ignore TLS options fail with a clear error. Configure a private camera CA in the backend operating system's trusted certificate store rather than disabling verification.

See the [root README](../README.md) for API routes, local Docker networking, test publishers, frontend configuration, deployment, and limitations.
