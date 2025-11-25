package ratelimit

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
}

var (
	globalLimiter = &rateLimiter{
		requests: make(map[string][]time.Time),
	}
	
	// Rate limits
	requestsPerMinute = 10
	requestsPerHour   = 100
	cleanupInterval   = 5 * time.Minute
)

func init() {
	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			globalLimiter.cleanup()
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
	if minuteCount >= requestsPerMinute {
		return false
	}
	if hourCount >= requestsPerHour {
		return false
	}

	// Add current request
	rl.requests[key] = append(times, now)
	return true
}

// Middleware implements rate limiting per IP address
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for health check and admin routes
			if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/admin") {
				next.ServeHTTP(w, r)
				return
			}

			// Use IP address as key
			ip := getClientIP(r)
			apiKey := r.Header.Get("X-API-Key")
			
			// Use API key if available, otherwise use IP
			key := apiKey
			if key == "" {
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
	// Check X-Forwarded-For header (Cloud Run sets this)
	// Extract first IP only to prevent header spoofing bypass
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take only the first IP (original client)
		// X-Forwarded-For can contain: "client, proxy1, proxy2"
		if idx := strings.Index(forwarded, ","); idx != -1 {
			forwarded = strings.TrimSpace(forwarded[:idx])
		} else {
			forwarded = strings.TrimSpace(forwarded)
		}
		// Remove port if present (e.g., "192.168.1.1:12345" -> "192.168.1.1")
		// Handle IPv6 addresses with brackets (e.g., "[::1]:12345" -> "[::1]")
		if strings.HasPrefix(forwarded, "[") {
			// IPv6 with brackets
			if idx := strings.Index(forwarded, "]:"); idx != -1 {
				forwarded = forwarded[:idx+1]
			}
		} else {
			// IPv4 or IPv6 without brackets
			if idx := strings.Index(forwarded, ":"); idx != -1 {
				forwarded = forwarded[:idx]
			}
		}
		return forwarded
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		// Remove port if present
		if idx := strings.Index(realIP, ":"); idx != -1 {
			realIP = realIP[:idx]
		}
		return strings.TrimSpace(realIP)
	}

	// Fallback to RemoteAddr (format: "IP:port" or "[IPv6]:port")
	remoteAddr := r.RemoteAddr
	if strings.HasPrefix(remoteAddr, "[") {
		// IPv6 with brackets
		if idx := strings.Index(remoteAddr, "]:"); idx != -1 {
			remoteAddr = remoteAddr[:idx+1]
		}
	} else {
		// IPv4 or IPv6 without brackets
		if idx := strings.Index(remoteAddr, ":"); idx != -1 {
			remoteAddr = remoteAddr[:idx]
		}
	}
	return remoteAddr
}

