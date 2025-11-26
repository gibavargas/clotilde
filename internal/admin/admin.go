package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/clotilde/carplay-assistant/internal/logging"
)

// Config holds admin configuration
type Config struct {
	Username string
	Password string
}

// csrfTokenInfo holds CSRF token metadata for reuse tracking
type csrfTokenInfo struct {
	Token       string
	ExpiresAt   time.Time
	CreatedAt   time.Time
	IP          string
	UserAgent   string
	LastUsed    time.Time
}

// adminRateLimiter tracks rate limits for admin routes
type adminRateLimiter struct {
	authAttempts map[string][]time.Time // IP -> failed auth timestamps
	requests     map[string][]time.Time // IP -> request timestamps
	csrfTokens   map[string]int         // IP -> token count
	lockedIPs   map[string]time.Time   // IP -> lockout expiration
	mu           sync.RWMutex
}

const (
	// CSRF token limits
	maxCSRFTokensGlobal = 1000
	maxCSRFTokensPerIP   = 10
	csrfTokenReuseWindow = 5 * time.Minute
	csrfTokenLifetime    = 24 * time.Hour
	
	// Rate limiting
	maxAuthAttemptsPerMinute = 5
	maxRequestsPerMinute     = 30
	bruteForceLockoutDuration = 15 * time.Minute
	
	// Request size limits
	maxSystemPromptSize = 10 * 1024  // 10KB
	maxConfigBodySize   = 50 * 1024  // 50KB
)

// Handler handles admin routes
type Handler struct {
	config       Config
	logger       *logging.Logger
	csrfTokens   map[string]*csrfTokenInfo // token -> info
	csrfTokensByIP map[string]map[string]bool // IP -> set of tokens
	csrfMutex    sync.RWMutex
	rateLimiter  *adminRateLimiter
}

// NewHandler creates a new admin handler
func NewHandler(logger *logging.Logger) *Handler {
	h := &Handler{
		config: Config{
			Username: os.Getenv("ADMIN_USER"),
			Password: os.Getenv("ADMIN_PASSWORD"),
		},
		logger:         logger,
		csrfTokens:      make(map[string]*csrfTokenInfo),
		csrfTokensByIP:  make(map[string]map[string]bool),
		rateLimiter: &adminRateLimiter{
			authAttempts: make(map[string][]time.Time),
			requests:     make(map[string][]time.Time),
			csrfTokens:   make(map[string]int),
			lockedIPs:   make(map[string]time.Time),
		},
	}
	// Start cleanup goroutines
	go h.cleanupExpiredTokens()
	go h.rateLimiter.cleanup()
	return h
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (Cloud Run sets this)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take only the first IP (original client) to prevent header spoofing
		if idx := strings.Index(forwarded, ","); idx != -1 {
			return strings.TrimSpace(forwarded[:idx])
		}
		return strings.TrimSpace(forwarded)
	}
	
	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return strings.TrimSpace(realIP)
	}
	
	// Fallback to RemoteAddr (remove port if present)
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return addr
}

// hashIPUserAgent creates a hash of IP + User-Agent for session tracking
func hashIPUserAgent(ip, userAgent string) string {
	data := ip + "|" + userAgent
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes
}

// generateCSRFToken creates a new CSRF token with memory limits and IP tracking
func (h *Handler) generateCSRFToken(r *http.Request) string {
	ip := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")
	
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("Error generating CSRF token: %v", err)
		return ""
	}
	token := base64.URLEncoding.EncodeToString(bytes)
	
	now := time.Now()
	
	h.csrfMutex.Lock()
	defer h.csrfMutex.Unlock()
	
	// Check per-IP limit
	ipTokens := h.csrfTokensByIP[ip]
	if ipTokens == nil {
		ipTokens = make(map[string]bool)
		h.csrfTokensByIP[ip] = ipTokens
	}
	
	// Enforce per-IP limit (LRU eviction)
	if len(ipTokens) >= maxCSRFTokensPerIP {
		// Find oldest token for this IP
		var oldestToken string
		var oldestTime time.Time
		for t := range ipTokens {
			if info, exists := h.csrfTokens[t]; exists {
				if oldestToken == "" || info.CreatedAt.Before(oldestTime) {
					oldestToken = t
					oldestTime = info.CreatedAt
				}
			}
		}
		if oldestToken != "" {
			delete(h.csrfTokens, oldestToken)
			delete(ipTokens, oldestToken)
		}
	}
	
	// Enforce global limit (LRU eviction)
	if len(h.csrfTokens) >= maxCSRFTokensGlobal {
		// Find oldest token globally
		var oldestToken string
		var oldestTime time.Time
		for t, info := range h.csrfTokens {
			if oldestToken == "" || info.CreatedAt.Before(oldestTime) {
				oldestToken = t
				oldestTime = info.CreatedAt
			}
		}
		if oldestToken != "" {
			// Remove from IP tracking
			if info := h.csrfTokens[oldestToken]; info != nil {
				if ipSet := h.csrfTokensByIP[info.IP]; ipSet != nil {
					delete(ipSet, oldestToken)
					if len(ipSet) == 0 {
						delete(h.csrfTokensByIP, info.IP)
					}
				}
			}
			delete(h.csrfTokens, oldestToken)
		}
	}
	
	// Create new token
	h.csrfTokens[token] = &csrfTokenInfo{
		Token:     token,
		ExpiresAt: now.Add(csrfTokenLifetime),
		CreatedAt: now,
		IP:        ip,
		UserAgent: userAgent,
		LastUsed:  now,
	}
	ipTokens[token] = true
	
	return token
}

// validateCSRFToken checks if a CSRF token is valid and allows reuse within window
func (h *Handler) validateCSRFToken(token string, r *http.Request) bool {
	if token == "" {
		return false
	}
	
	ip := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")
	now := time.Now()
	
	h.csrfMutex.Lock()
	defer h.csrfMutex.Unlock()
	
	info, exists := h.csrfTokens[token]
	if !exists {
		return false
	}
	
	// Check if token is expired
	if now.After(info.ExpiresAt) {
		// Token expired, remove it
		h.removeToken(token)
		return false
	}
	
	// Verify IP and User-Agent match (session validation)
	if info.IP != ip || info.UserAgent != userAgent {
		// IP or User-Agent changed - invalidate token
		h.removeToken(token)
		return false
	}
	
	// Check if token can be reused (within 5-minute window from last use)
	if now.Sub(info.LastUsed) > csrfTokenReuseWindow {
		// Reuse window expired - token must be regenerated
		h.removeToken(token)
		return false
	}
	
	// Token is valid - update last used time (allow reuse)
	info.LastUsed = now
	return true
}

// removeToken removes a token from all tracking structures
func (h *Handler) removeToken(token string) {
	info := h.csrfTokens[token]
	if info != nil {
		// Remove from IP tracking
		if ipSet := h.csrfTokensByIP[info.IP]; ipSet != nil {
			delete(ipSet, token)
			if len(ipSet) == 0 {
				delete(h.csrfTokensByIP, info.IP)
			}
		}
	}
	delete(h.csrfTokens, token)
}

// cleanupExpiredTokens periodically removes expired tokens
func (h *Handler) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		h.csrfMutex.Lock()
		now := time.Now()
		var expiredTokens []string
		for token, info := range h.csrfTokens {
			if now.After(info.ExpiresAt) {
				expiredTokens = append(expiredTokens, token)
			}
		}
		for _, token := range expiredTokens {
			h.removeToken(token)
		}
		h.csrfMutex.Unlock()
	}
}

// cleanup removes old rate limit entries
func (rl *adminRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		oneHourAgo := now.Add(-1 * time.Hour)
		
		// Clean up old auth attempts
		for ip, attempts := range rl.authAttempts {
			validAttempts := []time.Time{}
			for _, t := range attempts {
				if t.After(oneHourAgo) {
					validAttempts = append(validAttempts, t)
				}
			}
			if len(validAttempts) == 0 {
				delete(rl.authAttempts, ip)
			} else {
				rl.authAttempts[ip] = validAttempts
			}
		}
		
		// Clean up old requests
		for ip, requests := range rl.requests {
			validRequests := []time.Time{}
			for _, t := range requests {
				if t.After(oneHourAgo) {
					validRequests = append(validRequests, t)
				}
			}
			if len(validRequests) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = validRequests
			}
		}
		
		// Clean up expired lockouts
		for ip, lockoutUntil := range rl.lockedIPs {
			if now.After(lockoutUntil) {
				delete(rl.lockedIPs, ip)
			}
		}
		
		rl.mu.Unlock()
	}
}

// isIPLocked checks if an IP is currently locked out
func (rl *adminRateLimiter) isIPLocked(ip string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	lockoutUntil, locked := rl.lockedIPs[ip]
	return locked && time.Now().Before(lockoutUntil)
}

// recordFailedAuth records a failed authentication attempt
func (rl *adminRateLimiter) recordFailedAuth(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)
	
	// Get recent attempts
	attempts := rl.authAttempts[ip]
	recentAttempts := []time.Time{}
	for _, t := range attempts {
		if t.After(oneMinuteAgo) {
			recentAttempts = append(recentAttempts, t)
		}
	}
	
	// Add current attempt
	recentAttempts = append(recentAttempts, now)
	rl.authAttempts[ip] = recentAttempts
	
	// Check if limit exceeded
	if len(recentAttempts) >= maxAuthAttemptsPerMinute {
		// Lock out IP
		rl.lockedIPs[ip] = now.Add(bruteForceLockoutDuration)
		log.Printf("Admin: IP %s locked out after %d failed auth attempts", ip, len(recentAttempts))
		return false
	}
	
	return true
}

// checkRequestRate checks if request rate limit is exceeded
func (rl *adminRateLimiter) checkRequestRate(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)
	
	// Get recent requests
	requests := rl.requests[ip]
	recentRequests := []time.Time{}
	for _, t := range requests {
		if t.After(oneMinuteAgo) {
			recentRequests = append(recentRequests, t)
		}
	}
	
	// Check if limit exceeded
	if len(recentRequests) >= maxRequestsPerMinute {
		return false
	}
	
	// Add current request
	recentRequests = append(recentRequests, now)
	rl.requests[ip] = recentRequests
	return true
}

// IsEnabled returns true if admin is configured
func (h *Handler) IsEnabled() bool {
	return h.config.Username != "" && h.config.Password != ""
}

// setSecurityHeaders sets security headers on admin responses
func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:;")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// logAdminAction logs admin actions for audit purposes
func (h *Handler) logAdminAction(action, ip, details string) {
	log.Printf("[ADMIN_AUDIT] action=%s ip=%s details=%s", action, ip, details)
	// Could also send to structured logging system if needed
}

// BasicAuthMiddleware protects routes with HTTP Basic Authentication, rate limiting, and audit logging
func (h *Handler) BasicAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set security headers on all admin responses
		setSecurityHeaders(w)
		
		if !h.IsEnabled() {
			http.Error(w, "Admin interface not configured", http.StatusServiceUnavailable)
			return
		}

		ip := getClientIP(r)
		
		// Check if IP is locked out
		if h.rateLimiter.isIPLocked(ip) {
			h.logAdminAction("auth_blocked_locked", ip, "IP locked out")
			http.Error(w, "Too many failed authentication attempts. Please try again later.", http.StatusTooManyRequests)
			w.Header().Set("Retry-After", strconv.Itoa(int(bruteForceLockoutDuration.Seconds())))
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Clotilde Admin"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Use constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(h.config.Username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(h.config.Password)) == 1

		if !usernameMatch || !passwordMatch {
			// Record failed authentication attempt
			h.rateLimiter.recordFailedAuth(ip)
			h.logAdminAction("auth_failed", ip, "Invalid credentials")
			
			w.Header().Set("WWW-Authenticate", `Basic realm="Clotilde Admin"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check request rate limit
		if !h.rateLimiter.checkRequestRate(ip) {
			h.logAdminAction("rate_limit_exceeded", ip, r.URL.Path)
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			w.Header().Set("Retry-After", "60")
			return
		}

		// Successful authentication
		h.logAdminAction("auth_success", ip, r.URL.Path)
		
		next.ServeHTTP(w, r)
	}
}

// HandleDashboard serves the admin dashboard HTML page
func (h *Handler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	ip := getClientIP(r)
	h.logAdminAction("dashboard_view", ip, "")
	
	// Generate CSRF token for this session
	csrfToken := h.generateCSRFToken(r)
	
	// Inject CSRF token into dashboard HTML
	html := dashboardHTML
	if csrfToken != "" {
		// Replace placeholder with actual token
		html = replaceCSRFToken(html, csrfToken)
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// replaceCSRFToken injects CSRF token into dashboard HTML
func replaceCSRFToken(html, token string) string {
	// Replace {{CSRF_TOKEN}} placeholder with actual token
	return strings.ReplaceAll(html, "{{CSRF_TOKEN}}", token)
}

// HandleLogs returns log entries as JSON, querying Cloud Logging when needed
func (h *Handler) HandleLogs(w http.ResponseWriter, r *http.Request) {
	ip := getClientIP(r)
	h.logAdminAction("logs_view", ip, r.URL.RawQuery)
	query := r.URL.Query()

	// Pagination
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(query.Get("offset"))

	// Filters
	model := query.Get("model")
	status := query.Get("status")
	source := query.Get("source") // "memory", "cloud", or "both"

	var startDate, endDate *time.Time
	if start := query.Get("start_date"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			startDate = &t
		}
	}
	if end := query.Get("end_date"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			// End of day
			endOfDay := t.Add(24*time.Hour - time.Second)
			endDate = &endOfDay
		}
	}

	var entries []logging.LogEntry
	var total int
	var fromCloud bool

	bufferCount := h.logger.GetCount()

	// Determine if we need to query Cloud Logging
	needsCloudLogging := false
	if source == "cloud" || source == "both" {
		needsCloudLogging = true
	} else if source == "" {
		// Auto-detect: use Cloud Logging if:
		// 1. Buffer is empty
		// 2. Offset is beyond buffer capacity
		// 3. Date range extends beyond buffer (more than 1 hour ago)
		if bufferCount == 0 {
			needsCloudLogging = true
		} else if offset >= bufferCount {
			needsCloudLogging = true
		} else if startDate != nil && time.Since(*startDate) > time.Hour {
			needsCloudLogging = true
		}
	}

	// Get entries from in-memory buffer
	if source != "cloud" {
		if model != "" || status != "" || startDate != nil || endDate != nil {
			entries = h.logger.GetEntriesFiltered(limit, offset, model, status, startDate, endDate)
		} else {
			entries = h.logger.GetEntries(limit, offset)
		}
		total = bufferCount
	}

	// Query Cloud Logging if needed
	if needsCloudLogging {
		projectID := logging.GetProjectID()
		if projectID != "" {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()

			queryOpts := logging.QueryOptions{
				Limit:     limit * 2, // Get more to account for filtering
				Offset:    0,         // We'll handle offset after filtering
				Model:     model,
				Status:    status,
				StartDate: startDate,
				EndDate:   endDate,
			}

			cloudEntries, cloudTotal, err := logging.QueryCloudLogs(ctx, projectID, queryOpts)
			if err == nil && len(cloudEntries) > 0 {
				fromCloud = true

				// Apply pagination to filtered cloud entries
				if offset < len(cloudEntries) {
					end := offset + limit
					if end > len(cloudEntries) {
						end = len(cloudEntries)
					}
					
					if source == "both" && len(entries) > 0 {
						// Merge with in-memory entries (deduplicate by ID)
						entryMap := make(map[string]bool)
						for _, e := range entries {
							entryMap[e.ID] = true
						}
						for _, e := range cloudEntries[offset:end] {
							if !entryMap[e.ID] {
								entries = append(entries, e)
							}
						}
						total = bufferCount + cloudTotal
					} else {
						entries = cloudEntries[offset:end]
						total = cloudTotal
					}
				} else {
					// Offset beyond available entries
					entries = []logging.LogEntry{}
					total = cloudTotal
				}
			} else if err != nil {
				log.Printf("Error querying Cloud Logging: %v", err)
			}
		}
	}

	response := struct {
		Entries   []logging.LogEntry `json:"entries"`
		Count     int                 `json:"count"`
		Offset    int                 `json:"offset"`
		Limit     int                 `json:"limit"`
		Total     int                 `json:"total"`
		FromCloud bool               `json:"from_cloud,omitempty"`
	}{
		Entries:   entries,
		Count:     len(entries),
		Offset:    offset,
		Limit:     limit,
		Total:     total,
		FromCloud: fromCloud,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleStats returns aggregated statistics as JSON
func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	ip := getClientIP(r)
	h.logAdminAction("stats_view", ip, "")
	
	stats := h.logger.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleGetConfig returns the current runtime configuration as JSON
func (h *Handler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := getClientIP(r)
	h.logAdminAction("config_view", ip, "")

	config := GetConfig()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		log.Printf("Error encoding config: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HandleSetConfig updates the runtime configuration from JSON POST body
func (h *Handler) HandleSetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := getClientIP(r)

	// Validate CSRF token
	csrfToken := r.Header.Get("X-CSRF-Token")
	if !h.validateCSRFToken(csrfToken, r) {
		h.logAdminAction("config_update_failed", ip, "Invalid CSRF token")
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}

	// Limit request body size
	limitedReader := io.LimitReader(r.Body, maxConfigBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		h.logAdminAction("config_update_failed", ip, "Failed to read body")
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Check if body is too large
	if len(body) >= maxConfigBodySize {
		h.logAdminAction("config_update_failed", ip, "Request body too large")
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	var newConfig RuntimeConfig
	if err := json.Unmarshal(body, &newConfig); err != nil {
		h.logAdminAction("config_update_failed", ip, "Invalid JSON")
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate base system prompt size (prefer BaseSystemPrompt, fallback to SystemPrompt for legacy)
	basePrompt := newConfig.BaseSystemPrompt
	if basePrompt == "" {
		basePrompt = newConfig.SystemPrompt
	}
	if basePrompt != "" && len(basePrompt) > maxSystemPromptSize {
		h.logAdminAction("config_update_failed", ip, "Base system prompt too large")
		http.Error(w, "Base system prompt exceeds maximum size", http.StatusBadRequest)
		return
	}
	
	// Validate category prompts size
	for category, prompt := range newConfig.CategoryPrompts {
		if prompt != "" && len(prompt) > maxSystemPromptSize {
			h.logAdminAction("config_update_failed", ip, fmt.Sprintf("Category prompt %s too large", category))
			http.Error(w, fmt.Sprintf("Category prompt %s exceeds maximum size", category), http.StatusBadRequest)
			return
		}
	}

	if err := SetConfig(newConfig); err != nil {
		h.logAdminAction("config_update_failed", ip, err.Error())
		log.Printf("Error setting config: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Log successful config update
	h.logAdminAction("config_updated", ip, fmt.Sprintf("standard_model=%s premium_model=%s prompt_len=%d", 
		newConfig.StandardModel, newConfig.PremiumModel, len(newConfig.SystemPrompt)))

	// Return updated config
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newConfig)
}

// RegisterRoutes registers admin routes on the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Protected admin routes
	// Handle both /admin and /admin/ for better compatibility
	mux.HandleFunc("/admin", h.BasicAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Redirect /admin to /admin/ for consistency
		if r.URL.Path == "/admin" {
			http.Redirect(w, r, "/admin/", http.StatusMovedPermanently)
			return
		}
		h.HandleDashboard(w, r)
	}))
	mux.HandleFunc("/admin/", h.BasicAuthMiddleware(h.HandleDashboard))
	mux.HandleFunc("/admin/logs", h.BasicAuthMiddleware(h.HandleLogs))
	mux.HandleFunc("/admin/stats", h.BasicAuthMiddleware(h.HandleStats))
	mux.HandleFunc("/admin/config", h.BasicAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.HandleGetConfig(w, r)
		} else if r.Method == http.MethodPost {
			h.HandleSetConfig(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
}

