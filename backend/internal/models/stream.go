package models

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Status string

const (
	Idle         Status = "idle"
	Connecting   Status = "connecting"
	Live         Status = "live"
	Reconnecting Status = "reconnecting"
	Error        Status = "error"
	Stopped      Status = "stopped"
)

// Only this public snapshot is serialized. The original URL lives in a private session.
type Stream struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	RTSPURL          string     `json:"rtspUrl"`
	Hostname         string     `json:"hostname"`
	Status           Status     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
	StartedAt        *time.Time `json:"startedAt,omitempty"`
	LastError        string     `json:"lastError,omitempty"`
	ReconnectAttempt int        `json:"reconnectAttempt"`
	Viewers          int        `json:"viewers"`
}

func ValidateRTSPURL(raw string) (*url.URL, error) {
	if len(raw) == 0 || len(raw) > 4096 || strings.IndexFunc(raw, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return nil, fmt.Errorf("enter an RTSP URL without spaces (maximum 4096 characters)")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid RTSP URL; percent-encode special characters in credentials")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "rtsp" && u.Scheme != "rtsps" {
		return nil, fmt.Errorf("only rtsp:// and rtsps:// URLs are supported")
	}
	if u.Hostname() == "" || u.Opaque != "" || u.Fragment != "" {
		return nil, fmt.Errorf("RTSP URL must have a hostname and no fragment")
	}
	if net.ParseIP(u.Hostname()) == nil {
		for _, label := range strings.Split(u.Hostname(), ".") {
			if len(label) == 0 || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
				return nil, fmt.Errorf("invalid RTSP hostname")
			}
			for _, r := range label {
				if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
					return nil, fmt.Errorf("invalid RTSP hostname")
				}
			}
		}
	}
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("RTSP port must be between 1 and 65535")
		}
	}
	if u.User != nil && u.User.Username() == "" {
		return nil, fmt.Errorf("RTSP username cannot be empty")
	}
	if _, err := url.ParseQuery(u.RawQuery); err != nil {
		return nil, fmt.Errorf("invalid RTSP query parameters")
	}
	u.Host = strings.ToLower(u.Host)
	return u, nil
}

func SanitizeRTSPURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "[invalid RTSP URL]"
	}
	if u.User != nil {
		u.User = url.UserPassword("****", "****")
	}
	q := u.Query()
	for key := range q {
		q.Set(key, "****")
	}
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String()
}

func ValidateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if len([]rune(name)) > 80 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("stream name must be at most 80 characters with no control characters")
	}
	return name, nil
}
