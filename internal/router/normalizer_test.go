package router

import (
	"testing"
)

// TestNormalize pins the normalization contract: lowercase, accent-stripped,
// punctuation-free, RSLP-lite stemmed. These assertions were previously
// log-only, which let two contradicting expectation tables coexist invisibly.
func TestNormalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Notícias", "noticia"},
		{"noticiário", "noticiario"},
		{"Dúvidas", "duvida"},
		{"dúvida", "duvida"},
		{"QUAIS AS NOTÍCIAS?", "quai as noticia"}, // plural stem also drops -s on "quais"
		{"correndo", "corre"},                     // -ndo words get the gerund suffix stripped
		{"VamosCorrer", "vamoscorr"},
		{"Números 1, 2 e 3!", "numero 1 2 e 3"}, // stopword "e" is not filtered by Normalize
	}

	for _, tt := range tests {
		if got := Normalize(tt.input); got != tt.expected {
			t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// Invariant: normalization is idempotent (keyword matching relies on it).
func TestNormalizeIdempotent(t *testing.T) {
	inputs := []string{"Notícias", "Olá, como vai?", "Dúvidas sobre pedágio!", "Quem foi Santos Dumont?"}
	for _, in := range inputs {
		once := Normalize(in)
		if twice := Normalize(once); twice != once {
			t.Errorf("Normalize not idempotent for %q: %q -> %q", in, once, twice)
		}
	}
}
