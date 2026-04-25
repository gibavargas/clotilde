package logging

import (
	"testing"
	"time"
)

func TestNewLogger_AddStoresRawInputAndLegacyInput(t *testing.T) {
	logger := NewLogger(4)

	entry := LogEntry{
		ID:            "req-1",
		Timestamp:     time.Now(),
		IPHash:        "hash",
		MessageLength: 28,
		Model:         "gpt-4o-mini",
		ResponseTime:  120,
		TokenEstimate: 7,
		Status:        "success",
		RawInput:      "Qual é a capital do Brasil?",
		Input:         "Qual e a capital do Brasil?",
		Output:        "Brasília.",
	}

	logger.Add(entry)

	entries := logger.GetEntries(1, 0)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.RawInput != entry.RawInput {
		t.Fatalf("expected raw input %q, got %q", entry.RawInput, got.RawInput)
	}
	if got.Input != entry.Input {
		t.Fatalf("expected legacy input %q, got %q", entry.Input, got.Input)
	}
}
