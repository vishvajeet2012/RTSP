# Code review

Reviewed on 7 October 2026. Five issues were fixed, and the complete React/Go demo was prepared for one container and one public origin. Public hosting remains pending; no deployed URL is claimed.

## Fixed findings

| Severity | Problem and resulting change                                                                                                                                                                                                                                                                                                                                                      | Source                                                                                                    | Validation                                                                                                                                                                                                                                                                                                                    |
| -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| High     | FFmpeg 6.1 accepted an untrusted RTSPS certificate even when a TLS verification flag was supplied. Older RTSP demuxers can silently ignore that flag. RTSPS now requires GnuTLS and TLS controls exposed by the RTSP demuxer, requests peer and hostname verification explicitly, and fails immediately on unsupported builds. Runtime images use Ubuntu 26.04 for modern FFmpeg. | `backend/internal/services/ffmpeg_service.go`, `tls_options.go`, `stream_service.go`, backend Dockerfiles | Reproduced the original behavior against a local self-signed TLS server. Regression tests verify the capability gate and required process arguments. The production-mode native app rejects its FFmpeg 6.1 RTSPS input without retries. Successful verified-camera interoperability with the new container is still untested. |
| Medium   | A WebSocket handshake could pass the stream-status check, arrive after Stop disconnected the current viewers, and remain subscribed to a stopped source. Stop and terminal errors now atomically disable hub subscriptions and disconnect viewers. Start re-enables the hub; a removed hub stays closed.                                                                          | `backend/internal/websocket/hub.go`, `backend/internal/services/stream_service.go`                        | Regression test confirms late subscriptions are rejected, existing viewers close, restart resumes subscriptions, and removed hubs cannot reopen. Lifecycle and fan-out checks pass with the race detector.                                                                                                                    |
| Medium   | A diagnostic line longer than the scanner limit stopped stderr consumption. FFmpeg could then block on its full diagnostic pipe and stop producing video. Remaining diagnostics are now drained and discarded after a scanner error.                                                                                                                                              | `backend/internal/services/ffmpeg_service.go`                                                             | A helper process writes 256 KiB to stderr before MPEG-TS packets. The service receives its video and reaps the child within the test deadline.                                                                                                                                                                                |
| Medium   | A listener/bind failure logged an error but exited successfully, hiding startup failure from hosting supervisors. Server cleanup now returns the failure and main exits with status 1.                                                                                                                                                                                            | `backend/cmd/server/main.go`                                                                              | Bound the requested port, launched the original binary and reviewed binary, and observed exit codes 0 and 1 respectively.                                                                                                                                                                                                     |
| Low      | Standalone percent-encoded query tokens could remain in diagnostic text after decoded values were masked. Redaction now covers decoded, escaped, and original encoded query values.                                                                                                                                                                                               | `backend/internal/services/ffmpeg_service.go`                                                             | Regression coverage includes upper- and lower-case percent escapes and decoded values. Actual API responses and debug logs contain neither sentinel RTSP secrets nor the production access token.                                                                                                                             |

## Deployment preparation

- The demo Dockerfile now builds React and Go from repository-root context, bundles MediaMTX and two generated RTSP publishers, and serves the frontend through Go using `FRONTEND_DIR`.
- Frontend assets remain public; API actions retain bearer-token protection and WebSockets retain single-use tickets. Static directory listings and hidden files are rejected.
- When explicit origins are omitted, Render's `RENDER_EXTERNAL_URL` supplies the exact allowed origin. Custom domains and separate frontends can override it through `ALLOWED_ORIGINS`.
- `render.yaml` explicitly selects the Free plan and generates the access token. The source does not publish or expose that token in JavaScript.
- Browser tests can target an existing isolated server using `TEST_BASE_URL`, `TEST_API_URL`, and `TEST_API_TOKEN`, including the production authentication flow.
- CI now builds both the normal backend image and the complete demo image.

## Executed checks

| Check                                                             | Result                                 |
| ----------------------------------------------------------------- | -------------------------------------- |
| Strict frontend TypeScript, ESLint, and production build          | Passed                                 |
| Frontend utility tests                                            | 4 passed                               |
| Go vet, formatting, and executable build                          | Passed                                 |
| Go unit tests with race detector                                  | 19 passed                              |
| Real RTSP-to-WebSocket integration with race detector             | Passed separately                      |
| Authenticated production frontend and protected API on one origin | Passed natively                        |
| Browser playback for two real generated RTSP feeds                | Passed                                 |
| Viewer Pause/Play, source Restart/Stop/Start, Remove              | Passed                                 |
| Layout and overflow at 1920, 1440, 1024, 768, and 375 pixels      | Passed                                 |
| Browser JavaScript errors                                         | None during the successful workflow    |
| Unreachable camera retries, error state, and removal              | Passed                                 |
| Unsupported RTSPS build fails immediately                         | Passed                                 |
| Access-token and RTSP-secret redaction in captured logs           | Passed                                 |
| Demo supervisor SIGTERM cleanup                                   | Passed; HTTP and RTSP listeners closed |
| Render/Compose/CI YAML and publisher/supervisor shell syntax      | Passed                                 |
| Referenced Ubuntu 26.04 and Node 22 image manifests               | Registry returned 200                  |

The successful browser workflow used Chromium 141 through Playwright 1.56.1 and completed in 18.2 seconds. Two publishers sent actual H.264 test patterns through MediaMTX, FFmpeg produced MPEG-1/MPEG-TS, and JSMpeg decoded the WebSocket bytes into canvas pixels. Playback was not simulated.

## Remaining verification and limits

- A connected Render account and target source repository are required to publish. Neither was available for this review, and the source has not been pushed to GitHub.
- This environment has no Docker engine. The complete supervisor/frontend/backend flow was exercised natively; the new Ubuntu-based images still need a real image build and container test. CI is configured to run those checks but has not run remotely.
- The native host has FFmpeg 6.1. Its unverified RTSPS behavior is blocked. Successful RTSPS with modern FFmpeg, trusted/private camera CAs, and physical-camera interoperability still need execution in the target environment.
- Render Free-plan CPU capacity, cold starts, and actual public HTTPS/WSS playback remain unmeasured.
- Streams are in memory and the application uses one shared workspace/admin token. Host allowlists require complementary network egress controls for untrusted camera inputs. These documented architecture limits remain unchanged.

See [VERIFICATION.md](VERIFICATION.md) for the overall workflow record and the [README](../README.md) for deployment and runtime configuration.
