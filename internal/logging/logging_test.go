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

func TestLogger_NegativeOffsetDoesNotPanic(t *testing.T) {
	logger := NewLogger(4)
	logger.Add(LogEntry{ID: "1", Timestamp: time.Now(), Status: "success"})

	if got := logger.GetEntries(5, -3); len(got) != 1 {
		t.Fatalf("GetEntries(-3) len = %d, want 1", len(got))
	}
	if got := logger.GetEntriesFiltered(5, -3, "", "", nil, nil); len(got) != 1 {
		t.Fatalf("GetEntriesFiltered(-3) len = %d, want 1", len(got))
	}
}

func TestLogger_EmptyModelNotCountedAsPremium(t *testing.T) {
	logger := NewLogger(4)
	logger.Add(LogEntry{ID: "err-1", Timestamp: time.Now(), Status: "error", Model: ""})
	stats := logger.GetStats()
	if stats.ModelUsage.Premium != 0 || stats.ModelUsage.Standard != 0 {
		t.Fatalf("empty model should not increment model usage: %+v", stats.ModelUsage)
	}
}

func TestRedactPII_CardAndCPFBeforePhone(t *testing.T) {
	// Test compiled regex behavior directly without mutating sync.Once global state.
	cardText := "Cartão 1234-5678-9012-3456 e 1234567890123456"
	out := creditCardPattern.ReplaceAllString(cardText, "[CARD_REDACTED]")
	out = cpfPattern.ReplaceAllString(out, "[CPF_REDACTED]")
	out = phonePattern.ReplaceAllString(out, "[PHONE_REDACTED]")
	if out != "Cartão [CARD_REDACTED] e [CARD_REDACTED]" {
		t.Fatalf("card redaction leaked digits or misclassified: %q", out)
	}

	cpfText := "CPF 123.456.789-00 e 12345678900"
	out = creditCardPattern.ReplaceAllString(cpfText, "[CARD_REDACTED]")
	out = cpfPattern.ReplaceAllString(out, "[CPF_REDACTED]")
	out = phonePattern.ReplaceAllString(out, "[PHONE_REDACTED]")
	if out != "CPF [CPF_REDACTED] e [CPF_REDACTED]" {
		t.Fatalf("CPF redaction failed: %q", out)
	}
}

func TestStopPeriodicFlush_Idempotent(t *testing.T) {
	// Calling StopPeriodicFlush multiple times must never panic.
	StopPeriodicFlush()
	StopPeriodicFlush()
}
