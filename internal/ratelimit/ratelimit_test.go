package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

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
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1, got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_MultipleIPs(t *testing.T) {
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
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1:8080")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1 (port removed), got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_IPv6(t *testing.T) {
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "[2001:db8::1]:8080")
	req.RemoteAddr = "127.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "[2001:db8::1]" {
		t.Errorf("Expected [2001:db8::1] (IPv6 with brackets), got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_IPv6_Multiple(t *testing.T) {
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "[2001:db8::1], [2001:db8::2]")
	req.RemoteAddr = "127.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "[2001:db8::2]" {
		t.Errorf("Expected [2001:db8::2] (rightmost IPv6, added by Cloud Run), got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_Whitespace(t *testing.T) {
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Forwarded-For", "  192.168.1.1  ,  10.0.0.1  ")
	req.RemoteAddr = "127.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("Expected 10.0.0.1 (rightmost IP after whitespace trimmed), got %s", ip)
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Real-IP", "192.168.1.2")
	req.RemoteAddr = "10.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "192.168.1.2" {
		t.Errorf("Expected 192.168.1.2, got %s", ip)
	}
}

func TestGetClientIP_XRealIP_Precedence(t *testing.T) {
	// Critical security test: X-Real-IP should take precedence over X-Forwarded-For
	// X-Real-IP is set by Cloud Run and is more trustworthy
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-Real-IP", "192.168.1.2")
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8") // Spoofed IPs
	req.RemoteAddr = "10.0.0.1:12345"

	ip := getClientIP(req)
	if ip != "192.168.1.2" {
		t.Errorf("Expected 192.168.1.2 (X-Real-IP takes precedence), got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_SpoofingScenario(t *testing.T) {
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
	if ip != "[2001:db8::2]" {
		t.Errorf("Expected [2001:db8::2], got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor_SpoofingPrevention(t *testing.T) {
	// Critical security test: Attacker tries to bypass rate limiting
	// by sending different X-Forwarded-For values
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Simulate attacker sending unique IPs to bypass rate limit
	uniqueIPs := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4", "5.5.5.5"}
	
	// All requests should be rate limited by the same key (API key)
	// because we use API key, not IP, for rate limiting
	apiKey := "test-key-spoofing"
	
	for i, spoofedIP := range uniqueIPs {
		req := httptest.NewRequest("POST", "/chat", nil)
		req.Header.Set("X-API-Key", apiKey)
		req.Header.Set("X-Forwarded-For", spoofedIP)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// First request should succeed
		if i == 0 {
			if rr.Code != http.StatusOK {
				t.Errorf("First request should succeed, got %d", rr.Code)
			}
		}
	}

	// Make 10 more requests with the same API key (should hit rate limit)
	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Forwarded-For", "999.999.999.999") // Different IP, same key
	rr := httptest.NewRecorder()

	// Make requests to exhaust rate limit
	for i := 0; i < 10; i++ {
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	// Next request should be rate limited (11th request)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limit after 11 requests with same API key, got %d", rr.Code)
	}
}

func TestGetClientIP_ConcurrentAccess(t *testing.T) {
	// Test thread safety of getClientIP
	var wg sync.WaitGroup
	results := make(chan string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/chat", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.1")
			ip := getClientIP(req)
			results <- ip
		}(i)
	}

	wg.Wait()
	close(results)

	// All should return the same IP
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
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make 10 requests with key1
	key1 := "key-1"
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/chat", nil)
		req.Header.Set("X-API-Key", key1)
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

