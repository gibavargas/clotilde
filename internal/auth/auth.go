package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
)

// Context key for validated API key
type contextKey string

const validatedAPIKeyKey contextKey = "validatedAPIKey"

// WithValidatedAPIKey adds the validated API key to the context
func WithValidatedAPIKey(ctx context.Context, apiKey string) context.Context {
	return context.WithValue(ctx, validatedAPIKeyKey, apiKey)
}

// GetValidatedAPIKey retrieves the validated API key from context
func GetValidatedAPIKey(ctx context.Context) string {
	if key, ok := ctx.Value(validatedAPIKeyKey).(string); ok {
		return key
	}
	return ""
}

// Middleware validates the X-API-Key header against the expected API key
func Middleware(expectedAPIKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for health check and admin routes (admin has its own Basic Auth)
			if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/admin") {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				http.Error(w, `{"error":"API key required"}`, http.StatusUnauthorized)
				return
			}

			// Use constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(expectedAPIKey)) != 1 {
				http.Error(w, `{"error":"Invalid API key"}`, http.StatusUnauthorized)
				return
			}

			// API key is validated - add to context for rate limiter
			ctx := WithValidatedAPIKey(r.Context(), apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

