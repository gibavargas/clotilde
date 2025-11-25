package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

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

			next.ServeHTTP(w, r)
		})
	}
}

