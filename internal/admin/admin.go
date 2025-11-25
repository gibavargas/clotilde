package admin

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
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
	config Config
	logger *logging.Logger
}

// NewHandler creates a new admin handler
func NewHandler(logger *logging.Logger) *Handler {
	return &Handler{
		config: Config{
			Username: os.Getenv("ADMIN_USER"),
			Password: os.Getenv("ADMIN_PASSWORD"),
		},
		logger: logger,
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

// HandleLogs returns log entries as JSON
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
	if model != "" || status != "" || startDate != nil || endDate != nil {
		entries = h.logger.GetEntriesFiltered(limit, offset, model, status, startDate, endDate)
	} else {
		entries = h.logger.GetEntries(limit, offset)
	}

	response := struct {
		Entries []logging.LogEntry `json:"entries"`
		Count   int                `json:"count"`
		Offset  int                `json:"offset"`
		Limit   int                `json:"limit"`
		Total   int                `json:"total"`
	}{
		Entries: entries,
		Count:   len(entries),
		Offset:  offset,
		Limit:   limit,
		Total:   h.logger.GetCount(),
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

// RegisterRoutes registers admin routes on the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Protected admin routes
	mux.HandleFunc("/admin/", h.BasicAuthMiddleware(h.HandleDashboard))
	mux.HandleFunc("/admin/logs", h.BasicAuthMiddleware(h.HandleLogs))
	mux.HandleFunc("/admin/stats", h.BasicAuthMiddleware(h.HandleStats))
}

