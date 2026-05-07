package clientip

import (
	"net"
	"net/http"
	"strings"
)

// FromRequest returns the best client IP for deployments behind Google Cloud Run.
// Cloud Run appends the real client to X-Forwarded-For, so the rightmost value is
// the trusted one there. X-Real-IP is preferred when present.
func FromRequest(r *http.Request) string {
	if ip := cleanIP(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return cleanIP(parts[len(parts)-1])
	}

	return cleanIP(r.RemoteAddr)
}

func cleanIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if host, _, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(host, "[]")
	}

	return strings.Trim(value, "[]")
}
