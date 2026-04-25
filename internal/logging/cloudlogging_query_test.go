package logging

import (
	"testing"

	loggingpb "cloud.google.com/go/logging/apiv2/loggingpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConvertCloudLogEntryPrefersRawInputAndFallsBackToInput(t *testing.T) {
	t.Run("prefers raw_input", func(t *testing.T) {
		payload, err := structpb.NewStruct(map[string]interface{}{
			"request_id": "req-raw",
			"raw_input":  "Qual é a capital do Brasil?",
			"input":      "Qual e a capital do Brasil?",
			"output":     "Brasília.",
			"status":     "success",
		})
		if err != nil {
			t.Fatalf("failed to build payload: %v", err)
		}

		cloudEntry := &loggingpb.LogEntry{
			Payload: &loggingpb.LogEntry_JsonPayload{JsonPayload: payload},
		}

		entry := convertCloudLogEntry(cloudEntry)
		if entry == nil {
			t.Fatal("expected entry, got nil")
		}
		if entry.RawInput != "Qual é a capital do Brasil?" {
			t.Fatalf("expected raw_input to win, got %q", entry.RawInput)
		}
		if entry.Input != "Qual e a capital do Brasil?" {
			t.Fatalf("expected legacy input to be preserved, got %q", entry.Input)
		}
	})

	t.Run("falls back to input", func(t *testing.T) {
		payload, err := structpb.NewStruct(map[string]interface{}{
			"request_id": "req-legacy",
			"input":      "Texto legado",
			"output":     "Resposta",
			"status":     "success",
		})
		if err != nil {
			t.Fatalf("failed to build payload: %v", err)
		}

		cloudEntry := &loggingpb.LogEntry{
			Payload: &loggingpb.LogEntry_JsonPayload{JsonPayload: payload},
		}

		entry := convertCloudLogEntry(cloudEntry)
		if entry == nil {
			t.Fatal("expected entry, got nil")
		}
		if entry.RawInput != "Texto legado" {
			t.Fatalf("expected raw_input fallback to input, got %q", entry.RawInput)
		}
		if entry.Input != "Texto legado" {
			t.Fatalf("expected legacy input to be preserved, got %q", entry.Input)
		}
	})
}
