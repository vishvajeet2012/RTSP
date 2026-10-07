# Verification report

Original verification was completed on 6 October 2026. A review and production-mode verification pass was completed on 7 October 2026 using Node.js 24.19.0, Go 1.26.8, native FFmpeg 6.1.1, MediaMTX 1.21.1 and Chromium 141 through Playwright 1.56.1. See [CODE_REVIEW.md](CODE_REVIEW.md) for five fixed findings and their evidence.

## Completed checks

| Check                                               | Result                                                                                                                          |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Strict frontend TypeScript compilation              | Passed                                                                                                                          |
| Frontend ESLint                                     | Passed; no warnings or errors                                                                                                   |
| Frontend utility tests                              | 4 passed                                                                                                                        |
| Frontend production build                           | Passed; Vite 7.3.7                                                                                                              |
| Go formatting                                       | Applied to all Go files                                                                                                         |
| Go vet                                              | Passed                                                                                                                          |
| Go unit tests with race detector                    | 19 passed; optional real-stream test runs separately                                                                            |
| Go executable build                                 | Passed                                                                                                                          |
| Real RTSP integration with race detector            | Passed                                                                                                                          |
| Real browser workflow                               | Passed against the built frontend and protected API on one origin in production mode                                            |
| Docker Compose configuration                        | Passed official Compose JSON Schema 2020-12 validation                                                                          |
| MediaMTX / Render YAML                              | Parsed successfully; native MediaMTX configuration launched successfully                                                        |
| POSIX test-publisher and demo startup script syntax | Passed `bash -n`                                                                                                                |
| Container image tags                                | FFmpeg 7.1-alpine, MediaMTX 1.21.1, Go 1.26-bookworm, Ubuntu 26.04, and Node 22-bookworm-slim manifests exist in their registry |
| FFmpeg availability                                 | Actual execution and real transcoding confirmed                                                                                 |

The host's Go vet executable initially had an executable-permission issue when automatically selected by Go. It was run explicitly with `go vet -vettool=...`, and race tests were run with `-vet=off` after that separate vet check. No source workaround or test bypass is needed in the supplied project. Standard `go vet ./...` and `go test -race ./...` are the documented user/CI commands.

## Genuine streaming checks

Two FFmpeg test-pattern publishers pushed H.264 RTSP into native MediaMTX. The actual Go FFmpeg service read those streams, encoded MPEG-1/MPEG-TS and sent binary messages over Gorilla WebSockets. No mock playback source was used.

The Go integration test confirmed:

- Two WebSocket subscribers received actual MPEG-TS packets from one FFmpeg process.
- Idempotent Start did not start another process.
- Stop cancelled and reaped the process.
- Restart created exactly one replacement and resumed real output.
- Remove deleted metadata, closed subscribers and left no active runner.

The browser test confirmed:

- Clean invalid-URL feedback and an initial empty workspace.
- Two independent stream cards displaying live video, with hundreds of distinct decoded canvas colors.
- Viewer Pause held the last frame while the other camera stayed live.
- Play resumed live video; Restart, source Stop/Start and Remove worked.
- Correct two-column / one-column layouts and no horizontal overflow at 1920, 1440, 1024, 768 and 375 pixels.
- No browser JavaScript errors.
- Desktop and mobile screenshots were captured from real playback.

An additional live exercise added an unreachable RTSP camera with sentinel credentials and a query token. It exhausted its configured two-retry budget, reached Error, left the backend healthy, and could be removed. Neither API snapshots nor captured debug logs contained either sentinel secret. The final stream list was empty and the native test processes were shut down.

The review pass exercised `backend/deploy/run-demo.sh` with its actual MediaMTX process, two publisher loops, Go server, and built React files. Production token protection, the Access dialog, one-origin API calls, and ticket-authenticated video all worked. The supervisor received SIGTERM and both HTTP and RTSP listeners closed. A port-conflict exercise confirmed the reviewed server exits with status 1 rather than the original status 0.

A local self-signed TLS server reproduced FFmpeg 6.1's acceptance of untrusted RTSPS peers despite an explicit verification flag. The service now rejects builds lacking safe demuxer TLS controls, and a production-mode RTSPS attempt on that native build reached Error immediately with zero retries. Unit tests cover the capability gate and peer/hostname process arguments.

## Checks not performed here

- Full Docker image builds and container-based Compose execution: this environment has no Docker engine. The genuine pipeline was tested natively; YAML/schema, shell scripts and referenced image tags were checked. CI includes Docker build and Compose integration steps but has not yet been run on GitHub.
- Windows PowerShell execution: the Windows script is included, but this host is Linux.
- Public hosting and GitHub push: a Render connection was offered, but no connected container host or target source repository was available. The complete one-origin Render demo image, optional Vercel setup, and updated instructions are supplied. No public URL is claimed.
- Physical cameras, successful verified RTSPS interoperability on modern FFmpeg, private CA setup, audio and hardware acceleration: these remain untested. The supplied runtime images use Ubuntu 26.04; they still need a real container build. Browser preview is intentionally video-only.

This report separates executed checks from deployment preparation. It does not claim a published live URL or a completed container build.
