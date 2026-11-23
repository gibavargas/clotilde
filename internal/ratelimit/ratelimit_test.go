package ratelimit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clotilde/carplay-assistant/internal/auth"
)

// withCloudRun enables the trusted-proxy path (clientip gates header trust on
// Cloud Run's K_SERVICE env marker).
func withCloudRun(t *testing.T) {
	t.Helper()
	t.Setenv("K_SERVICE", "clotilde")
}

func TestMiddleware_AllowsFirstRequest(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-API-Key", "test-key-1")
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware_RateLimitExceeded(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	apiKey := "test-key-rate-limit"
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-API-Key", apiKey)
	rr := httptest.NewRecorder()

	// Make 11 requests (limit is 10 per minute)
	for i := 0; i < 11; i++ {
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rr.Code)
	}

	expectedBody := `{"error":"Rate limit exceeded"}`
	actualBody := strings.TrimSpace(rr.Body.String())
	if actualBody != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, actualBody)
	}
}

func TestMiddleware_HealthCheckBypass(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware_AdminRouteBypass(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin/dashboard", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	withCloudRun(t)
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1, got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_MultipleIPs(t *testing.T) {
	withCloudRun(t)
	// Critical security test: X-Forwarded-For can contain multiple IPs
	// In Cloud Run: if attacker sends "X-Forwarded-For: 1.2.3.4", Cloud Run appends real IP
	// Result: "1.2.3.4, <Real-IP>". We must use the RIGHTmost IP (most recent, added by Cloud Run)
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1, 172.16.0.1")
	req.RemoteAddr = "127.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "172.16.0.1" {
		t.Errorf("Expected 172.16.0.1 (rightmost IP, added by Cloud Run), got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_WithPort(t *testing.T) {
	withCloudRun(t)
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1:8080")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1 (port removed), got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_IPv6(t *testing.T) {
	withCloudRun(t)
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "[2001:db8::1]:8080")
	req.RemoteAddr = "127.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "2001:db8::1" {
		t.Errorf("Expected 2001:db8::1 (normalized IPv6), got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_IPv6_Multiple(t *testing.T) {
	withCloudRun(t)
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "[2001:db8::1], [2001:db8::2]")
	req.RemoteAddr = "127.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "2001:db8::2" {
		t.Errorf("Expected 2001:db8::2 (rightmost normalized IPv6, added by Cloud Run), got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_Whitespace(t *testing.T) {
	withCloudRun(t)
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "  192.168.1.1  ,  10.0.0.1  ")
	req.RemoteAddr = "127.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("Expected 10.0.0.1 (rightmost IP after whitespace trimmed), got %s", ip)
	}
}

func TestGetClientIP_XRealIP_Ignored(t *testing.T) {
	// X-Real-IP is client-controlled and never provisioned by Cloud Run's
	// GFE, so it must be ignored entirely.
	withCloudRun(t)
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Real-IP", "192.168.1.2")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("X-Real-IP must be ignored; expected RemoteAddr 10.0.0.1, got %s", ip)
	}
}

func TestGetClientIP_XRealIP_NoPrecedenceOverXFF(t *testing.T) {
	// Critical security test: X-Real-IP (client-set) must NOT take precedence
	// over the platform-appended rightmost X-Forwarded-For entry.
	withCloudRun(t)
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Real-IP", "192.168.1.2")
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "5.6.7.8" {
		t.Errorf("Expected rightmost XFF 5.6.7.8 (X-Real-IP ignored), got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_SpoofingScenario(t *testing.T) {
	withCloudRun(t)
	// Test the actual attack scenario: attacker sends X-Forwarded-For with spoofed IP
	// Cloud Run appends real IP, resulting in "spoofed, real-ip"
	// We must use the rightmost (real) IP
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 192.168.1.1") // Attacker spoofed 1.2.3.4, Cloud Run added 192.168.1.1
	req.RemoteAddr = "10.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1 (rightmost IP, real IP added by Cloud Run), got %s", ip)
	}
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("POST", "/chat", nil)
	req.RemoteAddr = "192.168.1.3:12345"

	ip := getClientIP(req)
	if ip != "192.168.1.3" {
		t.Errorf("Expected 192.168.1.3, got %s", ip)
	}
}

func TestGetClientIP_RemoteAddr_IPv6(t *testing.T) {
	req := httptest.NewRequest("POST", "/chat", nil)
	req.RemoteAddr = "[2001:db8::2]:12345"

	ip := getClientIP(req)
	if ip != "2001:db8::2" {
		t.Errorf("Expected 2001:db8::2, got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_SpoofingPrevention(t *testing.T) {
	// Critical security test: Attacker tries to bypass rate limiting
	// by sending different X-Forwarded-For values
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// The attacker can rotate client-supplied XFF entries at will, but the
	// rightmost entry is the one the platform appends and is stable, so the
	// pre-auth bucket must be stable too: 16 requests total -> 429 on the 11th.
	withCloudRun(t)
	realClientIP := "198.51.100.7"
	spoofed := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4", "5.5.5.5", "999.999.999.999"}

	var rr *httptest.ResponseRecorder
	for i, spoof := range spoofed {
		req := httptest.NewRequest("POST", "/chat", nil)
		req.Header.Set("X-Forwarded-For", spoof+", "+realClientIP)
		req.RemoteAddr = "10.0.0.1:12345"
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if i < 10 && rr.Code != http.StatusOK {
			t.Fatalf("Request %d should succeed (stable platform IP key), got %d", i+1, rr.Code)
		}
	}

	// Exhaust: keep rotating the spoofed leftmost entries; the bucket must hold.
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/chat", nil)
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.9.9.%d, %s", i, realClientIP))
		req.RemoteAddr = "10.0.0.1:12345"
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429 once the single real-IP bucket is exhausted, got %d", rr.Code)
	}
}

func TestGetClientIP_ConcurrentAccess(t *testing.T) {
	// Test thread safety of getClientIP
	var wg sync.WaitGroup
	results := make(chan string, 100)

	// All should return the same IP: the rightmost parseable XFF entry
	// (the one the trusted platform appends on Cloud Run).
	t.Setenv("K_SERVICE", "clotilde") // enable the trusted-proxy path
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/chat", nil)
			req.Header.Set("X-Forwarded-For", "198.51.100.1, 192.168.1.1")
			req.RemoteAddr = "10.0.0.1:1234"
			ip := getClientIP(req)
			results <- ip
		}(i)
	}

	wg.Wait()
	close(results)

	for ip := range results {
		if ip != "192.168.1.1" {
			t.Errorf("Expected 192.168.1.1, got %s", ip)
		}
	}
}

func TestRateLimiter_IPFallback(t *testing.T) {
	// Test that rate limiting works with IP when no API key is provided
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// No API key - should use IP
	req := httptest.NewRequest("POST", "/chat", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rr := httptest.NewRecorder()

	// Make 11 requests (limit is 10 per minute)
	for i := 0; i < 11; i++ {
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429 after 11 requests from same IP, got %d", rr.Code)
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	// Test that different API keys have separate rate limits
	// Note: Rate limiter now requires validated API keys from context (set by auth middleware)
	// This test simulates the context that auth middleware would set
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make 10 requests with key1
	// Simulate validated API key in context (as auth middleware would do)
	key1 := "key-1"
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/chat", nil)
		req.Header.Set("X-API-Key", key1)
		// Add validated API key to context (simulating auth middleware behavior)
		ctx := auth.WithValidatedAPIKey(req.Context(), key1)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Request %d with key1 should succeed, got %d", i+1, rr.Code)
		}
	}

	// Make 10 requests with key2 (should still work - separate limit)
	key2 := "key-2"
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/chat", nil)
		req.Header.Set("X-API-Key", key2)
		// Add validated API key to context (simulating auth middleware behavior)
		ctx := auth.WithValidatedAPIKey(req.Context(), key2)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Request %d with key2 should succeed, got %d", i+1, rr.Code)
		}
	}
}

func TestGetClientIP_XForwardedFor_EmptyString(t *testing.T) {
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "")
	req.RemoteAddr = "192.168.1.1:12345"

	ip := getClientIP(req)
	// Should fallback to RemoteAddr
	if !strings.HasPrefix(ip, "192.168.1.1") {
		t.Errorf("Expected fallback to RemoteAddr, got %s", ip)
	}
}

func TestRateLimiter_EvictsOnNewKeysBeyondCap(t *testing.T) {
	rl := &rateLimiter{requests: make(map[string][]time.Time)}
	for i := 0; i < maxTrackedKeys+50; i++ {
		if !rl.isAllowed(fmt.Sprintf("ip-%d", i)) {
			t.Fatalf("first request for key %d unexpectedly denied", i)
		}
	}
	if len(rl.requests) > maxTrackedKeys {
		t.Fatalf("limiter map grew beyond cap: %d > %d", len(rl.requests), maxTrackedKeys)
	}
}

func TestPreAuthMiddleware_BlocksBruteForceNotAuthenticatedTraffic(t *testing.T) {
	authHandler := auth.Middleware("good-key")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler := PreAuthMiddleware()(authHandler)

	// Valid authenticated traffic from one IP can exceed 5/min (up to auth quota)
	for i := 0; i < 8; i++ {
		req := httptest.NewRequest(http.MethodPost, "/chat", nil)
		req.RemoteAddr = "198.51.100.50:12345"
		req.Header.Set("X-API-Key", "good-key")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("authenticated request %d got %d, want 200", i+1, rr.Code)
		}
	}

	// Failed auth attempts from another IP get blocked with 429 after 5 failures
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/chat", nil)
		req.RemoteAddr = "198.51.100.99:12345"
		req.Header.Set("X-API-Key", "wrong-key")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("failed auth %d got %d, want 401", i+1, rr.Code)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/chat", nil)
	req.RemoteAddr = "198.51.100.99:12345"
	req.Header.Set("X-API-Key", "wrong-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("6th attempt after brute force got %d, want 429", rr.Code)
	}
}
