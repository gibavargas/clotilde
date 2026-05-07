package clientip

import (
	"net/http/httptest"
	"testing"
)

func TestFromRequestPrefersXRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "203.0.113.10")
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 198.51.100.2")
	req.RemoteAddr = "10.0.0.1:1234"

	if got := FromRequest(req); got != "203.0.113.10" {
		t.Fatalf("expected X-Real-IP, got %q", got)
	}
}

func TestFromRequestUsesRightmostForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.10")
	req.RemoteAddr = "10.0.0.1:1234"

	if got := FromRequest(req); got != "203.0.113.10" {
		t.Fatalf("expected rightmost forwarded IP, got %q", got)
	}
}

func TestFromRequestPreservesIPv6RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[2001:db8::1]:443"

	if got := FromRequest(req); got != "2001:db8::1" {
		t.Fatalf("expected IPv6 host without port, got %q", got)
	}
}
