package admin

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
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

// Handler handles admin routes
type Handler struct {
	config     Config
	logger     *logging.Logger
	csrfTokens map[string]time.Time
	csrfMutex  sync.RWMutex
}

// csrfToken represents a CSRF token with expiration
type csrfToken struct {
	Token     string
	ExpiresAt time.Time
}

// NewHandler creates a new admin handler
func NewHandler(logger *logging.Logger) *Handler {
	h := &Handler{
		config: Config{
			Username: os.Getenv("ADMIN_USER"),
			Password: os.Getenv("ADMIN_PASSWORD"),
		},
		logger:     logger,
		csrfTokens: make(map[string]time.Time),
	}
	// Start cleanup goroutine for expired tokens
	go h.cleanupExpiredTokens()
	return h
}

// generateCSRFToken creates a new CSRF token
func (h *Handler) generateCSRFToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("Error generating CSRF token: %v", err)
		return ""
	}
	token := base64.URLEncoding.EncodeToString(bytes)
	
	h.csrfMutex.Lock()
	h.csrfTokens[token] = time.Now().Add(24 * time.Hour) // Token valid for 24 hours
	h.csrfMutex.Unlock()
	
	return token
}

// validateCSRFToken checks if a CSRF token is valid and consumes it (single-use)
func (h *Handler) validateCSRFToken(token string) bool {
	if token == "" {
		return false
	}
	
	h.csrfMutex.Lock()
	defer h.csrfMutex.Unlock()
	
	expiresAt, exists := h.csrfTokens[token]
	if !exists {
		return false
	}
	
	// Check if token is expired
	if time.Now().After(expiresAt) {
		// Token expired, remove it
		delete(h.csrfTokens, token)
		return false
	}
	
	// Token is valid - consume it (delete) to prevent replay attacks
	delete(h.csrfTokens, token)
	return true
}

// cleanupExpiredTokens periodically removes expired tokens
func (h *Handler) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		h.csrfMutex.Lock()
		now := time.Now()
		for token, expiresAt := range h.csrfTokens {
			if now.After(expiresAt) {
				delete(h.csrfTokens, token)
			}
		}
		h.csrfMutex.Unlock()
	}
}

// IsEnabled returns true if admin is configured
func (h *Handler) IsEnabled() bool {
	return h.config.Username != "" && h.config.Password != ""
}

// BasicAuthMiddleware protects routes with HTTP Basic Authentication
func (h *Handler) BasicAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.IsEnabled() {
			http.Error(w, "Admin interface not configured", http.StatusServiceUnavailable)
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
			w.Header().Set("WWW-Authenticate", `Basic realm="Clotilde Admin"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// HandleDashboard serves the admin dashboard HTML page
func (h *Handler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	// Generate CSRF token for this session
	csrfToken := h.generateCSRFToken()
	
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

	// Validate CSRF token
	csrfToken := r.Header.Get("X-CSRF-Token")
	if !h.validateCSRFToken(csrfToken) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}

	var newConfig RuntimeConfig
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := SetConfig(newConfig); err != nil {
		log.Printf("Error setting config: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return updated config
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newConfig)
}

// RegisterRoutes registers admin routes on the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Protected admin routes
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

