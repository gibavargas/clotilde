package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddleware_ValidAPIKey(t *testing.T) {
	expectedKey := "test-api-key-123"
	handler := Middleware(expectedKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-API-Key", expectedKey)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware_MissingAPIKey(t *testing.T) {
	expectedKey := "test-api-key-123"
	handler := Middleware(expectedKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/chat", nil)
	// No X-API-Key header
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}

	expectedBody := `{"error":"API key required"}`
	actualBody := strings.TrimSpace(rr.Body.String())
	if actualBody != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, actualBody)
	}
}

func TestMiddleware_InvalidAPIKey(t *testing.T) {
	expectedKey := "test-api-key-123"
	handler := Middleware(expectedKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/chat", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}

	expectedBody := `{"error":"Invalid API key"}`
	actualBody := strings.TrimSpace(rr.Body.String())
	if actualBody != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, actualBody)
	}
}

func TestMiddleware_HealthCheckBypass(t *testing.T) {
	expectedKey := "test-api-key-123"
	handler := Middleware(expectedKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	// No X-API-Key header - should still work
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware_AdminRouteBypass(t *testing.T) {
	expectedKey := "test-api-key-123"
	handler := Middleware(expectedKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/admin/dashboard", nil)
	// No X-API-Key header - should still work (admin has its own auth)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware_ConstantTimeComparison(t *testing.T) {
	// This test verifies that constant-time comparison is used
	// We can't directly test timing, but we can verify it works correctly
	expectedKey := "test-api-key-123"
	handler := Middleware(expectedKey)

	// Test with keys of different lengths (should still use constant-time)
	testCases := []struct {
		name     string
		apiKey   string
		expected int
	}{
		{"correct key", expectedKey, http.StatusOK},
		{"wrong key same length", "test-api-key-456", http.StatusUnauthorized},
		{"wrong key shorter", "short", http.StatusUnauthorized},
		{"wrong key longer", "test-api-key-123-extra", http.StatusUnauthorized},
		{"empty key", "", http.StatusUnauthorized},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("POST", "/chat", nil)
			if tc.apiKey != "" {
				req.Header.Set("X-API-Key", tc.apiKey)
			}
			rr := httptest.NewRecorder()

			wrapped.ServeHTTP(rr, req)

			if rr.Code != tc.expected {
				t.Errorf("Expected status %d, got %d", tc.expected, rr.Code)
			}
		})
	}
}

