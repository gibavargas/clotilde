package validator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddleware_ValidRequest(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Message string `json:"message"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Message != "Hello" {
			t.Errorf("Expected message 'Hello', got %q", body.Message)
		}
		w.WriteHeader(http.StatusOK)
	}))

	reqBody := map[string]string{"message": "Hello"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware_InvalidJSON(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/chat", bytes.NewReader([]byte("not json")))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	expectedBody := `{"error":"Invalid JSON"}`
	actualBody := strings.TrimSpace(rr.Body.String())
	if actualBody != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, actualBody)
	}
}

func TestMiddleware_MissingMessageField(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handler should be called - missing message field is allowed (empty string)
		w.WriteHeader(http.StatusOK)
	}))

	reqBody := map[string]string{"wrong": "field"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Missing message field is still valid JSON, but message will be empty
	// The validator doesn't require message to be non-empty, so this should pass
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 (empty message is allowed), got %d", rr.Code)
	}
}

func TestMiddleware_MessageTooLong(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	// Create message longer than maxMessageLength (1000 chars)
	longMessage := strings.Repeat("a", 1001)
	reqBody := map[string]string{"message": longMessage}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	expectedBody := `{"error":"Message too long"}`
	actualBody := strings.TrimSpace(rr.Body.String())
	if actualBody != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, actualBody)
	}
}

func TestMiddleware_MessageAtLimit(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create message exactly at maxMessageLength (1000 chars)
	message := strings.Repeat("a", 1000)
	reqBody := map[string]string{"message": message}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 (message at limit should pass), got %d", rr.Code)
	}
}

func TestMiddleware_BodyTooLarge(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	// Create body larger than maxRequestBodySize (5KB)
	largeBody := bytes.Repeat([]byte("a"), 5*1024+1)
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(largeBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413, got %d", rr.Code)
	}

	expectedBody := `{"error":"Request body too large"}`
	actualBody := strings.TrimSpace(rr.Body.String())
	if actualBody != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, actualBody)
	}
}

func TestMiddleware_BodyAtLimit(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create body well under maxRequestBodySize (5KB = 5120 bytes)
	// But message must be <= 1000 chars (maxMessageLength)
	// Use message at the limit to test body size validation
	messageSize := 1000 // At maxMessageLength limit
	message := strings.Repeat("a", messageSize)
	reqBody := map[string]string{"message": message}
	bodyBytes, _ := json.Marshal(reqBody)

	// Verify we're under the limit (5KB = 5120 bytes)
	if len(bodyBytes) >= maxRequestBodySize {
		t.Fatalf("Test setup error: body size %d >= limit %d", len(bodyBytes), maxRequestBodySize)
	}

	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should pass if under limit
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 (body under limit should pass), got %d (body size: %d)", rr.Code, len(bodyBytes))
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

func TestMiddleware_OPTIONSBypass(t *testing.T) {
	// Critical: OPTIONS requests must bypass validation (CORS preflight)
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/chat", nil)
	// No body - should still work
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS request, got %d", rr.Code)
	}
}

func TestMiddleware_EmptyBody(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/chat", bytes.NewReader([]byte("")))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty body, got %d", rr.Code)
	}
}

func TestMiddleware_ValidJSON_InvalidStructure(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	}))

	// Valid JSON but not an object
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader([]byte(`["array", "not", "object"]`)))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid structure, got %d", rr.Code)
	}
}

func TestMiddleware_BodyPreserved(t *testing.T) {
	// Test that body is preserved for next handler
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode body: %v", err)
		}
		if body.Message != "Test message" {
			t.Errorf("Expected 'Test message', got %q", body.Message)
		}
		w.WriteHeader(http.StatusOK)
	}))

	reqBody := map[string]string{"message": "Test message"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestMiddleware_UnicodeMessage(t *testing.T) {
	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Message string `json:"message"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Message != "Olá, mundo! 🌍" {
			t.Errorf("Expected Unicode message, got %q", body.Message)
		}
		w.WriteHeader(http.StatusOK)
	}))

	reqBody := map[string]string{"message": "Olá, mundo! 🌍"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(bodyBytes))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

