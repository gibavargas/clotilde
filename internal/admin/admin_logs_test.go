package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clotilde/carplay-assistant/internal/logging"
)

func TestHandleLogs_ReturnsRawInput(t *testing.T) {
	logger := logging.NewLogger(8)
	logger.Add(logging.LogEntry{
		ID:            "req-1",
		Timestamp:     time.Now(),
		IPHash:        "ip",
		MessageLength: 28,
		Model:         "gpt-4o-mini",
		ResponseTime:  42,
		TokenEstimate: 7,
		Status:        "success",
		RawInput:      "Qual é a capital do Brasil?",
		Input:         "Qual e a capital do Brasil?",
		Output:        "Brasília.",
	})

	handler := NewHandler(logger)
	req := httptest.NewRequest(http.MethodGet, "/admin/logs?limit=1&offset=0", nil)
	rr := httptest.NewRecorder()

	handler.HandleLogs(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Entries []logging.LogEntry `json:"entries"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
	}
	if resp.Entries[0].RawInput != "Qual é a capital do Brasil?" {
		t.Fatalf("expected raw_input to be returned, got %q", resp.Entries[0].RawInput)
	}
}
