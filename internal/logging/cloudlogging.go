package logging

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/logging"
)

// CloudLogger handles Google Cloud Logging integration
type CloudLogger struct {
	client  *logging.Client
	logger  *logging.Logger
	enabled bool
	mu      sync.RWMutex
}

var (
	cloudLogger     *CloudLogger
	cloudLoggerOnce sync.Once
	flushTicker     *time.Ticker
	flushStop       chan bool
)

// GetCloudLogger returns the singleton Cloud Logger instance
func GetCloudLogger() *CloudLogger {
	cloudLoggerOnce.Do(func() {
		cloudLogger = &CloudLogger{
			enabled: false,
		}

		projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
		if projectID == "" {
			log.Printf("Cloud Logging disabled: GOOGLE_CLOUD_PROJECT not set")
			return
		}

		ctx := context.Background()
		client, err := logging.NewClient(ctx, projectID)
		if err != nil {
			log.Printf("Cloud Logging disabled: failed to create client: %v", err)
			return
		}

		cloudLogger.client = client
		cloudLogger.logger = client.Logger("clotilde-requests")
		cloudLogger.enabled = true
		log.Printf("Cloud Logging enabled for project: %s", projectID)

		// Start periodic flush (every 10 seconds) to ensure logs are sent before container restarts
		startPeriodicFlush()
	})
	return cloudLogger
}

// startPeriodicFlush starts a goroutine that flushes logs every 10 seconds
func startPeriodicFlush() {
	flushTicker = time.NewTicker(10 * time.Second)
	flushStop = make(chan bool)

	go func() {
		for {
			select {
			case <-flushTicker.C:
				cloudLogger := GetCloudLogger()
				if cloudLogger.IsEnabled() {
					if err := cloudLogger.Flush(); err != nil {
						log.Printf("Error flushing Cloud Logging: %v", err)
					}
				}
			case <-flushStop:
				flushTicker.Stop()
				return
			}
		}
	}()
}

// StopPeriodicFlush stops the periodic flush goroutine
func StopPeriodicFlush() {
	if flushStop != nil {
		close(flushStop)
	}
}

// IsEnabled returns whether Cloud Logging is active
func (cl *CloudLogger) IsEnabled() bool {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.enabled
}

// Log writes a log entry to Cloud Logging
func (cl *CloudLogger) Log(entry LogEntry) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	if !cl.enabled || cl.logger == nil {
		return
	}

	// Create structured log payload
	payload := map[string]interface{}{
		"request_id":       entry.ID,
		"timestamp":        entry.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
		"ip_hash":          entry.IPHash,
		"message_length":   entry.MessageLength,
		"model":            entry.Model,
		"response_time_ms": entry.ResponseTime,
		"token_estimate":   entry.TokenEstimate,
		"status":           entry.Status,
		"input":            entry.Input,
		"output":           entry.Output,
	}

	if entry.ErrorMessage != "" {
		payload["error_message"] = entry.ErrorMessage
	}

	// Determine severity based on status
	severity := logging.Info
	if entry.Status == "error" {
		severity = logging.Error
	}

	cl.logger.Log(logging.Entry{
		Payload:  payload,
		Severity: severity,
	})
}

// Close flushes and closes the Cloud Logging client
func (cl *CloudLogger) Close() error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if cl.client != nil {
		if err := cl.client.Close(); err != nil {
			return err
		}
		cl.enabled = false
	}
	return nil
}

// Flush ensures all buffered logs are sent
func (cl *CloudLogger) Flush() error {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	if cl.client != nil {
		return cl.logger.Flush()
	}
	return nil
}

