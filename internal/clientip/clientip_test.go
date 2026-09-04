package clientip

import (
	"net/http/httptest"
	"testing"
)

// withCloudRunEnv simulates a Cloud Run container (K_SERVICE is set by the
// platform and cannot be spoofed by clients).
func withCloudRunEnv(t *testing.T) {
	t.Helper()
	t.Setenv("K_SERVICE", "clotilde")
}

func TestFromRequestCloudRunUsesRightmostForwardedFor(t *testing.T) {
	withCloudRunEnv(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.10")
	req.RemoteAddr = "10.0.0.1:1234"

	if got := FromRequest(req); got != "203.0.113.10" {
		t.Fatalf("expected rightmost forwarded IP, got %q", got)
	}
}

// The attacker-controlled X-Real-IP header must never decide the result:
// Cloud Run's GFE does not set or strip it, so trusting it defeats every
// IP-based defense (pre-auth limiter, admin lockout, CSRF binding).
func TestFromRequestIgnoresXRealIP(t *testing.T) {
	withCloudRunEnv(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "203.0.113.10")
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 198.51.100.2")
	req.RemoteAddr = "10.0.0.1:1234"

	if got := FromRequest(req); got != "198.51.100.2" {
		t.Fatalf("X-Real-IP must be ignored; expected rightmost XFF, got %q", got)
	}
}

func TestFromRequestLocalFallsBackToRemoteAddrEvenWithXFF(t *testing.T) {
	// No K_SERVICE: not a Cloud Run container, so XFF is client-supplied and
	// untrusted — the socket address is the only honest source.
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	req.RemoteAddr = "10.0.0.1:1234"

	if got := FromRequest(req); got != "10.0.0.1" {
		t.Fatalf("expected RemoteAddr fallback, got %q", got)
	}
}

func TestFromRequestCloudRunFallsBackToRemoteAddrWithoutXFF(t *testing.T) {
	withCloudRunEnv(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"

	if got := FromRequest(req); got != "10.0.0.1" {
		t.Fatalf("expected RemoteAddr fallback, got %q", got)
	}
}

func TestFromRequestPreservesIPv6RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[2001:db8::1]:443"

	if got := FromRequest(req); got != "2001:db8::1" {
		t.Fatalf("expected IPv6 host without port, got %q", got)
	}
}

func TestFromRequestCloudRunRightmostXFFIPv6(t *testing.T) {
	withCloudRunEnv(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "2001:db8::1, 2001:db8::2")
	req.RemoteAddr = "10.0.0.1:1234"

	if got := FromRequest(req); got != "2001:db8::2" {
		t.Fatalf("expected rightmost IPv6 XFF entry, got %q", got)
	}
}

func TestFromRequestSkipsGarbageXFFEntries(t *testing.T) {
	withCloudRunEnv(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "not-an-ip, 203.0.113.10")
	req.RemoteAddr = "10.0.0.1:1234"

	if got := FromRequest(req); got != "203.0.113.10" {
		t.Fatalf("expected last parseable XFF entry, got %q", got)
	}
}

func TestCloudRunDetectionFollowsEnv(t *testing.T) {
	if runningOnCloudRun() {
		t.Fatal("K_SERVICE unset: must not detect Cloud Run")
	}
	t.Setenv("K_SERVICE", "clotilde")
	if !runningOnCloudRun() {
		t.Fatal("K_SERVICE set: must detect Cloud Run")
	}
}
