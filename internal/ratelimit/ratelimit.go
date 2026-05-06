package ratelimit

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/clotilde/carplay-assistant/internal/auth"
	"github.com/clotilde/carplay-assistant/internal/clientip"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
}

var (
	globalLimiter = &rateLimiter{
		requests: make(map[string][]time.Time),
	}

	// Pre-auth IP-based rate limiter for brute force protection
	preAuthLimiter = &rateLimiter{
		requests: make(map[string][]time.Time),
	}

	// Rate limits
	requestsPerMinute = 10
	requestsPerHour   = 100
	cleanupInterval   = 5 * time.Minute

	// Pre-auth rate limits (stricter to prevent brute force)
	preAuthRequestsPerMinute = 5  // Lower limit before authentication
	preAuthRequestsPerHour   = 20 // Lower hourly limit
)

func init() {
	// Start cleanup goroutines
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			globalLimiter.cleanup()
			preAuthLimiter.cleanup()
		}
	}()
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	for key, times := range rl.requests {
		// Remove old entries
		validTimes := []time.Time{}
		for _, t := range times {
			if t.After(oneHourAgo) {
				validTimes = append(validTimes, t)
			}
		}

		if len(validTimes) == 0 {
			delete(rl.requests, key)
		} else {
			rl.requests[key] = validTimes
		}
	}
}

func (rl *rateLimiter) isAllowed(key string) bool {
	return rl.isAllowedWithLimits(key, requestsPerMinute, requestsPerHour)
}

func (rl *rateLimiter) isAllowedWithLimits(key string, perMinute, perHour int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)
	oneHourAgo := now.Add(-1 * time.Hour)

	times, exists := rl.requests[key]
	if !exists {
		rl.requests[key] = []time.Time{now}
		return true
	}

	// Count requests in last minute
	minuteCount := 0
	hourCount := 0
	for _, t := range times {
		if t.After(oneMinuteAgo) {
			minuteCount++
		}
		if t.After(oneHourAgo) {
			hourCount++
		}
	}

	// Check limits
	if minuteCount >= perMinute {
		return false
	}
	if hourCount >= perHour {
		return false
	}

	// Add current request
	rl.requests[key] = append(times, now)
	return true
}

// Middleware implements rate limiting per validated API key or IP address
// IMPORTANT: This middleware must run AFTER auth.Middleware to ensure API keys are validated
// Only validated API keys (from context) are used for rate limiting to prevent bypass attacks
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for health check and admin routes
			if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/admin") {
				next.ServeHTTP(w, r)
				return
			}

			// Get validated API key from context (set by auth.Middleware)
			// Only use API key if it has been validated to prevent rate limit bypass
			validatedAPIKey := auth.GetValidatedAPIKey(r.Context())

			var key string
			if validatedAPIKey != "" {
				// Use validated API key for rate limiting
				key = validatedAPIKey
			} else {
				// Fallback to IP address if no validated API key
				// This should only happen if auth middleware was skipped
				ip := clientip.FromRequest(r)
				key = ip
			}

			if !globalLimiter.isAllowed(key) {
				http.Error(w, `{"error":"Rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	return clientip.FromRequest(r)
}

// PreAuthMiddleware implements IP-based rate limiting BEFORE authentication
// This protects against brute force attacks on the authentication endpoint
// It runs before auth.Middleware and uses stricter limits
func PreAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip pre-auth rate limiting for health check and admin routes
			if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/admin") {
				next.ServeHTTP(w, r)
				return
			}

			// Use IP address as key (no API key validation yet)
			ip := clientip.FromRequest(r)

			// Use stricter limits for pre-auth requests
			if !preAuthLimiter.isAllowedWithLimits(ip, preAuthRequestsPerMinute, preAuthRequestsPerHour) {
				http.Error(w, `{"error":"Too many requests"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
