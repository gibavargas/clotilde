package validator

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"unicode/utf8"
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
			// Only POST /chat has the {"message": "..."} contract. Other routes
			// and non-POST methods (OPTIONS, HEAD, GET) pass through to auth
			// and route handlers.
			if r.URL.Path != "/chat" || r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			// Limit request body size (read +1 byte so bodies of exactly maxRequestBodySize are allowed)
			limitedReader := io.LimitReader(r.Body, maxRequestBodySize+1)
			body, err := io.ReadAll(limitedReader)
			if err != nil {
				http.Error(w, `{"error":"Failed to read request body"}`, http.StatusBadRequest)
				return
			}
			r.Body.Close()

			// Check if body is too large
			if len(body) > maxRequestBodySize {
				http.Error(w, `{"error":"Request body too large"}`, http.StatusRequestEntityTooLarge)
				return
			}

			// Validate JSON structure
			var reqBody requestBody
			if err := json.Unmarshal(body, &reqBody); err != nil {
				http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
				return
			}

			// Validate message field (count characters/runes, not UTF-8 bytes)
			if utf8.RuneCountInString(reqBody.Message) > maxMessageLength {
				http.Error(w, `{"error":"Message too long"}`, http.StatusBadRequest)
				return
			}

			// Replace body for next handler
			r.Body = io.NopCloser(bytes.NewReader(body))

			next.ServeHTTP(w, r)
		})
	}
}
