package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	mode := os.Getenv("RTSP_TEST_FFMPEG_HELPER")
	if mode != "" {
		args := strings.Join(os.Args[1:], " ")
		if strings.Contains(args, "-h demuxer=rtsp") {
			if mode == "supported-tls" {
				_, _ = os.Stdout.WriteString("RTSP options: -tls_verify -verifyhost\n")
			}
			os.Exit(0)
		}
		if strings.Contains(args, "-version") {
			_, _ = os.Stdout.WriteString("ffmpeg configuration: --enable-gnutls\n")
			os.Exit(0)
		}
		if mode == "supported-tls" && !strings.Contains(args, "-tls_verify 1 -verifyhost localhost") {
			os.Exit(2)
		}
		if mode == "oversized-diagnostic" {
			_, _ = os.Stderr.Write(append(bytes.Repeat([]byte("x"), 256*1024), '\n'))
		}
		_, _ = os.Stdout.Write(bytes.Repeat(append([]byte{0x47}, make([]byte, 187)...), 32))
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestRTSPSRequiresVerificationControls(t *testing.T) {
	t.Setenv("RTSP_TEST_FFMPEG_HELPER", "unsupported-tls")
	runner := NewFFmpegService(os.Args[0], time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	err := runner.Run(context.Background(), "rtsps://localhost/live", func([]byte) { t.Fatal("unverified camera output was accepted") })
	if !errors.Is(err, ErrRTSPTLSUnavailable) {
		t.Fatalf("unsupported TLS build was accepted: %v", err)
	}
}

func TestRTSPSRequestsPeerAndHostnameVerification(t *testing.T) {
	t.Setenv("RTSP_TEST_FFMPEG_HELPER", "supported-tls")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	runner := NewFFmpegService(os.Args[0], 10*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	bytesReceived := 0
	_ = runner.Run(ctx, "rtsps://localhost/live", func(data []byte) { bytesReceived += len(data) })
	if bytesReceived != 188*32 || ctx.Err() != nil {
		t.Fatal("RTSPS was launched without its required TLS and hostname flags")
	}
}

func TestFFmpegDrainsOversizedDiagnostics(t *testing.T) {
	t.Setenv("RTSP_TEST_FFMPEG_HELPER", "oversized-diagnostic")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	runner := NewFFmpegService(os.Args[0], 10*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	bytesReceived := 0
	_ = runner.Run(ctx, "rtsp://localhost/live", func(data []byte) { bytesReceived += len(data) })
	if bytesReceived != 188*32 || ctx.Err() != nil {
		t.Fatalf("stderr blocked video output: received %d bytes, context error %v", bytesReceived, ctx.Err())
	}
}

func TestDiagnosticRedaction(t *testing.T) {
	raw := "rtsp://admin:s3cr%40t@camera/live?token=query-secret"
	message := "failed to open " + raw + " decoded password=s3cr@t token=query-secret"
	clean := sanitizeDiagnostic(message, raw)
	for _, secret := range []string{"s3cr", "query-secret"} {
		if strings.Contains(clean, secret) {
			t.Fatalf("diagnostic leaked a secret: %s", clean)
		}
	}
}

func TestEncodedQueryDiagnosticRedaction(t *testing.T) {
	for _, encoded := range []string{"secret%2Fvalue", "secret%2fvalue", "secret/value"} {
		clean := sanitizeDiagnostic("token="+encoded, "rtsp://camera/live?token="+encoded)
		if strings.Contains(clean, "secret") {
			t.Fatalf("diagnostic leaked an encoded query token: %s", clean)
		}
	}
}
