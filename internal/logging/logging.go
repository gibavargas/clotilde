package logging

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LogEntry represents a single request/response log entry
type LogEntry struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	IPHash        string    `json:"ip_hash"`
	MessageLength int       `json:"message_length"`
	Model         string    `json:"model"`
	Category      string    `json:"category,omitempty"` // Router category (web_search, complex, factual, etc.)
	ResponseTime  int64     `json:"response_time_ms"`
	TokenEstimate int       `json:"token_estimate"`
	Status        string    `json:"status"` // "success" or "error"
	ErrorMessage  string    `json:"error_message,omitempty"`
	Input         string    `json:"input,omitempty"`  // Full user input (question)
	Output        string    `json:"output,omitempty"` // Full AI response
}

// Stats represents aggregated statistics
type Stats struct {
	TotalRequests      int64     `json:"total_requests"`
	TotalRequestsToday int64     `json:"total_requests_today"`
	AvgResponseTimeMs  float64   `json:"avg_response_time_ms"`
	ErrorRate          float64   `json:"error_rate"`
	ModelUsage         ModelUsage `json:"model_usage"`
	Uptime             string    `json:"uptime"`
	LastRequestTime    *time.Time `json:"last_request_time,omitempty"`
}

// ModelUsage tracks usage by model
type ModelUsage struct {
	Standard int64 `json:"standard"` // Fast/cheap models (gpt-4o-mini, Claude Haiku, etc.)
	Premium  int64 `json:"premium"`  // Powerful/expensive models (gpt-4o, Claude Sonnet, etc.)
	
	// Legacy fields for backward compatibility
	Nano int64 `json:"nano"` // Deprecated: use Standard
	Full int64 `json:"full"` // Deprecated: use Premium
}

// Logger is a thread-safe ring buffer logger
type Logger struct {
	entries       []LogEntry
	capacity      int
	head          int
	count         int
	mu            sync.RWMutex
	startTime     time.Time
	totalCount    int64
	errorCount    int64
	totalTime     int64
	standardCount int64 // Fast/cheap models
	premiumCount  int64 // Powerful/expensive models
}

var (
	globalLogger *Logger
	once         sync.Once
)

// GetLogger returns the singleton logger instance
func GetLogger() *Logger {
	once.Do(func() {
		capacity := 1000
		if envSize := os.Getenv("LOG_BUFFER_SIZE"); envSize != "" {
			if size, err := strconv.Atoi(envSize); err == nil && size > 0 {
				capacity = size
			}
		}
		globalLogger = &Logger{
			entries:   make([]LogEntry, capacity),
			capacity:  capacity,
			startTime: time.Now(),
		}
	})
	return globalLogger
}

// Add adds a new log entry to the ring buffer and Cloud Logging
func (l *Logger) Add(entry LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Store entry at head position
	l.entries[l.head] = entry
	l.head = (l.head + 1) % l.capacity
	if l.count < l.capacity {
		l.count++
	}

	// Update aggregates
	l.totalCount++
	l.totalTime += entry.ResponseTime
	if entry.Status == "error" {
		l.errorCount++
	}

	// Track model usage - categorize models as standard (fast/cheap) or premium (powerful/expensive)
	// Standard models: gpt-4o-mini, Claude Haiku variants, gpt-3.5-turbo, etc.
	// Premium models: gpt-4o, gpt-5, Claude Sonnet, Claude Opus, o1, o3, etc.
	modelLower := strings.ToLower(entry.Model)
	isStandard := strings.Contains(modelLower, "mini") ||
		strings.Contains(modelLower, "haiku") ||
		strings.Contains(modelLower, "nano") ||
		strings.Contains(modelLower, "3.5-turbo")
	
	if isStandard {
		l.standardCount++
	} else {
		l.premiumCount++
	}

	// Also send to Cloud Logging for persistence
	go func(e LogEntry) {
		cloudLogger := GetCloudLogger()
		if cloudLogger.IsEnabled() {
			cloudLogger.Log(e)
		}
	}(entry)
}

// GetEntries returns log entries with pagination (newest first)
func (l *Logger) GetEntries(limit, offset int) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.count == 0 {
		return []LogEntry{}
	}

	// Calculate available entries
	available := l.count - offset
	if available <= 0 {
		return []LogEntry{}
	}

	if limit <= 0 || limit > available {
		limit = available
	}

	result := make([]LogEntry, limit)
	
	// Start from most recent entry and go backwards
	startIdx := (l.head - 1 - offset + l.capacity) % l.capacity
	
	for i := 0; i < limit; i++ {
		idx := (startIdx - i + l.capacity) % l.capacity
		result[i] = l.entries[idx]
	}

	return result
}

// GetEntriesFiltered returns filtered log entries
func (l *Logger) GetEntriesFiltered(limit, offset int, model, status string, startDate, endDate *time.Time) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.count == 0 {
		return []LogEntry{}
	}

	var filtered []LogEntry
	
	// Start from most recent entry and go backwards
	startIdx := (l.head - 1 + l.capacity) % l.capacity
	
	for i := 0; i < l.count; i++ {
		idx := (startIdx - i + l.capacity) % l.capacity
		entry := l.entries[idx]
		
		// Skip entries with zero timestamp (empty slots in ring buffer)
		if entry.Timestamp.IsZero() {
			continue
		}
		
		// Apply filters
		if model != "" && entry.Model != model {
			continue
		}
		if status != "" && entry.Status != status {
			continue
		}
		if startDate != nil && entry.Timestamp.Before(*startDate) {
			continue
		}
		if endDate != nil && entry.Timestamp.After(*endDate) {
			continue
		}
		
		filtered = append(filtered, entry)
	}

	// Apply pagination
	if offset >= len(filtered) {
		return []LogEntry{}
	}
	
	filtered = filtered[offset:]
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}

	return filtered
}

// GetStats returns aggregated statistics
func (l *Logger) GetStats() Stats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := Stats{
		TotalRequests: l.totalCount,
		ModelUsage: ModelUsage{
			Standard: l.standardCount,
			Premium:  l.premiumCount,
			// Legacy fields for backward compatibility
			Nano: l.standardCount,
			Full: l.premiumCount,
		},
		Uptime: time.Since(l.startTime).Round(time.Second).String(),
	}

	// Calculate today's requests
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	for i := 0; i < l.count; i++ {
		idx := (l.head - 1 - i + l.capacity) % l.capacity
		entry := l.entries[idx]
		// Skip entries with zero timestamp (empty slots in ring buffer)
		if entry.Timestamp.IsZero() {
			continue
		}
		if entry.Timestamp.After(todayStart) {
			stats.TotalRequestsToday++
		}
	}

	// Calculate averages
	if l.totalCount > 0 {
		stats.AvgResponseTimeMs = float64(l.totalTime) / float64(l.totalCount)
		stats.ErrorRate = float64(l.errorCount) / float64(l.totalCount) * 100
	}

	// Get last request time (find most recent valid entry)
	if l.count > 0 {
		for i := 0; i < l.count; i++ {
			idx := (l.head - 1 - i + l.capacity) % l.capacity
			entry := l.entries[idx]
			if !entry.Timestamp.IsZero() {
				stats.LastRequestTime = &entry.Timestamp
				break
			}
		}
	}

	return stats
}

// GetCount returns the current number of entries
func (l *Logger) GetCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.count
}

