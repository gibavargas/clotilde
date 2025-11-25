package validator

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const (
	maxRequestBodySize = 5 * 1024 // 5KB
	maxMessageLength   = 1000     // characters
)

type requestBody struct {
	Message string `json:"message"`
}

// Middleware validates request size and content
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for health check, admin routes, and OPTIONS preflight
			if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/admin") || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Limit request body size
			limitedReader := io.LimitReader(r.Body, maxRequestBodySize)
			body, err := io.ReadAll(limitedReader)
			if err != nil {
				http.Error(w, `{"error":"Failed to read request body"}`, http.StatusBadRequest)
				return
			}
			r.Body.Close()

			// Check if body is too large
			if len(body) >= maxRequestBodySize {
				http.Error(w, `{"error":"Request body too large"}`, http.StatusRequestEntityTooLarge)
				return
			}

			// Validate JSON structure
			var reqBody requestBody
			if err := json.Unmarshal(body, &reqBody); err != nil {
				http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
				return
			}

			// Validate message field
			if len(reqBody.Message) > maxMessageLength {
				http.Error(w, `{"error":"Message too long"}`, http.StatusBadRequest)
				return
			}

			// Replace body for next handler
			r.Body = io.NopCloser(bytes.NewReader(body))

			next.ServeHTTP(w, r)
		})
	}
}

