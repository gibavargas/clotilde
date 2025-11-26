package logging

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	loggingv2 "cloud.google.com/go/logging/apiv2"
	"cloud.google.com/go/logging/apiv2/loggingpb"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/structpb"
)

// QueryOptions contains parameters for querying Cloud Logging
type QueryOptions struct {
	Limit     int
	Offset    int
	Model     string
	Status    string
	StartDate *time.Time
	EndDate   *time.Time
}

// QueryCloudLogs queries Cloud Logging for historical log entries
func QueryCloudLogs(ctx context.Context, projectID string, opts QueryOptions) ([]LogEntry, int, error) {
	if projectID == "" {
		return []LogEntry{}, 0, fmt.Errorf("project ID not available")
	}

	// Create Logging Service client for queries
	client, err := loggingv2.NewClient(ctx)
	if err != nil {
		return []LogEntry{}, 0, fmt.Errorf("failed to create logging client: %w", err)
	}
	defer client.Close()

	// Build the filter query
	filter := fmt.Sprintf(`resource.type="cloud_run_revision" AND logName="projects/%s/logs/clotilde-requests"`, projectID)

	// Add filters
	if opts.StartDate != nil {
		filter += fmt.Sprintf(` AND timestamp>="%s"`, opts.StartDate.Format(time.RFC3339))
	}
	if opts.EndDate != nil {
		filter += fmt.Sprintf(` AND timestamp<="%s"`, opts.EndDate.Format(time.RFC3339))
	}
	if opts.Model != "" {
		filter += fmt.Sprintf(` AND jsonPayload.model="%s"`, opts.Model)
	}
	if opts.Status != "" {
		filter += fmt.Sprintf(` AND jsonPayload.status="%s"`, opts.Status)
	}

	// Build the request
	req := &loggingpb.ListLogEntriesRequest{
		ResourceNames: []string{fmt.Sprintf("projects/%s", projectID)},
		Filter:        filter,
		OrderBy:       "timestamp desc",
		PageSize:      int32(opts.Limit + opts.Offset + 100), // Get more to account for filtering
	}

	// Query Cloud Logging
	it := client.ListLogEntries(ctx, req)
	
	var entries []LogEntry
	var totalCount int

	for {
		entry, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating Cloud Logging entries: %v", err)
			break
		}

		totalCount++

		// Skip entries before offset
		if totalCount <= opts.Offset {
			continue
		}

		// Stop if we've reached the limit
		if opts.Limit > 0 && len(entries) >= opts.Limit {
			break
		}

		// Convert Cloud Logging entry to LogEntry
		logEntry := convertCloudLogEntry(entry)
		if logEntry != nil {
			entries = append(entries, *logEntry)
		}
	}

	return entries, totalCount, nil
}

// convertCloudLogEntry converts a Cloud Logging entry to our LogEntry format
func convertCloudLogEntry(cloudEntry *loggingpb.LogEntry) *LogEntry {
	entry := &LogEntry{}

	// Extract timestamp
	if cloudEntry.Timestamp != nil {
		entry.Timestamp = cloudEntry.Timestamp.AsTime()
	}

	// Extract JSON payload (it's a oneof field)
	var payload map[string]interface{}
	switch p := cloudEntry.Payload.(type) {
	case *loggingpb.LogEntry_JsonPayload:
		if p.JsonPayload != nil {
			// Convert structpb.Struct to map[string]interface{}
			payloadMap := make(map[string]interface{})
			for k, v := range p.JsonPayload.Fields {
				payloadMap[k] = extractValue(v)
			}
			payload = payloadMap
		} else {
			return nil
		}
	default:
		// Not a JSON payload, skip
		return nil
	}

	// Extract fields from payload
	if id, ok := payload["request_id"].(string); ok {
		entry.ID = id
	}
	if ipHash, ok := payload["ip_hash"].(string); ok {
		entry.IPHash = ipHash
	}
	if msgLen, ok := payload["message_length"].(float64); ok {
		entry.MessageLength = int(msgLen)
	}
	if model, ok := payload["model"].(string); ok {
		entry.Model = model
	}
	if category, ok := payload["category"].(string); ok {
		entry.Category = category
	}
	if respTime, ok := payload["response_time_ms"].(float64); ok {
		entry.ResponseTime = int64(respTime)
	}
	if tokenEst, ok := payload["token_estimate"].(float64); ok {
		entry.TokenEstimate = int(tokenEst)
	}
	if status, ok := payload["status"].(string); ok {
		entry.Status = status
	}
	if errMsg, ok := payload["error_message"].(string); ok {
		entry.ErrorMessage = errMsg
	}
	if input, ok := payload["input"].(string); ok {
		entry.Input = input
	}
	if output, ok := payload["output"].(string); ok {
		entry.Output = output
	}

	return entry
}

// extractValue extracts a value from a protobuf Value
func extractValue(v *structpb.Value) interface{} {
	switch v := v.Kind.(type) {
	case *structpb.Value_StringValue:
		return v.StringValue
	case *structpb.Value_NumberValue:
		return v.NumberValue
	case *structpb.Value_BoolValue:
		return v.BoolValue
	default:
		return nil
	}
}

// GetProjectID returns the project ID from environment
func GetProjectID() string {
	return os.Getenv("GOOGLE_CLOUD_PROJECT")
}
