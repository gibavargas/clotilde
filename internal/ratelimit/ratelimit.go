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
	// Check X-Real-IP header first (most trusted in Google Cloud Run)
	// Cloud Run sets X-Real-IP reliably and strips user input, making it safe to trust
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		realIP = strings.TrimSpace(realIP)
		// Remove port if present
		if strings.HasPrefix(realIP, "[") {
			// IPv6 with brackets
			if idx := strings.Index(realIP, "]:"); idx != -1 {
				realIP = realIP[:idx+1]
			}
		} else {
			// IPv4 or IPv6 without brackets
			if idx := strings.Index(realIP, ":"); idx != -1 {
				realIP = realIP[:idx]
			}
		}
		return realIP
	}

	// Fallback to X-Forwarded-For header (Cloud Run appends real IP to existing header)
	// In Cloud Run: if attacker sends "X-Forwarded-For: 1.2.3.4", Cloud Run appends real IP
	// Result: "1.2.3.4, <Real-IP>". We must take the RIGHTmost IP (after trusted proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Extract rightmost IP (most recent, added by Cloud Run)
		// X-Forwarded-For format: "client, proxy1, proxy2" or "spoofed, real-ip"
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			// Take the last (rightmost) IP
			forwarded = strings.TrimSpace(ips[len(ips)-1])
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

	// Final fallback to RemoteAddr (format: "IP:port" or "[IPv6]:port")
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

