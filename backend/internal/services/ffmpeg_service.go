package services

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"rtspviewer/internal/models"
)

var ErrFFmpegUnavailable = errors.New("FFmpeg is not available on the backend")
var ErrRTSPTLSUnavailable = errors.New("RTSPS requires an FFmpeg build with GnuTLS and RTSP TLS verification controls")

type Runner interface {
	Check() error
	Run(context.Context, string, func([]byte)) error
}

type FFmpegService struct {
	path    string
	timeout time.Duration
	log     *slog.Logger
	tlsOnce sync.Once
	tlsErr  error
}

func NewFFmpegService(path string, timeout time.Duration, log *slog.Logger) *FFmpegService {
	return &FFmpegService{path: path, timeout: timeout, log: log}
}
func (f *FFmpegService) Check() error {
	if _, err := exec.LookPath(f.path); err != nil {
		return ErrFFmpegUnavailable
	}
	return nil
}

func (f *FFmpegService) Run(parent context.Context, rawURL string, onData func([]byte)) error {
	if err := f.Check(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	args := []string{
		"-hide_banner", "-nostdin", "-loglevel", "warning",
		"-rtsp_transport", "tcp", "-timeout", fmt.Sprint(f.timeout.Microseconds()),
		"-protocol_whitelist", "rtsp,rtsps,tcp,tls,crypto",
	}
	if strings.HasPrefix(rawURL, "rtsps://") {
		if err := f.checkRTSPTLS(); err != nil {
			return err
		}
		u, _ := url.Parse(rawURL)
		args = append(args, "-tls_verify", "1", "-verifyhost", u.Hostname())
	}
	args = append(args,
		"-probesize", "1000000", "-analyzeduration", "1000000", "-i", rawURL,
		"-map", "0:v:0", "-an", "-sn", "-dn",
		"-vf", "scale=min(960\\,iw):-2", "-c:v", "mpeg1video", "-pix_fmt", "yuv420p",
		"-b:v", "1200k", "-maxrate", "1800k", "-bufsize", "1800k",
		"-r", "25", "-bf", "0", "-g", "25", "-threads", "1",
		"-f", "mpegts", "-mpegts_flags", "resend_headers", "-muxdelay", "0.001", "-flush_packets", "1", "pipe:1",
	)
	cmd := exec.CommandContext(ctx, f.path, args...)
	cmd.WaitDelay = 3 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create FFmpeg output: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		return fmt.Errorf("create FFmpeg diagnostics: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		return fmt.Errorf("start FFmpeg: %w", err)
	}
	var lastOutput atomic.Int64
	lastOutput.Store(time.Now().UnixNano())
	var timedOut atomic.Bool
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if time.Since(time.Unix(0, lastOutput.Load())) >= f.timeout {
					timedOut.Store(true)
					cancel()
					return
				}
			}
		}
	}()
	stderrDone := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 4096), 64*1024)
		tail := ""
		for scanner.Scan() {
			line := sanitizeDiagnostic(scanner.Text(), rawURL)
			tail += line + "\n"
			if len(tail) > 8192 {
				tail = tail[len(tail)-8192:]
			}
		}
		// Scanner rejects oversized lines. Continue draining so stderr backpressure cannot
		// block FFmpeg's video output; oversized diagnostics are intentionally discarded.
		if scanner.Err() != nil {
			_, _ = io.Copy(io.Discard, stderr)
		}
		stderrDone <- tail
	}()
	buffer := make([]byte, 188*32)
	for {
		n, readErr := io.ReadFull(stdout, buffer)
		if n > 0 {
			lastOutput.Store(time.Now().UnixNano())
			// Each immutable chunk is shared across subscribers; the read buffer is reused.
			complete := n / 188 * 188
			if complete > 0 {
				onData(append([]byte(nil), buffer[:complete]...))
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				cancel()
			}
			break
		}
	}
	diagnostics := <-stderrDone
	err = cmd.Wait() // Always reap, including cancellation and startup/data timeouts.
	cancel()
	<-watchDone
	if parent.Err() != nil {
		return parent.Err()
	}
	if diagnostics != "" {
		f.log.Debug("ffmpeg diagnostics", "output", strings.TrimSpace(diagnostics))
	}
	if timedOut.Load() {
		return fmt.Errorf("RTSP stream timed out without video data")
	}
	if err != nil {
		lower := strings.ToLower(diagnostics)
		if strings.Contains(lower, "401") || strings.Contains(lower, "unauthorized") {
			return fmt.Errorf("camera rejected the RTSP credentials")
		}
		return fmt.Errorf("unable to read RTSP video; check camera availability, credentials and network access")
	}
	return fmt.Errorf("RTSP input ended")
}

var diagnosticURL = regexp.MustCompile(`(?i)rtsps?://[^\s'"<>]+`)

func sanitizeDiagnostic(message, rawURL string) string {
	message = strings.ReplaceAll(message, rawURL, models.SanitizeRTSPURL(rawURL))
	message = diagnosticURL.ReplaceAllStringFunc(message, models.SanitizeRTSPURL)
	u, err := url.Parse(rawURL)
	if err == nil {
		if u.User != nil {
			password, _ := u.User.Password()
			for _, secret := range []string{password, url.QueryEscape(password), url.PathEscape(password)} {
				if secret != "" {
					message = strings.ReplaceAll(message, secret, "[redacted]")
				}
			}
		}
		for _, values := range u.Query() {
			for _, secret := range values {
				for _, encoded := range []string{secret, url.QueryEscape(secret), url.PathEscape(secret)} {
					if encoded != "" {
						message = strings.ReplaceAll(message, encoded, "[redacted]")
					}
				}
			}
		}
		for _, field := range strings.Split(u.RawQuery, "&") {
			if _, encoded, ok := strings.Cut(field, "="); ok && encoded != "" {
				message = strings.ReplaceAll(message, encoded, "[redacted]")
			}
		}
	}
	return message
}
