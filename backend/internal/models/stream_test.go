package models

import (
	"strings"
	"testing"
)

func TestValidateRTSPURL(t *testing.T) {
	for _, raw := range []string{"rtsp://localhost:8554/test", "rtsps://camera.example/live", "rtsp://admin:p%40ss@192.168.0.2:554/live", "rtsp://[::1]:8554/test"} {
		if _, err := ValidateRTSPURL(raw); err != nil {
			t.Errorf("valid URL %q: %v", raw, err)
		}
	}
	for _, raw := range []string{"", "https://camera/live", "file:///etc/passwd", "rtsp:///live", "rtsp://camera:99999/live", "rtsp://camera:0/live", "rtsp://camera/#fragment", "rtsp://camera/live\n", "rtsp://-camera/live", "rtsp://camera/%zz", "rtsp://:password@camera/live"} {
		if _, err := ValidateRTSPURL(raw); err == nil {
			t.Errorf("invalid URL accepted: %q", raw)
		}
	}
}

func TestSanitizeRTSPURL(t *testing.T) {
	raw := "rtsp://admin:secret%40password@camera:554/live?token=private&session=also-private"
	masked := SanitizeRTSPURL(raw)
	for _, secret := range []string{"admin", "secret", "password", "private"} {
		if strings.Contains(masked, secret) {
			t.Errorf("credential leaked: %s", masked)
		}
	}
	if !strings.Contains(masked, "camera:554/live") {
		t.Fatal("lost public camera address")
	}
	if SanitizeRTSPURL("rtsp://%zz") != "[invalid RTSP URL]" {
		t.Fatal("invalid URL sanitization failed")
	}
}
