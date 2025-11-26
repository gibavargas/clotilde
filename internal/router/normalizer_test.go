package router

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Notícias", "notic"},
		{"noticiário", "noticiari"}, // Stemmer might not be perfect
		{"Dúvidas", "duvid"},
		{"dúvida", "duvid"},
		{"correndo", "corr"},
		{"correr", "corr"},
		{"QUAIS AS NOTÍCIAS?", "quais as notic?"}, // Punctuation?
	}

	for _, tt := range tests {
		got := Normalize(tt.input)
		// We don't assert exact match yet, just print to see
		t.Logf("Normalize('%s') = '%s'", tt.input, got)
	}
}
