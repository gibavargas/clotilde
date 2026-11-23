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
		if len(rl.requests) >= maxTrackedKeys {
			evictOldestKey(rl.requests)
		}
		rl.requests[key] = []time.Time{now}
		return true
	}

	// Prune entries older than one hour and count requests in last minute/hour.
	validTimes := times[:0]
	minuteCount := 0
	hourCount := 0
	for _, t := range times {
		if t.After(oneHourAgo) {
			validTimes = append(validTimes, t)
			hourCount++
			if t.After(oneMinuteAgo) {
				minuteCount++
			}
		}
	}

	// Check limits
	if minuteCount >= perMinute || hourCount >= perHour {
		rl.requests[key] = validTimes
		return false
	}

	// Add current request
	rl.requests[key] = append(validTimes, now)
	return true
}

func (rl *rateLimiter) isExhausted(key string, perMinute, perHour int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	times, exists := rl.requests[key]
	if !exists {
		return false
	}

	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)
	oneHourAgo := now.Add(-1 * time.Hour)

	minuteCount := 0
	hourCount := 0
	for _, t := range times {
		if t.After(oneHourAgo) {
			hourCount++
			if t.After(oneMinuteAgo) {
				minuteCount++
			}
		}
	}
	return minuteCount >= perMinute || hourCount >= perHour
}

func (rl *rateLimiter) record(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	times, exists := rl.requests[key]
	if !exists {
		if len(rl.requests) >= maxTrackedKeys {
			evictOldestKey(rl.requests)
		}
		rl.requests[key] = []time.Time{now}
		return
	}

	validTimes := times[:0]
	for _, t := range times {
		if t.After(oneHourAgo) {
			validTimes = append(validTimes, t)
		}
	}
	rl.requests[key] = append(validTimes, now)
}

func evictOldestKey(m map[string][]time.Time) {
	evictOldest(m, func(times []time.Time) time.Time {
		if len(times) == 0 {
			return time.Unix(0, 0)
		}
		return times[len(times)-1]
	})
}

// maxTrackedKeys bounds the number of distinct limiter keys (blunts a
// memory-exhaustion attempt that rotates source IPs).
const maxTrackedKeys = 10000

// evictOldest deletes the entry with the oldest stamp, keeping the map at a
// bounded size. O(n) per call, amortized negligible: it only runs on the
// rare insert that crosses the bound.
func evictOldest[K comparable, V any](m map[K]V, stamp func(V) time.Time) {
	var oldestKey K
	var oldest time.Time
	hasOldest := false
	for k, v := range m {
		t := stamp(v)
		if !hasOldest || t.Before(oldest) {
			oldestKey, oldest, hasOldest = k, t, true
		}
	}
	if hasOldest {
		delete(m, oldestKey)
	}
}

// Middleware implements rate limiting per validated API key or IP address
// IMPORTANT: This middleware must run AFTER auth.Middleware to ensure API keys are validated
// Only validated API keys (from context) are used for rate limiting to prevent bypass attacks
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for CORS preflight, health check, and admin routes
			if r.Method == http.MethodOptions || r.URL.Path == "/health" || r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/") {
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

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Unwrap() http.ResponseWriter {
	return sr.ResponseWriter
}

// PreAuthMiddleware implements IP-based brute-force protection BEFORE authentication.
// It blocks IPs that have exhausted their failed-auth budget before auth runs,
// while only counting authentication failures against the stricter pre-auth
// limits so legitimate authenticated traffic can use the full per-key quota.
func PreAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip pre-auth rate limiting for CORS preflight, health check, and admin routes
			if r.Method == http.MethodOptions || r.URL.Path == "/health" || r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/") {
				next.ServeHTTP(w, r)
				return
			}

			// Use IP address as key (no API key validation yet)
			ip := clientip.FromRequest(r)

			// Reject IPs that already hit the pre-auth brute-force cap before auth runs
			if preAuthLimiter.isExhausted(ip, preAuthRequestsPerMinute, preAuthRequestsPerHour) {
				http.Error(w, `{"error":"Too many requests"}`, http.StatusTooManyRequests)
				return
			}

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Record failed auth attempts (401 on protected endpoints, or 404 on
			// /metrics when bearer token is missing/invalid)
			if rec.status == http.StatusUnauthorized || (r.URL.Path == "/metrics" && rec.status == http.StatusNotFound) {
				preAuthLimiter.record(ip)
			}
		})
	}
}
