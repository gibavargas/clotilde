package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

// Context key for request ID
type contextKey string

const requestIDKey contextKey = "requestID"

// GenerateRequestID creates a short unique request ID
func GenerateRequestID() string {
	bytes := make([]byte, 8) // 16 hex characters
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to less secure but functional ID
		return "fallback-id"
	}
	return hex.EncodeToString(bytes)
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request already has an ID (from upstream proxy)
		requestID := r.Header.Get("X-Request-ID")
		if requestID != "" {
			requestID = sanitizeRequestID(requestID)
		}
		if requestID == "" {
			requestID = GenerateRequestID()
		}

		// Add to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Add to request context
		ctx := WithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

const maxRequestIDLength = 64

func sanitizeRequestID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}

	if len(id) > maxRequestIDLength {
		id = id[:maxRequestIDLength]
	}

	for _, r := range id {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' {
			continue
		}
		// Invalid character found; reject entire header value
		return ""
	}
	return id
}

