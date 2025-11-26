package router

import (
	"testing"

	"github.com/clotilde/carplay-assistant/internal/admin"
)

func TestRoute(t *testing.T) {
	// Initialize default config for testing
	admin.SetDefaultConfig("System prompt %s")

	tests := []struct {
		name              string
		question          string
		expectedCategory  Category
		expectedWebSearch bool
	}{
		{
			name:              "Web Search - News",
			question:          "Quais as últimas notícias do Brasil?",
			expectedCategory:  CategoryWebSearch,
			expectedWebSearch: true,
		},
		{
			name:              "Web Search - Weather",
			question:          "Qual a previsão do tempo para amanhã?",
			expectedCategory:  CategoryWebSearch,
			expectedWebSearch: true,
		},
		{
			name:              "Complex - Explain",
			question:          "Explique a teoria da relatividade",
			expectedCategory:  CategoryComplex,
			expectedWebSearch: false,
		},
		{
			name:              "Complex - Compare",
			question:          "Compare Python e Go",
			expectedCategory:  CategoryComplex,
			expectedWebSearch: false,
		},
		{
			name:              "Factual - Who is",
			question:          "Quem é o presidente do Brasil?",
			expectedCategory:  CategoryFactual,
			expectedWebSearch: false,
		},
		{
			name:              "Mathematical - Calculate",
			question:          "Calcule a raiz quadrada de 144",
			expectedCategory:  CategoryMathematical,
			expectedWebSearch: false,
		},
		{
			name:              "Creative - Suggest",
			question:          "Sugira 5 nomes para um gato",
			expectedCategory:  CategoryCreative,
			expectedWebSearch: false,
		},
		{
			name:              "Simple - Default",
			question:          "Olá, tudo bem?",
			expectedCategory:  CategorySimple, // Should fall back to simple if no strong keywords
			expectedWebSearch: false,
		},
		{
			name:              "Word Boundary Check",
			question:          "Eu vi um noticiarista (should not match the keyword)",
			expectedCategory:  CategorySimple, // "noticiarista" contains "noticia" but is not in the list
			expectedWebSearch: false,
		},
		{
			name:              "Negative Keyword - Create News",
			question:          "Crie uma notícia falsa sobre aliens",
			expectedCategory:  CategoryCreative, // "Crie" is creative, "notícia" is web search. But "Crie" is negative for web search.
			expectedWebSearch: false,
		},
		{
			name:              "Semantic Match - Stemming & Accents",
			question:          "Tenho duvidas sobre isso", // "duvidas" (no accent, plural) should match "dúvida" (accent, singular)
			expectedCategory:  CategoryComplex,            // Assuming "dúvida" or similar is in complex keywords
			expectedWebSearch: false,
		},
		{
			name:              "Case Insensitivity",
			question:          "QUAIS AS NOTÍCIAS?",
			expectedCategory:  CategoryWebSearch,
			expectedWebSearch: true,
		},
		{
			name:              "Drink Suggestion - Portuguese",
			question:          "me dê uma sugestão de drinks",
			expectedCategory:  CategoryCreative,
			expectedWebSearch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Route(tt.question)
			if decision.Category != tt.expectedCategory {
				t.Errorf("Route(%q) Category = %v, want %v", tt.question, decision.Category, tt.expectedCategory)
			}
			if decision.WebSearch != tt.expectedWebSearch {
				t.Errorf("Route(%q) WebSearch = %v, want %v", tt.question, decision.WebSearch, tt.expectedWebSearch)
			}
		})
	}
}
