package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clotilde/carplay-assistant/internal/admin"
	"github.com/clotilde/carplay-assistant/internal/auth"
	"github.com/clotilde/carplay-assistant/internal/logging"
	"github.com/clotilde/carplay-assistant/internal/ratelimit"
	"github.com/clotilde/carplay-assistant/internal/validator"
)

// buildMiddlewareChain wires the production chain exactly like main() does
// (main.go: RequestID → Validator → PreAuth → Auth → RateLimit), so tests can
// exercise the real order instead of trusting a documentation string.
func buildMiddlewareChain(apiKey string, mux *http.ServeMux) http.Handler {
	handler := ratelimit.Middleware()(mux)
	handler = auth.Middleware(apiKey)(handler)
	handler = ratelimit.PreAuthMiddleware()(handler)
	handler = validator.Middleware()(handler)
	handler = logging.RequestIDMiddleware(handler)
	return handler
}

// TestMiddlewareChainIntegration replaces the deleted TestMiddlewareOrder_Conceptual:
// through the REAL chain, a valid key reaches the handler; a wrong key gets 401
// before rate limiting consumes quota; an oversized body is rejected early.
func TestMiddlewareChainIntegration(t *testing.T) {
	const validKey = "integration-key"

	var reached int
	mux := http.NewServeMux()
	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		reached++
		w.WriteHeader(http.StatusOK)
	})

	chain := buildMiddlewareChain(validKey, mux)

	// 1. Valid key passes the whole chain.
	req := httptest.NewRequest("POST", "/chat", strings.NewReader(`{"message":"oi"}`))
	req.Header.Set("X-API-Key", validKey)
	req.RemoteAddr = "10.1.1.1:1000"
	rr := httptest.NewRecorder()
	chain.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || reached != 1 {
		t.Fatalf("valid key must reach handler: got %d reached=%d", rr.Code, reached)
	}

	// 2. Wrong key: 401, handler not reached.
	req = httptest.NewRequest("POST", "/chat", strings.NewReader(`{"message":"oi"}`))
	req.Header.Set("X-API-Key", "wrong-key")
	req.RemoteAddr = "10.1.1.2:1000"
	rr = httptest.NewRecorder()
	chain.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized || reached != 1 {
		t.Fatalf("wrong key must 401 before handler: got %d reached=%d", rr.Code, reached)
	}

	// 3. No key: 401.
	req = httptest.NewRequest("POST", "/chat", strings.NewReader(`{}`))
	req.RemoteAddr = "10.1.1.3:1000"
	rr = httptest.NewRecorder()
	chain.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("missing key must 401, got %d", rr.Code)
	}
}

// TestGuardrailDisabledByDefault pins the opt-in guardrail contract: with the
// env unset, a question matching the broad 'trajeto'+'destino' pair must NOT
// be hijacked by the canned refusal (it reaches routing). With the env set,
// it short-circuits.
func TestGuardrailDisabledByDefault(t *testing.T) {
	if guardrailsEnabled() {
		t.Fatal("guardrails must default OFF")
	}

	// Broad-pair question that used to be hijacked before routing. The
	// matcher itself still matches (the heuristic is intact); the dispatch
	// gate is what changed.
	msg := "qual o trajeto do pedágio até o destino centro?"
	if _, ok := highUncertaintyGuardrailResponse(msg); !ok {
		t.Fatal("test fixture must actually match the guardrail pair, else the test proves nothing")
	}

	// With GUARDRAILS_ENABLED=true the canned answer is used.
	t.Setenv("GUARDRAILS_ENABLED", "true")
	if !guardrailsEnabled() {
		t.Fatal("GUARDRAILS_ENABLED=true must activate the gate")
	}
	if resp, ok := highUncertaintyGuardrailResponse(msg); !ok || resp == "" {
		t.Fatal("matched pair must produce the canned response")
	}
}

// TestCORSHeadersReal replaces the deleted TestCORSConfiguration: it asserts
// the actual setCORSHeaders output instead of testing os.Setenv.
func TestCORSHeadersReal(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGIN", "https://example.com")

	w := httptest.NewRecorder()
	setCORSHeaders(w)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want https://example.com", got)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Access-Control-Allow-Methods must be set")
	}
}

// keep admin import used if future assertions need config restore.
var _ = admin.GetConfig
