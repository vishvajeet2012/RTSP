package services

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

func (f *FFmpegService) checkRTSPTLS() error {
	f.tlsOnce.Do(func() {
		// Older RTSP demuxers silently ignore TLS protocol flags. Require controls on
		// the demuxer itself and a TLS backend that validates the camera hostname.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		help, err := exec.CommandContext(ctx, f.path, "-hide_banner", "-h", "demuxer=rtsp").CombinedOutput()
		if err != nil || !strings.Contains(string(help), "-tls_verify") || !strings.Contains(string(help), "-verifyhost") {
			f.tlsErr = ErrRTSPTLSUnavailable
			return
		}
		version, err := exec.CommandContext(ctx, f.path, "-hide_banner", "-version").CombinedOutput()
		if err != nil || !strings.Contains(string(version), "--enable-gnutls") {
			f.tlsErr = ErrRTSPTLSUnavailable
		}
	})
	return f.tlsErr
}
