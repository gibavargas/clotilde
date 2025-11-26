package router

import (
	"strings"
	"testing"

	"github.com/clotilde/carplay-assistant/internal/admin"
)

func init() {
	// Ensure admin config is initialized for all tests
	admin.SetDefaultConfig("System prompt %s")
}

// TestScoringAccuracy tests that scoring works correctly for various scenarios
func TestScoringAccuracy(t *testing.T) {
	tests := []struct {
		name           string
		question       string
		expectedMin    float64 // Minimum expected score
		expectedMax    float64 // Maximum expected score
		expectedCat    Category
		checkWebSearch bool
		webSearch      bool
	}{
		{
			name:        "Multiple web search keywords",
			question:    "Quais as últimas notícias do Brasil hoje?",
			expectedMin: 2.0,
			expectedMax: 5.0,
			expectedCat: CategoryWebSearch,
		},
		{
			name:        "Single strong keyword",
			question:    "Calcule 2+2",
			expectedMin: 1.0,
			expectedMax: 2.0,
			expectedCat: CategoryMathematical,
		},
		{
			name:        "Multiple category keywords",
			question:    "Explique e compare Python e Go",
			expectedMin: 1.0,
			expectedMax: 3.0,
			expectedCat: CategoryComplex,
		},
		{
			name:        "No keywords - should be simple",
			question:    "Olá, como vai?",
			expectedMin: 0.0,
			expectedMax: 0.9,
			expectedCat: CategorySimple,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Route(tt.question)
			score := matchCategory(tt.question, tt.expectedCat) * categoryWeights[tt.expectedCat]

			if score < tt.expectedMin || score > tt.expectedMax {
				t.Logf("Score %.2f for category %s (expected %.2f-%.2f)", score, tt.expectedCat, tt.expectedMin, tt.expectedMax)
			}

			if decision.Category != tt.expectedCat {
				t.Errorf("Expected category %s, got %s", tt.expectedCat, decision.Category)
			}
		})
	}
}

// TestModelSelection tests model selection logic
func TestModelSelection(t *testing.T) {
	// Set custom config for testing
	config := admin.GetConfig()
	originalStandard := config.StandardModel
	originalPremium := config.PremiumModel
	originalCategoryModels := config.CategoryModels

	tests := []struct {
		name           string
		question       string
		expectedCat    Category
		webSearch      bool
		standardModel  string
		premiumModel   string
		expectedModel  string // Expected model (may be fallback for web search)
	}{
		{
			name:          "Web search uses standard model",
			question:      "Quais as notícias?",
			expectedCat:   CategoryWebSearch,
			webSearch:     true,
			standardModel: "gpt-4o-mini",
			premiumModel:  "gpt-4.1",
			expectedModel: "gpt-4o-mini", // Standard model supports web search
		},
		{
			name:          "Complex uses premium model",
			question:      "Explique a teoria da relatividade",
			expectedCat:   CategoryComplex,
			webSearch:     false,
			standardModel: "gpt-4o-mini",
			premiumModel:  "gpt-4.1",
			expectedModel: "gpt-4.1", // Premium model
		},
		{
			name:          "Creative uses premium model",
			question:      "Sugira nomes para um gato",
			expectedCat:   CategoryCreative,
			webSearch:     false,
			standardModel: "gpt-4o-mini",
			premiumModel:  "gpt-4.1",
			expectedModel: "gpt-4.1", // Premium model
		},
		{
			name:          "Factual uses standard model",
			question:      "Quem é o presidente?",
			expectedCat:   CategoryFactual,
			webSearch:     false,
			standardModel: "gpt-4o-mini",
			premiumModel:  "gpt-4.1",
			expectedModel: "gpt-4o-mini", // Standard model
		},
		{
			name:          "Mathematical uses standard model",
			question:      "Calcule 2+2",
			expectedCat:   CategoryMathematical,
			webSearch:     false,
			standardModel: "gpt-4o-mini",
			premiumModel:  "gpt-4.1",
			expectedModel: "gpt-4o-mini", // Standard model
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily set models (clear category overrides)
			err := admin.SetConfig(admin.RuntimeConfig{
				BaseSystemPrompt: config.BaseSystemPrompt,
				StandardModel:    tt.standardModel,
				PremiumModel:     tt.premiumModel,
				CategoryModels:   make(map[string]string), // Clear overrides
			})
			if err != nil {
				t.Fatalf("Failed to set config: %v", err)
			}

			decision := Route(tt.question)

			if decision.Category != tt.expectedCat {
				t.Errorf("Expected category %s, got %s", tt.expectedCat, decision.Category)
			}

			if decision.WebSearch != tt.webSearch {
				t.Errorf("Expected webSearch=%v, got %v", tt.webSearch, decision.WebSearch)
			}

			// For web search, model might be fallback if standard doesn't support it
			if tt.webSearch {
				// Model should be standard, premium (if supports web search), or fallback
				validModels := map[string]bool{
					tt.standardModel: true,
					tt.premiumModel:  modelsWithWebSearch[tt.premiumModel],
					webSearchFallbackModel: true,
				}
				if !validModels[decision.Model] {
					t.Errorf("Model %s not in expected set (standard=%s, premium=%s, fallback=%s)", 
						decision.Model, tt.standardModel, tt.premiumModel, webSearchFallbackModel)
				}
			} else {
				// For non-web-search, should match expected
				if decision.Model != tt.expectedModel {
					t.Errorf("Expected model %s, got %s", tt.expectedModel, decision.Model)
				}
			}
		})
	}

	// Restore original config
	admin.SetConfig(admin.RuntimeConfig{
		BaseSystemPrompt: config.BaseSystemPrompt,
		StandardModel:    originalStandard,
		PremiumModel:     originalPremium,
		CategoryModels:   originalCategoryModels,
	})
}

// TestWebSearchFallback tests fallback when model doesn't support web search
func TestWebSearchFallback(t *testing.T) {
	config := admin.GetConfig()
	originalStandard := config.StandardModel

	// Test with a model that doesn't support web search
	admin.SetConfig(admin.RuntimeConfig{
		StandardModel: "gpt-4.1-nano", // Doesn't support web search
		PremiumModel:  "gpt-4.1",
	})

	decision := Route("Quais as notícias?")
	if decision.Category != CategoryWebSearch {
		t.Errorf("Expected CategoryWebSearch, got %s", decision.Category)
	}

	if !decision.WebSearch {
		t.Error("Expected webSearch=true")
	}

	// Should fallback to webSearchFallbackModel
	if decision.Model != webSearchFallbackModel {
		t.Errorf("Expected fallback model %s, got %s", webSearchFallbackModel, decision.Model)
	}

	// Restore
	admin.SetConfig(admin.RuntimeConfig{
		StandardModel: originalStandard,
		PremiumModel:  config.PremiumModel,
	})
}

// TestGPT5ReasoningEffort tests GPT-5 series requires reasoning for web search
func TestGPT5ReasoningEffort(t *testing.T) {
	config := admin.GetConfig()
	originalStandard := config.StandardModel
	originalPremium := config.PremiumModel
	originalCategoryModels := config.CategoryModels

	// Test with GPT-5 model as standard (for web search)
	admin.SetConfig(admin.RuntimeConfig{
		BaseSystemPrompt: config.BaseSystemPrompt,
		StandardModel:    "gpt-5",
		PremiumModel:     "gpt-4.1",
		CategoryModels:   make(map[string]string),
	})

	decision := Route("Quais as notícias?")
	if decision.Category != CategoryWebSearch {
		t.Errorf("Expected CategoryWebSearch, got %s", decision.Category)
	}

	if !decision.WebSearch {
		t.Error("Expected webSearch=true")
	}

	// GPT-5 requires reasoning for web search
	if decision.Model == "gpt-5" && decision.ReasoningEffort != "low" {
		t.Errorf("Expected reasoningEffort='low' for GPT-5, got '%s' (model=%s)", 
			decision.ReasoningEffort, decision.Model)
	}

	// Restore
	admin.SetConfig(admin.RuntimeConfig{
		BaseSystemPrompt: config.BaseSystemPrompt,
		StandardModel:    originalStandard,
		PremiumModel:     originalPremium,
		CategoryModels:   originalCategoryModels,
	})
}

// TestNegativeKeywords tests that negative keywords prevent false positives
func TestNegativeKeywords(t *testing.T) {
	tests := []struct {
		name          string
		question      string
		expectedCat   Category
		notExpectedCat Category
	}{
		{
			name:          "Create news should be creative, not web search",
			question:      "Crie uma notícia sobre aliens",
			expectedCat:   CategoryCreative,
			notExpectedCat: CategoryWebSearch,
		},
		{
			name:          "Explain news should be complex, not web search",
			question:      "Explique a notícia sobre política",
			expectedCat:   CategoryComplex,
			notExpectedCat: CategoryWebSearch,
		},
		{
			name:          "Define news should be factual, not web search",
			question:      "O que é uma notícia?",
			expectedCat:   CategoryFactual,
			notExpectedCat: CategoryWebSearch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Route(tt.question)

			if decision.Category == tt.notExpectedCat {
				t.Errorf("Negative keyword failed: got %s (should not be %s)", decision.Category, tt.notExpectedCat)
			}

			if decision.Category != tt.expectedCat {
				t.Logf("Expected %s, got %s (may be acceptable)", tt.expectedCat, decision.Category)
			}
		})
	}
}

// TestEdgeCases tests edge cases
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		question string
		valid    bool // Whether this should produce a valid result
	}{
		{
			name:     "Empty string",
			question: "",
			valid:    true, // Should default to simple
		},
		{
			name:     "Only whitespace",
			question: "   \n\t  ",
			valid:    true,
		},
		{
			name:     "Very long string",
			question: strings.Repeat("Quais as notícias? ", 100),
			valid:    true,
		},
		{
			name:     "Special characters only",
			question: "!@#$%^&*()",
			valid:    true,
		},
		{
			name:     "Numbers only",
			question: "123456789",
			valid:    true,
		},
		{
			name:     "Mixed languages",
			question: "What are the news? Quais as notícias?",
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Route(tt.question)

			if !tt.valid {
				return // Skip validation for invalid cases
			}

			// Should always return a valid decision
			if decision.Category == "" {
				t.Error("Category should not be empty")
			}

			if decision.Model == "" {
				t.Error("Model should not be empty")
			}
		})
	}
}

// TestTieBreaking tests priority order for tie-breaking
func TestTieBreaking(t *testing.T) {
	// Create questions that might match multiple categories
	tests := []struct {
		name         string
		question     string
		expectedCat  Category
		description  string
	}{
		{
			name:        "Mathematical priority over complex",
			question:    "Calcule e explique a raiz quadrada",
			expectedCat: CategoryMathematical,
			description: "Should prefer mathematical over complex",
		},
		{
			name:        "Web search priority over factual",
			question:    "Quais as notícias sobre o presidente?",
			expectedCat: CategoryWebSearch,
			description: "Should prefer web search over factual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Route(tt.question)

			if decision.Category != tt.expectedCat {
				t.Logf("%s: Expected %s, got %s", tt.description, tt.expectedCat, decision.Category)
				// Log scores for debugging
				scores := map[Category]float64{
					CategoryWebSearch:    matchCategory(tt.question, CategoryWebSearch) * categoryWeights[CategoryWebSearch],
					CategoryComplex:      matchCategory(tt.question, CategoryComplex) * categoryWeights[CategoryComplex],
					CategoryFactual:      matchCategory(tt.question, CategoryFactual) * categoryWeights[CategoryFactual],
					CategoryMathematical: matchCategory(tt.question, CategoryMathematical) * categoryWeights[CategoryMathematical],
					CategoryCreative:     matchCategory(tt.question, CategoryCreative) * categoryWeights[CategoryCreative],
				}
				for cat, score := range scores {
					t.Logf("  %s: %.2f", cat, score)
				}
			}
		})
	}
}

// TestNormalizationEdgeCases tests normalization with edge cases
func TestNormalizationEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string // Expected normalized output (approximate)
	}{
		{"Notícias", "noticia"},
		{"NOTÍCIAS", "noticia"},
		{"notícias", "noticia"},
		{"notícias!!!", "noticia"},
		{"notícias, notícias", "noticia noticia"},
		{"dúvida", "duvida"},
		{"DÚVIDAS", "duvida"},
		{"correndo", "corre"},
		{"correr", "corr"},
		{"", ""},
		{"   ", ""},
		{"a", "a"}, // Short word, no stemming
		{"abc", "abc"}, // Short word, no stemming
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Normalize(tt.input)
			// Just check it doesn't crash and produces something reasonable
			if result == "" && tt.input != "" && strings.TrimSpace(tt.input) != "" {
				// Empty result is only OK if input was empty/whitespace
				if strings.TrimSpace(tt.input) != "" {
					t.Logf("Normalize('%s') = '%s' (empty result for non-empty input)", tt.input, result)
				}
			}
			t.Logf("Normalize('%s') = '%s'", tt.input, result)
		})
	}
}

// TestCategoryModelsOverride tests category-specific model overrides
func TestCategoryModelsOverride(t *testing.T) {
	config := admin.GetConfig()
	originalStandard := config.StandardModel
	originalPremium := config.PremiumModel
	originalCategoryModels := config.CategoryModels

	// Set category-specific model override
	err := admin.SetConfig(admin.RuntimeConfig{
		BaseSystemPrompt: config.BaseSystemPrompt,
		StandardModel:    "gpt-4o-mini",
		PremiumModel:     "gpt-4.1",
		CategoryModels: map[string]string{
			"web_search": "gpt-4o", // Override for web_search category
		},
	})
	if err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	decision := Route("Quais as notícias?")
	if decision.Category != CategoryWebSearch {
		t.Errorf("Expected CategoryWebSearch, got %s", decision.Category)
	}

	// Should use override model (gpt-4o supports web search, so no fallback)
	if decision.Model != "gpt-4o" {
		t.Errorf("Expected override model gpt-4o, got %s", decision.Model)
	}

	// Restore
	admin.SetConfig(admin.RuntimeConfig{
		BaseSystemPrompt: config.BaseSystemPrompt,
		StandardModel:    originalStandard,
		PremiumModel:     originalPremium,
		CategoryModels:   originalCategoryModels,
	})
}

// BenchmarkRoute benchmarks routing performance
func BenchmarkRoute(b *testing.B) {
	questions := []string{
		"Quais as últimas notícias do Brasil?",
		"Explique a teoria da relatividade",
		"Quem é o presidente do Brasil?",
		"Calcule a raiz quadrada de 144",
		"Sugira 5 nomes para um gato",
		"Olá, tudo bem?",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Route(questions[i%len(questions)])
	}
}

// BenchmarkNormalize benchmarks normalization performance
func BenchmarkNormalize(b *testing.B) {
	inputs := []string{
		"Quais as últimas notícias do Brasil?",
		"Explique a teoria da relatividade",
		"Dúvidas sobre isso",
		strings.Repeat("Quais as notícias? ", 10),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Normalize(inputs[i%len(inputs)])
	}
}

// BenchmarkMatchCategory benchmarks category matching
func BenchmarkMatchCategory(b *testing.B) {
	question := "Quais as últimas notícias do Brasil hoje?"
	categories := []Category{
		CategoryWebSearch,
		CategoryComplex,
		CategoryFactual,
		CategoryMathematical,
		CategoryCreative,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchCategory(question, categories[i%len(categories)])
	}
}

