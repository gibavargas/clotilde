package router

// Comprehensive Portuguese keywords for routing (Expanded to ~10,000+ total)
// This file acts as the central registry. Keywords are populated via init() functions in other files.

// Web Search Required keywords
var webSearchKeywords = []string{}

// Complex Analysis keywords
var complexKeywords = []string{}

// Factual Lookup keywords
var factualKeywords = []string{}

// Mathematical/Calculation keywords
var mathematicalKeywords = []string{}

// Creative/Open-ended keywords
var creativeKeywords = []string{}

// Negative keywords to filter false positives
var negativeKeywords = map[string][]string{
	"web_search": {
		"crie", "imagine", "invente", "escreva", "redija", "traduza", "explique", "defina",
		"o que é", "significado", "conceito", "resuma", "sintetize", "analise", "compare",
	},
	"factual": {
		"crie", "imagine", "invente", "sugira", "recomende", "opinião",
	},
}
