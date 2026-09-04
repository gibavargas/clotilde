// Package clientip extracts the client IP for deployments behind Google Cloud
// Run. Clients can set arbitrary X-Forwarded-For and X-Real-IP headers, so a
// header is only trusted when the platform itself is known to have appended
// it: on Cloud Run (detected via the K_SERVICE environment variable, which is
// container-scoped and client-invisible) the Google Front End appends the real
// client address as the rightmost X-Forwarded-For entry; anywhere else the
// only honest source is the socket address.
package clientip

import (
	"net"
	"net/http"
	"os"
	"strings"
)

// runningOnCloudRun reports whether this process runs in a Cloud Run
// container. K_SERVICE is set by the Cloud Run container environment itself;
// unlike HTTP headers, clients cannot influence it.
func runningOnCloudRun() bool {
	return os.Getenv("K_SERVICE") != ""
}

// FromRequest returns the client IP for rate limiting, lockouts, and audit
// logs. On Cloud Run: the rightmost parseable X-Forwarded-For entry (GFE's
// appended value; client-supplied junk entries are skipped by parse
// validation). Outside Cloud Run: RemoteAddr. X-Real-IP is deliberately never
// trusted — no proxy in this deployment path provisions it, and it is
// trivially spoofable.
func FromRequest(r *http.Request) string {
	if runningOnCloudRun() {
		if ip := rightmostForwardedFor(r.Header.Get("X-Forwarded-For")); ip != "" {
			return ip
		}
	}
	return cleanIP(r.RemoteAddr)
}

// rightmostForwardedFor walks the XFF chain from the right and returns the
// first entry that parses as an IP address. The rightmost entry is the one
// the trusted platform appended; entries to its left are client-supplied.
func rightmostForwardedFor(forwarded string) string {
	if strings.TrimSpace(forwarded) == "" {
		return ""
	}
	parts := strings.Split(forwarded, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		if ip := cleanIP(parts[i]); ip != "" && net.ParseIP(ip) != nil {
			return ip
		}
	}
	return ""
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
